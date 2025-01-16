package fileserver

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"github.com/pquerna/otp/totp"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sophuwu.site/cdn/config"
	"strings"
)

type DirEntry struct {
	Name     string
	FullName string
	Size     int
	IsDir    bool
}

func (d DirEntry) Si() string {
	if d.IsDir {
		return ""
	}
	f := float64(d.Size)
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

var Temp *template.Template

func FillTemp(w io.Writer, path string, items []DirEntry) error {
	var data = TemplateData{Path: path, Dirs: []DirEntry{}, Items: []DirEntry{}}
	for _, item := range items {
		item.FullName = filepath.Join(path, item.Name)
		if item.IsDir {
			data.Dirs = append(data.Dirs, item)
		} else {
			data.Items = append(data.Items, item)
		}
	}
	return Temp.ExecuteTemplate(w, "index", data)
}

func FillUpload(w io.Writer, path string) error {
	return Temp.ExecuteTemplate(w, "index", map[string]string{
		"Upload": path,
	})
}

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
	_ = Temp.ExecuteTemplate(w, "index", map[string]string{
		"Error": fmt.Sprintf("%d: %s", code, HttpCodes[code]),
	})
	return true
}

func init() {
	Temp = template.New("index")
	Temp.Parse(`{{ define "index" }}
<!DOCTYPE html>
<html>
<head>
{{ if .Path }}
<title>{{ .Path }}</title>
{{ else }}
{{ if .Upload }}
<title>Upload</title>
{{ else }}
<title>Error</title>
{{ end }}
{{ end }}
<style>
:root {--bord: #ccc;--fg: #fff;}
body {width: calc(100% - 2ch);margin: auto auto auto auto ;max-width: 800px;background: #262833;color: var(--fg);font-family: sans-serif;}
.trees {width: 100%;display: flex;flex-direction: column;padding: 0;margin: auto auto;border: 1px solid var(--bord);border-radius: 1ch;overflow: hidden;}
.trees a {display: contents;text-decoration: none;}
.filelabel {padding: 8px;font-size: 1rem;width: auto;margin: 0;display: grid;grid-template-columns: auto auto;grid-gap: 1ch;justify-content: space-between;align-items: center;border-radius: 0;background: transparent;}
.trees > a > *  {border-bottom: 1px solid var(--bord);background: #1c1e26;}
.trees > a > *:hover {background: #2c2e46;}
.trees > a:last-child > *  {border-bottom: none;}
a {color: var(--fg);text-decoration: none;}
.filelabel > :last-child {text-align: right;}
</style>
</head>
<body>
{{ if .Path }}
<h1>Index of: {{ .Path }}</h1>
<div class="trees">
{{ range .Dirs }}
<a href="{{ .FullName }}"><div class="filelabel"><span>{{ .Name }}</span><span>{{ .Si }}</span></div></a>
{{ end }}
{{ range .Items }}
<a href="{{ .FullName }}"><div class="filelabel"><span>{{ .Name }}</span><span>{{ .Si }}</span></div></a>
{{ end }}
</div>
{{ else }}
{{ if .Error }}
<h1>{{ .Error }}</h1>
{{ else }}
{{ if .Upload }}
<h1>Upload</h1>
<form class="trees" enctype="multipart/form-data" action="{{ .Upload }}" method="post">
	<div class="filelabel"><span>Path:</span><input type="text" name="path" /></div>
	<div class="filelabel"><span>File:</span><input type="file" name="myFile" /></div>
	<div class="filelabel"><span>Username:</span><input type="text" name="username" /></div>
	<div class="filelabel"><span>Password:</span><input type="password" name="password" /></div>
	<div class="filelabel"><span>OTP:</span><input type="text" name="otp" /></div>
	<div class="filelabel"><span></span><input type="submit" value="Upload" /></div>
</form>
{{ end }}
{{ end }}
{{ end }}
</body>
</html>
{{ end }}`)
}

func Handler(prefix string, dir func(string) ([]byte, []DirEntry, error)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, items, err := dir(strings.TrimPrefix(r.URL.Path, prefix))
		if FillError(w, err, 404) {
			return
		}
		w.WriteHeader(http.StatusOK)
		if data != nil {
			w.Write(data)
			return
		}
		items = append([]DirEntry{{
			Name:     "../",
			FullName: filepath.Dir(strings.TrimSuffix(r.URL.Path, "/")),
			Size:     0,
			IsDir:    true,
		}}, items...)
		err = FillTemp(w, r.URL.Path, items)
		if FillError(w, err, 500) {
			return
		}
	})
}

func Handle(prefix string, dir func(string) ([]byte, []DirEntry, error)) {
	http.Handle(prefix, Handler(prefix, dir))
}

func VerifyOtp(p, o string) bool {
	b, err := os.ReadFile(config.OtpPath)
	if err != nil {
		return false
	}
	s := string(b)
	s = strings.Split(s, "\n")[0]
	b, err = base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return false
	}
	b = b[:32]
	bb := sha256.Sum256([]byte(p))
	for i := 0; i < len(b); i++ {
		b[i] = b[i] ^ bb[i]
	}
	s = string(b)
	return totp.Validate(o, s)
}
func VerifyBasicAuth(u, p string) bool {
	b, err := os.ReadFile(config.OtpPath)
	if err != nil {
		return false
	}
	ss := strings.Split(string(b), "\n")
	if len(ss) < 2 {
		return false
	}
	b, err = base64.StdEncoding.DecodeString(strings.TrimSpace(ss[1]))
	if err != nil {
		return false
	}
	bb := sha256.Sum256([]byte(u + ";" + p))
	return subtle.ConstantTimeCompare(b, bb[:]) == 1
}

func Authenticate(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, apass, authOK := r.BasicAuth()
		if !authOK || !VerifyBasicAuth(user, apass) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func UpHandler(prefix string, save func(string, []byte) error) http.Handler {
	return Authenticate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if r.Method == "GET" {
			err := FillUpload(w, prefix)
			if FillError(w, err, 500) {
				return
			}
			return
		}
		if r.Method == "POST" {
			r.ParseMultipartForm(256 * 1024 * 1024)
			path := r.Form.Get("path")

			username := r.Form.Get("username")
			password := r.Form.Get("password")
			otp := r.Form.Get("otp")
			if !VerifyOtp(username+";"+password, otp) {
				FillError(w, fmt.Errorf("unauthorized"), 401)
				return
			}

			file, _, err := r.FormFile("myFile")
			if FillError(w, err, 400) {
				return
			}
			defer file.Close()
			data, err := io.ReadAll(file)
			if FillError(w, err, 500) {
				return
			}
			err = save(path, data)
			if FillError(w, err, 500) {
				return
			}
			http.Redirect(w, r, strings.ToLower(prefix)+path, http.StatusFound)
			return
		}
		FillError(w, fmt.Errorf("method not allowed"), 405)
	}))
}

func UpHandle(prefix string, save func(string, []byte) error) {
	http.Handle(prefix, UpHandler(prefix, save))
}
