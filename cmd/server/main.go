// Command server serves the api and the player's page.
//
//	go run ./cmd/server -addr localhost:8080 -root internal/scenario/testdata
package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/heaprip/intercessio/internal/api"
	"github.com/heaprip/intercessio/internal/front"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "address to listen on")
	root := flag.String("root", "internal/scenario/testdata", "directory with schema.json and scenario directories")
	flag.Parse()
	mux := http.NewServeMux()
	mux.Handle("/api/", (&api.Server{Root: *root}).Handler())
	mux.Handle("/", front.Handler())
	log.Printf("intercessio on http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
