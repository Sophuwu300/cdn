package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"git.sophuwu.com/cdn/config"
	"git.sophuwu.com/cdn/imgconv"
	"github.com/asdine/storm/v3"
	"go.etcd.io/bbolt"
	"golang.org/x/sys/unix"
	"os/signal"
	pathlib "path"

	"io/fs"
	"time"

	_ "embed"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed html/*
var embedHtml embed.FS

type TimeStr struct {
	Full  string
	Short string
}

type DirEntry struct {
	Icon    string
	Name    string
	Size    string
	SizeN   int64
	Url     string
	ModUnix int64
	Mod     TimeStr
}

func FmtTime(t time.Time, today time.Time) TimeStr {
	var pt TimeStr
	pt.Full = t.Format(time.RFC822Z)
	d := today.Sub(t)
	if d < 5*24*time.Hour {
		pt.Short = t.Format("Mon, 15:04")
	} else {
		pt.Short = t.Format("2006-01-02")
	}
	return pt
}

func Si(d int64) string {
	f := float64(d)
	i := 0
	for f > 1024 {
		f /= 1024
		i++
	}
	s := fmt.Sprintf("%.2f", f)
	s = strings.TrimRight(s, ".0")
	return fmt.Sprintf("%s %cB", s, " KMGTPEZY"[i])
}

type TemplateData struct {
	Path  string
	Dirs  []DirEntry
	Items []DirEntry
}

func (t *TemplateData) add(a DirEntry, size int64, dir bool) {
	if dir {
		a.SizeN = func() int64 {
			n, _ := os.ReadDir(filepath.Join(config.HttpDir, a.Url))
			return int64(len(n))
		}()
		a.Url += "/"
		a.Size = fmt.Sprintf("%d items", a.SizeN)
		a.Icon = "F"
		t.Dirs = append(t.Dirs, a)
	} else {
		a.Icon = "f"
		a.SizeN = size
		a.Size = Si(size)
		t.Items = append(t.Items, a)
	}
}
func (t *TemplateData) sortNewest() {
	for _, tt := range []*[]DirEntry{&t.Items, &t.Dirs} {
		for i := 0; i < len(*tt); i++ {
			for j := i + 1; j < len(*tt); j++ {
				if (*tt)[i].ModUnix < (*tt)[j].ModUnix {
					(*tt)[i], (*tt)[j] = (*tt)[j], (*tt)[i]
				}
			}
		}
	}
}

var Temp *template.Template

type HttpCode struct {
	Code    int
	Name    string
	Message string
}

var HttpCodes = map[int]HttpCode{
	404: HttpCode{404, "Not Found", "The file you requested was not found on this server."},
	500: HttpCode{500, "Internal Server Error", "The server encountered an internal error and was unable to complete your request."},
	403: HttpCode{403, "Forbidden", "The server understood the request, but is refusing to fulfill it."},
	401: HttpCode{401, "Unauthorized", "You lack the necessary permissions to access this resource."},
	400: HttpCode{400, "Bad Request", "The server could not understand the request as it was malformed."},
}

func FillError(w http.ResponseWriter, err error, code int) bool {
	if err == nil {
		return false
	}
	fmt.Fprintf(os.Stderr, "error: %s\n", err)
	w.WriteHeader(code)
	ht, ok := HttpCodes[code]
	if !ok {
		ht = HttpCodes[500]
	}
	_ = Temp.ExecuteTemplate(w, "error", ht)
	return true
}

type ImgIcon struct {
	ImgPath string `storm:"id"`
	ModTime string
	PngData []byte
}

func CleanPath(d http.Dir, name string) (string, error) {
	path := pathlib.Clean("/" + name)[1:]
	if path == "" {
		return "", errors.New("http: empty file path")
	}
	path, err := filepath.Localize(path)
	if err != nil {
		return "", errors.New("http: invalid or unsafe file path")
	}
	dir := string(d)
	if !filepath.IsAbs(dir) {
		return "", errors.New("http: invalid or unsafe file path")
	}
	return filepath.Join(dir, path), nil
}

func CustomFileServer(root http.Dir) http.Handler {
	iconFunc := func(w http.ResponseWriter, r *http.Request) {
		qq := r.URL.Query()
		var icon ImgIcon
		if qq.Get("icon") == "" || qq.Get("mod") == "" {
			FillError(w, fmt.Errorf("icon or mod not found"), 400)
			return
		}
		err := DB.One("ImgPath", qq.Get("icon"), &icon)
		if err != nil || icon.ModTime != qq.Get("mod") {
			fn := DB.Update
			if errors.Is(err, storm.ErrNotFound) {
				fn = DB.Save
			}
			icon.ImgPath = qq.Get("icon")
			icon.ModTime = qq.Get("mod")
			var path string
			if path, err = CleanPath(root, qq.Get("icon")); err != nil {
				FillError(w, fmt.Errorf("icon or mod not found"), 400)
				return
			}
			icon.PngData, err = imgconv.Media2Icon(path)
			if FillError(w, err, 404) {
				return
			}
			err = fn(&icon)
			if FillError(w, err, 500) {
				return
			}
		}
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(200)
		w.Write(icon.PngData)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Has("icon") {
			iconFunc(w, r)
			return
		}
		upath := r.URL.Path
		f, err := root.Open(upath)
		if FillError(w, err, 404) {
			return
		}
		defer f.Close()
		var fi fs.FileInfo
		fi, err = f.Stat()
		if FillError(w, err, 500) {
			return
		}
		if fi.IsDir() {
			var fi []fs.FileInfo
			fi, err = f.Readdir(0)
			if FillError(w, err, 500) {
				return
			}
			t := TemplateData{Path: upath, Dirs: []DirEntry{}, Items: []DirEntry{}}
			today := time.Now().UTC()
			for _, d := range fi {
				t.add(DirEntry{
					Name:    d.Name(),
					Url:     filepath.Join(upath, d.Name()),
					ModUnix: d.ModTime().Local().Unix(),
					Mod:     FmtTime(d.ModTime().UTC(), today),
				}, d.Size(), d.IsDir())
			}
			t.sortNewest()
			Temp.ExecuteTemplate(w, "index", t)
			return
		}
		http.FileServer(root).ServeHTTP(w, r)
	})
}

var DB *storm.DB

func main() {
	Temp = template.Must(template.ParseFS(embedHtml, "html/*"))
	db, err := storm.Open(config.DbPath, storm.BoltOptions(0600, &bbolt.Options{Timeout: 1 * time.Second}))
	if err != nil {
		fmt.Println("Failed to open database:", err)
		os.Exit(1)
	}
	DB = db
	server := http.Server{
		Addr:    config.Addr + ":" + config.Port,
		Handler: CustomFileServer(http.Dir(config.HttpDir)),
	}
	fmt.Printf("starting cdn server with pid: %d\n\tlistening on %s\n\tserving directory: %s\n", os.Getpid(), server.Addr, config.HttpDir)

	closeDB := func() {
		err1 := DB.Close()
		if err1 != nil {
			fmt.Fprintf(os.Stderr, "error closing database: %s\n", err1)
			os.Exit(1)
		}
		fmt.Println("Database closed safely")
	}

	go func() {
		if err1 := server.ListenAndServe(); err1 != nil && !errors.Is(err1, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "error: %s\n", err1)
			closeDB()
			os.Exit(1)
		}
	}()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, unix.SIGINT, unix.SIGTERM)
	fmt.Printf("got signal %v, stopping\n", <-sigChan)
	server.Shutdown(context.Background())
	closeDB()
}
