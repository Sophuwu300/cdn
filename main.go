package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sophuwu.site/cdn/config"
	"sophuwu.site/cdn/dbfs"
	"sophuwu.site/cdn/dir"
	"sophuwu.site/cdn/fileserver"
)

func main() {
	config.Get()
	fmt.Println(config.DbPath, config.OtpPath, config.Port, config.Addr)

	var err error
	var db *dbfs.DBFS

	if config.DbPath != "" {
		db, err = dbfs.OpenDB(config.DbPath)
		if err != nil {
			fmt.Println(err)
			return
		}
		fileserver.Handle("/x/", db.GetEntry)
		if config.OtpPath != "" {
			fileserver.UpHandle("/X/", db.PutFile)
		}
	}
	fileserver.Handle("/", dir.Open("."))

	server := http.Server{
		Addr:    config.Addr + ":" + config.Port,
		Handler: nil,
	}

	go func() {
		err = server.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Println(err)
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	fmt.Println("Closing databases")
	err = db.Close()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("Closed databases")
	fmt.Println("Stopping server")
	_ = server.Shutdown(context.Background())
	fmt.Println("Server stopped")
}
