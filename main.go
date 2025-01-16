package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sophuwu.site/cdn/dbfs"
	"sophuwu.site/cdn/dir"
	"sophuwu.site/cdn/fileserver"
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

func main() {
	db, err := dbfs.OpenDB("build/my.db")
	if err != nil {
		fmt.Println(err)
		return
	}

	fileserver.Handle("/dir/", dir.Open("."))
	fileserver.Handle("/db/", db.GetEntry)

	sig := make(chan os.Signal)
	signal.Notify(sig, os.Interrupt)

	server := http.Server{
		Addr:    ":8080",
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
