package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

const PORT = 6769

// Helpers

func getBlob(digest string) (int64, error) {
	file, err := os.Stat(filepath.Join("blobs", digest))
	if err != nil {
		return 0, fmt.Errorf("digest %s not found", digest)
	}

	return file.Size(), nil
}

// Handlers

func handshake(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "{}")
}

func getBlobByDigest(w http.ResponseWriter, r *http.Request) {
	digest := r.PathValue("digest")

	size, err := getBlob(digest)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(int(size)))
	w.Header().Set("Docker-Content-Digest", digest)
	w.WriteHeader(200)

	return
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
