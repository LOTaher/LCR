package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
)

const PORT = 6769

// Handlers

func handshake(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "{}")
}

func getBlobByDigest(w http.ResponseWriter, r *http.Request) {
	// Search /blobs/ directory for the digest
}

// https://github.com/opencontainers/distribution-spec/blob/main/spec.md

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v2/", handshake)
	mux.HandleFunc("HEAD /v2/{name}/blobs/{digest}", getBlobByDigest)

	fmt.Printf("LCR Listening on %d\n", PORT)
	http.ListenAndServe(":"+strconv.Itoa(PORT), mux)

	os.Exit(0)
}
