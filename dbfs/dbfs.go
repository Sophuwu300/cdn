package dbfs

import (
	"fmt"
	bolt "go.etcd.io/bbolt"
	"path/filepath"
	"sophuwu.site/cdn/fileserver"
	"strings"
)

type DirEntry = fileserver.DirEntry

// DBFS represents a file system in a database
type DBFS struct {
	db *bolt.DB
}

// OpenDB opens or creates the database
func OpenDB(dbPath string) (*DBFS, error) {
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte("root"))
		return err
	})
	return &DBFS{db: db}, err
}

// GetEntry retrieves the content of a file or the list of items in a directory
func (fs *DBFS) GetEntry(path string) ([]byte, []DirEntry, error) {
	path = strings.Trim(path, "/")
	fullpath := path
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
				if len(k) < 1 || strings.HasPrefix(string(k), ".") {
					return nil
				}
				item = DirEntry{
					Name:     string(k),
					FullName: filepath.Join(fullpath, string(k)),
					Size:     len(v),
					IsDir:    false,
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
func (fs *DBFS) PutFile(path string, data []byte) error {
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
func (fs *DBFS) Close() error {
	return fs.db.Close()
}
