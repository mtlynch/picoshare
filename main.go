package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	"github.com/ncruces/go-sqlite3"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	"github.com/ncruces/go-sqlite3/ext/blobio"
	"github.com/tetratelabs/wazero"
)

func main() {
	log.Printf("starting up!")

	sqlite3.RuntimeConfig = wazero.NewRuntimeConfig().WithMemoryLimitPages(64)
	sqlite3.AutoExtension(blobio.Register)

	var err error
	db, err := sql.Open("sqlite3", "file:test.db?_pragma=journal_mode(WAL)")
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS files (name, data)`)
	if err != nil {
		log.Panic(err)
	}

	for _, path := range os.Args[1:] {
		func() {
			file, err := os.Open(path)
			if err != nil {
				log.Panic(err)
			}
			defer file.Close()

			info, err := file.Stat()
			if err != nil {
				log.Panic(err)
			}
			log.Println("size", info.Size())

			r, err := db.Exec(`INSERT INTO files (name, data) VALUES (?, ?)`,
				filepath.Base(path), sqlite3.ZeroBlob(info.Size()))
			if err != nil {
				log.Panic(err)
			}

			id, err := r.LastInsertId()
			if err != nil {
				log.Panic(err)
			}
			log.Println("id", id)

			_, err = db.Exec(`SELECT writeblob('main', 'files', 'data', ?, 0, ?)`,
				id, sqlite3.Pointer(file))
			if err != nil {
				log.Panic(err)
			}
		}()
	}
}
