package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/omerkarahan/loldmg/internal/web"
)

func main() {
	var (
		addr   = flag.String("addr", ":8080", "listen address")
		apiKey = flag.String("api-key", os.Getenv("RIOT_API_KEY"), "Riot API key (defaults to RIOT_API_KEY env var)")
		region = flag.String("region", "europe", "default Riot routing region")
		dev    = flag.String("dev", "", "load sample timeline/match JSON for local dev (e.g. internal/analyzer/testdata)")
	)
	flag.Parse()

	if *apiKey == "" {
		fmt.Fprintln(os.Stderr, "warning: RIOT_API_KEY not set; /api/analyze will fail")
	}

	srv := web.New(*apiKey, *region, *dev)
	log.Printf("server listening on %s", *addr)
	if err := http.ListenAndServe(*addr, srv.Handler()); err != nil {
		log.Fatalf("server: %v", err)
	}
}
