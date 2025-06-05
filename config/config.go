package config

import (
	"fmt"
	"os"
	"strings"
)

var DbPath string
var HttpDir string
var Port string
var Addr string

func checkAbs(path string) bool {
	if len(path) < 2 || path[0] != '/' {
		return false
	}
	return true
}

func checkDir(path string) bool {
	if st, err := os.Stat(path); err == nil && st.IsDir() {
		return true
	}
	return false
}

func checkAbsDir(path string) bool {
	return checkAbs(path) && checkDir(path)
}

func init() {
	DbPath = strings.TrimSpace(os.Getenv("DB_PATH"))
	HttpDir = strings.TrimSpace(os.Getenv("HTTP_DIR"))
	if !checkAbs(DbPath) || !checkAbsDir(HttpDir) {
		fmt.Println("Please set the environment variables DB_PATH and HTTP_DIR to absolute paths.")
		os.Exit(1)
	}
	Port = strings.TrimSpace(os.Getenv("PORT"))
	if Port == "" {
		Port = "8080"
	}
	Addr = strings.TrimSpace(os.Getenv("ADDR"))
	if Addr == "" {
		Addr = "127.0.0.1"
	}

}
