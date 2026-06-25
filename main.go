package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const PORT = 6769

// DB

func initDB(db *sql.DB) error {
	createCommand := `
	CREATE TABLE IF NOT EXISTS images (
		digest TEXT PRIMARY KEY,
		manifest JSON NOT NULL
	);

	CREATE TABLE IF NOT EXISTS tags (
		image_name TEXT NOT NULL,
		tag_name TEXT NOT NULL,
		digest TEXT NOT NULL REFERENCES images(digest),
		PRIMARY KEY (image_name, tag_name)
	);

	CREATE TABLE IF NOT EXISTS uploads (
		uuid TEXT PRIMARY KEY,
		image_name TEXT NOT NULL
	);
	`

	_, err := db.Exec(createCommand)
	if err != nil {
		return err
	}

	fmt.Println("Database initialized")

	return nil
}

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
}

func startBlobUpload(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		id := uuid.New().String()

		_, err := db.Exec("INSERT INTO uploads (uuid, image_name) VALUES (?, ?)", id, name)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, id))
		w.Header().Set("Docker-Upload-UUID", id)
		w.WriteHeader(202)
	}
}

// https://github.com/opencontainers/distribution-spec/blob/main/spec.md

func main() {
	// Init DB
	db, err := sql.Open("sqlite3", "./lcr.db")
	if err != nil {
		panic(err)
	}

	err = initDB(db)
	if err != nil {
		panic(err)
	}

	defer db.Close()

	// Init HTTP
	mux := http.NewServeMux()

	mux.HandleFunc("GET /v2/", handshake)
	mux.HandleFunc("HEAD /v2/{name}/blobs/{digest}", getBlobByDigest)
	mux.HandleFunc("POST /v2/{name}/blobs/uploads/", startBlobUpload(db))

	fmt.Printf("LCR Listening on %d\n", PORT)
	http.ListenAndServe(":"+strconv.Itoa(PORT), mux)

	os.Exit(0)
}
