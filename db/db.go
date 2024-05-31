package db

import (
	"github.com/asdine/storm/v3"
)

func fn() {
	db, err := storm.Open("my.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()
}