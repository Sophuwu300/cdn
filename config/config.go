package config

import (
	"os"
	"strings"
)

var OtpPath string
var DbPath string
var HttpDir string
var Port string
var Addr string

func Get() {
	OtpPath = strings.TrimSpace(os.Getenv("OTP_PATH"))
	DbPath = strings.TrimSpace(os.Getenv("DB_PATH"))
	HttpDir = strings.TrimSpace(os.Getenv("HTTP_DIR"))
	if HttpDir == "" {
		HttpDir = "."
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
