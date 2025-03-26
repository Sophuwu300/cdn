package main

import (
	"embed"
	"errors"
	"fmt"
	"git.sophuwu.com/cdn/config"
	"git.sophuwu.com/cdn/imgconv"
	"github.com/asdine/storm/v3"
	"go.etcd.io/bbolt"

	"io/fs"
	"time"

	_ "embed"
	"html/template"
	"io"
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
	Icon string
	Name string
	Size string
	Url  string
	mod  int64
	Mod  TimeStr
}

func DateToInt(t time.Time) int { // bit size: y 12, mon 4, day 5
	return t.Year()<<(12+4+5) | int(t.Month())<<(4+5) | t.Day()
}

func FmtTime(t time.Time, today int) TimeStr {
	var pt TimeStr
	pt.Full = t.Format("Mon 02 Jan 2006 15:04")
	if DateToInt(t) == today {
		pt.Short = "today " + t.Format("15:04")
	} else {
		pt.Short = t.Format("02 Jan 2006")
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
		a.Size = func() string {
			n, e := os.ReadDir(filepath.Join(config.HttpDir, a.Url))
			if e == nil {
				return fmt.Sprintf("%d items", len(n))
			}
			return "0 items"
		}()
		a.Icon = "F"
		t.Dirs = append(t.Dirs, a)
	} else {
		a.Icon = "f"
		a.Size = Si(size)
		t.Items = append(t.Items, a)
	}
}
func (t *TemplateData) sortNewest() {
	for _, tt := range []*[]DirEntry{&t.Items, &t.Dirs} {
		for i := 0; i < len(*tt); i++ {
			for j := i + 1; j < len(*tt); j++ {
				if (*tt)[i].mod < (*tt)[j].mod {
					(*tt)[i], (*tt)[j] = (*tt)[j], (*tt)[i]
				}
			}
		}
	}
}

var Temp *template.Template

var HttpCodes = map[int]string{
	404: "Not Found",
	500: "Internal Server Error",
	403: "Forbidden",
	401: "Unauthorized",
	400: "Bad Request",
	200: "OK",
}

func FillError(w io.Writer, err error, code int) bool {
	if err == nil {
		return false
	}
	fmt.Fprintf(os.Stderr, "error: %s\n", err)
	_ = Temp.ExecuteTemplate(w, "index", map[string]string{
		"Error": fmt.Sprintf("%d: %s", code, HttpCodes[code]),
	})
	return true
}

type ImgIcon struct {
	ImgPath string `storm:"id"`
	ModTime string
	PngData []byte
}

func customFileServer(root http.Dir) http.Handler {
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
			var f http.File
			f, err = root.Open(icon.ImgPath)
			if FillError(w, err, 404) {
				return
			}
			var fi fs.FileInfo
			fi, err = f.Stat()
			if err == nil && fi.IsDir() {
				err = fmt.Errorf("icon is dir")
			}
			if FillError(w, err, 400) {
				f.Close()
				return
			}
			icon.PngData, err = imgconv.Media2Icon(icon.ImgPath, f)
			f.Close()
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
			today := DateToInt(time.Now())
			for _, d := range fi {
				t.add(DirEntry{
					Name: d.Name(),
					Url:  filepath.Join(upath, d.Name()),
					mod:  d.ModTime().Unix(),
					Mod:  FmtTime(d.ModTime(), today),
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
	// Fs := os.DirFS(Config.HTTPDir)

	http.Handle("/", customFileServer(http.Dir(config.HttpDir)))
	// http.Handle("/", http.StripPrefix("/", FileServer(http.Dir(Config.HTTPDir))))

	http.ListenAndServe(config.Addr+":"+config.Port, nil)

}

func init() {
	Temp = template.Must(template.ParseFS(embedHtml, "html/*"))
	config.Get()
	db, err := storm.Open(config.DbPath, storm.BoltOptions(0600, &bbolt.Options{Timeout: 1 * time.Second}))
	if err != nil {
		fmt.Println("Failed to open database:", err)
		os.Exit(1)
	}
	db.Init(&ImgIcon{})
	DB = db
}
