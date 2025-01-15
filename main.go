package main

import (
	"fmt"
	bolt "go.etcd.io/bbolt"
	"path/filepath"
	"strings"
)

// var db *bolt.DB
//
// func init() {
// 	var err error
// 	db, err = bolt.Open("build/my.db", 0600, nil)
// 	if err != nil {
// 		log.Fatalln(err)
// 	}
// }

// FileSystem represents a file system in a database
type FileSystem struct {
	db *bolt.DB
}

// OpenDB opens or creates the database
func OpenDB(dbPath string) (*FileSystem, error) {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("root"))
		return err
	})
	return &FileSystem{db: db}, err
}

type DirEntry struct {
	Name  string
	Size  int
	IsDir bool
}

// GetEntry retrieves the content of a file or the list of items in a directory
func (fs *FileSystem) GetEntry(path string) ([]byte, []DirEntry, error) {
	path = strings.Trim(path, "/")
	paths := strings.Split(filepath.Dir(path), "/")
	path = filepath.Base(path)
	var items []DirEntry
	var item DirEntry
	var data []byte
	return data, items, fs.db.View(func(tx *bolt.Tx) error {
		currentBucket := tx.Bucket([]byte("root"))
		if currentBucket == nil {
			return fmt.Errorf("root bucket does not exist")
		}

		for _, dir := range paths {
			if dir == "" || dir == "." {
				continue
			}
			currentBucket = currentBucket.Bucket([]byte(dir))
			if currentBucket == nil {
				return fmt.Errorf("directory %s does not exist", dir)
			}
		}
		if path != "" && path != "." {
			data = currentBucket.Get([]byte(path))
			if data != nil {
				return nil
			}
			currentBucket = currentBucket.Bucket([]byte(path))
		}
		if currentBucket != nil {
			return currentBucket.ForEach(func(k, v []byte) error {
				item = DirEntry{
					Name:  string(k),
					Size:  len(v),
					IsDir: false,
				}
				if item.Size == 0 && currentBucket.Bucket(k) != nil {
					item.Name += "/"
					item.IsDir = true
				}
				items = append(items, item)
				return nil
			})
		}
		return fmt.Errorf("entry %s does not exist", path)
	})
}

// PutFile stores the content of a file, creating directories as needed
func (fs *FileSystem) PutFile(path string, data []byte) error {
	path = strings.Trim(path, "/")
	paths := strings.Split(filepath.Dir(path), "/")
	path = filepath.Base(path)
	if len(path) < 1 || strings.HasPrefix(path, ".") {
		return fmt.Errorf("invalid file name")
	}
	var err error
	return fs.db.Update(func(tx *bolt.Tx) error {
		currentBucket := tx.Bucket([]byte("root"))
		if currentBucket == nil {
			return fmt.Errorf("root bucket does not exist")
		}

		for _, dir := range paths {
			if dir == "" || dir == "." {
				continue
			}
			if currentBucket.Get([]byte(dir)) != nil {
				return fmt.Errorf("directory %s is a file", dir)
			}
			currentBucket, err = currentBucket.CreateBucketIfNotExists([]byte(dir))
			if err != nil || currentBucket == nil {
				return fmt.Errorf("directory %s does not exist", dir)
			}
		}
		if currentBucket.Bucket([]byte(path)) != nil {
			return fmt.Errorf("file %s is a directory", path)
		}
		return currentBucket.Put([]byte(path), data)
	})
}

// Close the database
func (fs *FileSystem) Close() error {
	return fs.db.Close()
}

func main() {
	fs, err := OpenDB("build/my.db")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer fs.Close()

	// err = fs.PutFile("index.txt", []byte("Hello"))
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	data, items, err := fs.GetEntry("/")
	if err != nil {
		fmt.Println(err)
		return
	}
	if data != nil {
		fmt.Println(string(data))
		return
	}
	for _, item := range items {
		fmt.Println(item)
	}

}
