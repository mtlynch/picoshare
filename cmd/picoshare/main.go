package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	gorilla "github.com/mtlynch/gorilla-handlers"

	"github.com/mtlynch/picoshare/garbagecollect"
	"github.com/mtlynch/picoshare/handlers"
	"github.com/mtlynch/picoshare/handlers/auth/shared_secret"
	"github.com/mtlynch/picoshare/picoshare"
	"github.com/mtlynch/picoshare/space"
	"github.com/mtlynch/picoshare/store/sqlite"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Llongfile)
	log.Print("starting picoshare server")

	dbPath := flag.String("db", "data/store.db", "path to database")
	flag.Parse()

	dbDir := filepath.Dir(*dbPath)

	ensureDirExists(dbDir)

	secret, err := sharedSecretFromEnv()
	if err != nil {
		log.Fatalf("failed to read shared secret: %v", err)
	}
	authenticator := shared_secret.New(secret)

	store := sqlite.New(sqlite.Params{
		Path:                  *dbPath,
		OptimizeForLitestream: isLitestreamEnabled(),
		Now:                   time.Now,
	})

	spaceChecker := space.NewChecker(*dbPath, &store)

	collector := garbagecollect.NewCollector(store)
	gc := garbagecollect.NewScheduler(&collector, 7*time.Hour)
	gc.StartAsync()

	clock := handlers.NewClock()

	server := handlers.New(authenticator, &store, spaceChecker, &collector, &clock)

	// CrossOriginProtection rejects non-safe cross-origin requests to prevent CSRF.
	protectedRouter := http.NewCrossOriginProtection().Handler(server.Router())
	h := gorilla.LoggingHandler(os.Stdout, protectedRouter)
	if os.Getenv("PS_BEHIND_PROXY") != "" {
		h = gorilla.ProxyIPHeadersHandler(h)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "4001"
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	stop := setupSignalHandler()
	httpSrv := http.Server{Addr: fmt.Sprintf(":%s", port), Handler: h}
	go func() {
		log.Printf("listening on http://%s:%s", hostname, port)
		log.Printf("http server exit: %s", httpSrv.ListenAndServe())
	}()
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = httpSrv.Shutdown(ctx)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
}

func sharedSecretFromEnv() (picoshare.Passphrase, error) {
	if path := os.Getenv("PS_SHARED_SECRET_FILE"); path != "" {
		return sharedSecretFromFile(path)
	}
	secret := os.Getenv("PS_SHARED_SECRET")
	if secret == "" {
		return picoshare.Passphrase{},
			fmt.Errorf("PS_SHARED_SECRET or PS_SHARED_SECRET_FILE must be set")
	}
	passphrase, err := picoshare.NewPassphrase(secret)
	if err != nil {
		return picoshare.Passphrase{}, fmt.Errorf("invalid PS_SHARED_SECRET: %w", err)
	}
	return passphrase, nil
}

func sharedSecretFromFile(path string) (picoshare.Passphrase, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return picoshare.Passphrase{}, fmt.Errorf("failed to read PS_SHARED_SECRET_FILE: %w", err)
	}

	stripped := strings.TrimRight(string(data), "\r\n")
	passphrase, err := picoshare.NewPassphrase(stripped)
	if err != nil {
		return picoshare.Passphrase{}, fmt.Errorf("invalid PS_SHARED_SECRET_FILE: %w", err)
	}
	return passphrase, nil
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

func setupSignalHandler() <-chan struct{} {
	stop := make(chan struct{})
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		close(stop)
		<-c
		os.Exit(1) // second signal. Exit directly.
	}()
	return stop
}
