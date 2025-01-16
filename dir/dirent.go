package dir

import (
	"errors"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"
	"sophuwu.site/cdn/fileserver"
)

func Open(path string) func(string) ([]byte, []fileserver.DirEntry, error) {
	d := Dir{http.Dir(path)}
	return d.GetEntry
}

type Dir struct {
	H http.FileSystem
}

type DirEntry = fileserver.DirEntry

func (d *Dir) GetEntry(path string) (data []byte, items []DirEntry, err error) {
	var f http.File
	f, err = d.H.Open(path)
	if err != nil {
		return
	}
	var fi fs.FileInfo
	fi, err = f.Stat()
	if err != nil {
		return
	}
	if fi.IsDir() {
		var de []fs.FileInfo
		de, err = f.Readdir(0)
		if err != nil {
			return
		}
		items = []DirEntry{}
		for _, d := range de {
			items = append(items, DirEntry{
				Name: d.Name() + func() string {
					if d.IsDir() {
						return "/"
					}
					return ""
				}(),
				FullName: filepath.Join(path, d.Name()),
				Size:     int(d.Size()),
				IsDir:    d.IsDir(),
			})
		}
		return
	}
	if fi.Mode().IsRegular() {
		data, err = io.ReadAll(f)
		return
	}
	err = errors.New("not a regular file")
	return
}
