// Package server exposes a mini.Log over plain HTTP + JSON.
//
// In the parent project this same role is played by internal/server,
// but using gRPC + protobuf + TLS + ACL. JSON over net/http is enough
// to feel the client/server split without any of that machinery.
package server

import (
	"encoding/json"
	"errors"
	"net/http"

	minilog "github.com/sithuaung/distributed-logs/mini/log"
)

type httpServer struct {
	Log *minilog.Log
}

// New returns an http.Handler with two endpoints:
//
//	POST /produce  {"value": "<base64-bytes>"}            -> {"offset": N}
//	POST /consume  {"offset": N}                          -> {"record": {...}}
//
// Note that JSON encodes []byte as base64, so a curl call looks like:
//
//	curl -d '{"value":"aGVsbG8="}' http://localhost:8080/produce
func New(log *minilog.Log) http.Handler {
	s := &httpServer{Log: log}
	mux := http.NewServeMux()
	mux.HandleFunc("/produce", s.handleProduce)
	mux.HandleFunc("/consume", s.handleConsume)
	return mux
}

type produceRequest struct {
	Value []byte `json:"value"`
}

type produceResponse struct {
	Offset uint64 `json:"offset"`
}

type consumeRequest struct {
	Offset uint64 `json:"offset"`
}

type consumeResponse struct {
	Record minilog.Record `json:"record"`
}

func (s *httpServer) handleProduce(w http.ResponseWriter, r *http.Request) {
	var req produceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	off, err := s.Log.Append(req.Value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(produceResponse{Offset: off})
}

func (s *httpServer) handleConsume(w http.ResponseWriter, r *http.Request) {
	var req consumeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rec, err := s.Log.Read(req.Offset)
	if errors.Is(err, minilog.ErrOffsetOutOfRange) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(consumeResponse{Record: rec})
}
