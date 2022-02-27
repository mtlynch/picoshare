package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	gorilla "github.com/mtlynch/gorilla-handlers"

	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret"
	"github.com/mtlynch/picoshare/v2/store/sqlite"
	"github.com/mtlynch/picoshare/v2/types"
)

type fakeReader struct {
	n int
}

func (r *fakeReader) Read(p []byte) (int, error) {
	if r.n > (5 * 1000 * 1000 * 1000) {
		return 0, io.EOF
	}
	for i := 0; i < len(p); i++ {
		p[i] = 'A'
	}
	r.n += len(p)
	return len(p), nil
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Print("Starting picoshare server")

	dbPath := flag.String("db", "data/store.db", "path to database")
	flag.Parse()

	if *dbPath == "dbg" {
		db := sqlite.New("/tmp/db.db")
		fr := fakeReader{}
		db.InsertEntry(&fr, types.UploadMetadata{
			Filename: "dummy-file.txt",
		})
		return
	}

	authenticator, err := shared_secret.New(requireEnv("PS_SHARED_SECRET"))
	if err != nil {
		log.Fatalf("invalid shared secret: %v", err)
	}

	ensureDirExists(filepath.Dir(*dbPath))

	h := gorilla.LoggingHandler(os.Stdout, handlers.New(authenticator, sqlite.New(*dbPath)).Router())
	if os.Getenv("PS_BEHIND_PROXY") != "" {
		h = gorilla.ProxyIPHeadersHandler(h)
	}
	http.Handle("/", h)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4001"
	}
	log.Printf("Listening on %s", port)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), nil))
}

func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprintf("missing required environment variable: %s", key))
	}
	return val
}

func ensureDirExists(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			panic(err)
		}
	}
}
