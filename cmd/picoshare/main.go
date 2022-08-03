package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	gorilla "github.com/mtlynch/gorilla-handlers"

	"github.com/mtlynch/picoshare/v2/garbagecollect"
	"github.com/mtlynch/picoshare/v2/handlers"
	"github.com/mtlynch/picoshare/v2/handlers/auth/shared_secret"
	"github.com/mtlynch/picoshare/v2/store/sqlite"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Print("Starting picoshare server")

	dbPath := flag.String("db", "data/store.db", "path to database")
	flag.Parse()

	authenticator, err := shared_secret.New(requireEnv("PS_SHARED_SECRET"))
	if err != nil {
		log.Fatalf("invalid shared secret: %v", err)
	}

	ensureDirExists(filepath.Dir(*dbPath))

	store := sqlite.New(*dbPath)

	collector := garbagecollect.NewCollector(store)
	gc := garbagecollect.NewScheduler(&collector, 7*time.Hour)
	gc.StartAsync()

	h := gorilla.LoggingHandler(os.Stdout, handlers.New(authenticator, store, &collector).Router())
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
