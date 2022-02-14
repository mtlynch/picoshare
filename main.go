package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	gorilla "github.com/mtlynch/gorilla-handlers"

	"github.com/mtlynch/picoshare/v2/handlers"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Print("Starting picoshare server")

	h := gorilla.LoggingHandler(os.Stdout, handlers.New().Router())
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
