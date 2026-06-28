package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
)

const PORT = 6769

// DB

func initDB(db *sql.DB) error {
	createCommand := `
	CREATE TABLE IF NOT EXISTS images (
		digest TEXT PRIMARY KEY,
		manifest TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS tags (
		image_name TEXT NOT NULL,
		tag_name TEXT NOT NULL,
		digest TEXT NOT NULL REFERENCES images(digest),
		PRIMARY KEY (image_name, tag_name)
	);

	CREATE TABLE IF NOT EXISTS uploads (
		uuid TEXT PRIMARY KEY,
		image_name TEXT NOT NULL,
		range_end INTEGER NOT NULL DEFAULT 0
	);
	`

	_, err := db.Exec(createCommand)
	if err != nil {
		return err
	}

	fmt.Println("Database initialized")

	return nil
}

// Handlers

func handshake(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "{}")
}

func getBlobByDigest(w http.ResponseWriter, r *http.Request) {
	digest := r.PathValue("digest")

	file, err := os.Stat(filepath.Join("blobs", digest))
	if err != nil {
		w.WriteHeader(404)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(int(file.Size())))
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

func handleBlobUpload(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		id := r.PathValue("uuid")

		var uploadId string
		var rangeEnd int
		err := db.QueryRow("SELECT uuid, range_end FROM uploads WHERE uuid = ?", id).Scan(&uploadId, &rangeEnd)
		if err == sql.ErrNoRows {
			w.WriteHeader(404)
			return
		}
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		file, err := os.OpenFile(path.Join("blobs", "uploads", id), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		defer file.Close()

		buffer, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		_, err = file.Write(buffer)
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		var newRangeEnd = len(buffer) + rangeEnd
		_, err = db.Exec("UPDATE uploads SET range_end = ? WHERE uuid = ?", newRangeEnd, id)
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		w.Header().Add("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, id))
		w.Header().Add("Range", fmt.Sprintf("0-%d", newRangeEnd-1))
		w.Header().Add("Docker-Upload-UUID", id)

		w.WriteHeader(204)
	}
}

func completeBlobUpload(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		id := r.PathValue("uuid")
		digest := r.URL.Query().Get("digest")

		if digest == "" {
			w.WriteHeader(400)
			return
		}

		var uploadId string
		err := db.QueryRow("SELECT uuid FROM uploads WHERE uuid = ?", id).Scan(&uploadId)
		if err == sql.ErrNoRows {
			w.WriteHeader(404)
			return
		}
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		file, err := os.Open(path.Join("blobs", "uploads", id))
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}
		defer file.Close()

		bytes, err := io.ReadAll(file)
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		fileDigest := sha256.Sum256(bytes)
		digestString := fmt.Sprintf("sha256:%x", fileDigest)

		if digestString != digest {
			w.WriteHeader(400)
			return
		}

		err = os.Rename(path.Join("blobs", "uploads", id), path.Join("blobs", digestString))
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		_, err = db.Exec("DELETE FROM uploads WHERE uuid = ?", id)
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		w.Header().Add("Docker-Content-Digest", digestString)
		w.Header().Add("Location", fmt.Sprintf("/v2/%s/blobs/%s", name, digestString))
		w.WriteHeader(201)
	}
}

func handleManifest(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		reference := r.PathValue("reference")

		buffer, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		bodyDigest := sha256.Sum256(buffer)
		digestString := fmt.Sprintf("sha256:%x", bodyDigest)

		_, err = db.Exec("INSERT INTO images (digest, manifest) VALUES (?, ?)", digestString, string(buffer))
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		if !strings.HasPrefix(reference, "sha256:") {
			_, err = db.Exec("INSERT INTO tags (image_name, tag_name, digest) VALUES (?, ?, ?) ON CONFLICT(image_name, tag_name) DO UPDATE SET digest=excluded.digest", name, reference, digestString)
			if err != nil {
				fmt.Println(err.Error())
				w.WriteHeader(500)
				return
			}
		}

		w.Header().Add("Docker-Content-Digest", digestString)
		w.Header().Set("Location", fmt.Sprintf("/v2/%s/manifests/%s", name, reference))
		w.WriteHeader(201)
	}
}

func getManifest(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		reference := r.PathValue("reference")

		var manifest string
		var digest string
		if !strings.HasPrefix(reference, "sha256:") {
			err := db.QueryRow("SELECT digest FROM tags WHERE tag_name = ?", reference).Scan(&digest)
			if err == sql.ErrNoRows {
				w.WriteHeader(404)
				return
			}
			if err != nil {
				fmt.Println(err.Error())
				w.WriteHeader(500)
				return
			}
			err = db.QueryRow("SELECT manifest FROM images WHERE digest = ?", digest).Scan(&manifest)
			if err == sql.ErrNoRows {
				w.WriteHeader(404)
				return
			}
			if err != nil {
				fmt.Println(err.Error())
				w.WriteHeader(500)
				return
			}
		} else {
			digest = reference
			err := db.QueryRow("SELECT manifest FROM images WHERE digest = ?", digest).Scan(&manifest)
			if err == sql.ErrNoRows {
				w.WriteHeader(404)
				return
			}
			if err != nil {
				fmt.Println(err.Error())
				w.WriteHeader(500)
				return
			}
		}

		w.Header().Add("Docker-Content-Digest", digest)
		w.Header().Add("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		w.WriteHeader(200)
		fmt.Fprint(w, manifest)
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

	// Push
	mux.HandleFunc("HEAD /v2/{name}/blobs/{digest}", getBlobByDigest)
	mux.HandleFunc("POST /v2/{name}/blobs/uploads/", startBlobUpload(db))
	mux.HandleFunc("PATCH /v2/{name}/blobs/uploads/{uuid}", handleBlobUpload(db))
	mux.HandleFunc("PUT /v2/{name}/blobs/uploads/{uuid}", completeBlobUpload(db))
	mux.HandleFunc("PUT /v2/{name}/manifests/{reference}", handleManifest(db))

	// Pull
	mux.HandleFunc("GET /v2/{name}/manifests/{reference}", getManifest(db))

	fmt.Printf("LCR Listening on %d\n", PORT)
	http.ListenAndServe(":"+strconv.Itoa(PORT), mux)

	os.Exit(0)
}
