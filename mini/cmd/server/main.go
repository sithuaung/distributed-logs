// Command mini-server runs the mini log over HTTP on :8080.
//
// Try it:
//
//	go run ./mini/cmd/server
//
//	# in another terminal
//	curl -s -d '{"value":"aGVsbG8="}'  http://localhost:8080/produce
//	curl -s -d '{"value":"d29ybGQ="}'  http://localhost:8080/produce
//	curl -s -d '{"offset":0}'          http://localhost:8080/consume
//	curl -s -d '{"offset":1}'          http://localhost:8080/consume
//
// The "log" package imported as stdlib_log below is Go's stdlib
// logger — separate from the project's own log package, which we
// alias as minilog to avoid the name collision.
package main

import (
	stdlib_log "log"
	"net/http"

	minilog "github.com/sithuaung/distributed-logs/mini/log"
	miniserver "github.com/sithuaung/distributed-logs/mini/server"
)

func main() {
	l := minilog.New()
	handler := miniserver.New(l)

	addr := ":8080"
	stdlib_log.Printf("mini log server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		stdlib_log.Fatal(err)
	}
}
