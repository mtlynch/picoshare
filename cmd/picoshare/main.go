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
	"github.com/mtlynch/picoshare/v2/space"
	"github.com/mtlynch/picoshare/v2/store/sqlite"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Print("starting picoshare server")

	dbPath := flag.String("db", "data/store.db", "path to database")
	vacuumDb := flag.Bool("vacuum", false, "vacuum database periodically to reclaim disk space")
	flag.Parse()

	authenticator, err := shared_secret.New(requireEnv("PS_SHARED_SECRET"))
	if err != nil {
		log.Fatalf("invalid shared secret: %v", err)
	}

	dbDir := filepath.Dir(*dbPath)

	ensureDirExists(dbDir)

	store := sqlite.New(*dbPath, isLitestreamEnabled())

	spaceChecker := space.NewChecker(dbDir)

	collector := garbagecollect.NewCollector(store, *vacuumDb)
	gc := garbagecollect.NewScheduler(&collector, 7*time.Hour)
	gc.StartAsync()

	server, err := handlers.New(authenticator, store, spaceChecker, &collector)
	if err != nil {
		panic(err)
	}

	h := gorilla.LoggingHandler(os.Stdout, server.Router())
	if os.Getenv("PS_BEHIND_PROXY") != "" {
		h = gorilla.ProxyIPHeadersHandler(h)
	}
	http.Handle("/", h)

	port := os.Getenv("PORT")
	if port == "" {
		port = "4001"
	}
	log.Printf("listening on %s", port)

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

func isLitestreamEnabled() bool {
	return os.Getenv("LITESTREAM_BUCKET") != ""
}
