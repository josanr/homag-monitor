package stats

import (
	"database/sql"
	_ "github.com/mattn/go-sqlite3"
	"log"
)

func createConnection() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "/home/ruslan/Projects/Golang/HomagMonitor/test_assets/db.sqlite3")
	if err != nil {
		log.Println("error open: " + err.Error())
		return nil, err
	}
	if err := db.Ping(); err != nil {
		log.Println("error ping:" + err.Error())
		return nil, err
	}

	return db, nil
}
