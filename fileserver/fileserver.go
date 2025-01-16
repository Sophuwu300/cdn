package fileserver

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
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
		if item.IsDir {
			data.Dirs = append(data.Dirs, item)
		} else {
			data.Items = append(data.Items, item)
		}
	}
	return Temp.ExecuteTemplate(w, "index", data)
}

func init() {
	Temp = template.New("index")
	Temp.Parse(`{{ define "index" }}
<!DOCTYPE html>
<html>
<head>
<title>{{ .Path }}</title>
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
<h1>Index of: {{.Path}}</h1>
<div class="trees">
{{ range .Dirs }}
<a href="{{ .FullName }}"><div class="filelabel"><span>{{ .Name }}</span><span>{{ .Si }}</span></div></a>
{{ end }}
{{ range .Items }}
<a href="{{ .FullName }}"><div class="filelabel"><span>{{ .Name }}</span><span>{{ .Si }}</span></div></a>
{{ end }}
</div>
</body>
</html>
{{ end }}`)
}

func Handler(prefix string, dir func(string) ([]byte, []DirEntry, error)) (string, http.Handler) {
	return prefix, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, items, err := dir(strings.TrimPrefix(r.URL.Path, prefix))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
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
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func Handle(prefix string, dir func(string) ([]byte, []DirEntry, error)) {
	http.Handle(Handler(prefix, dir))
}
