package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"

	_ "github.com/mattn/go-sqlite3"

	"lcr/internal/controller"
	"lcr/internal/service"
)

const PORT = 6769
const LCR_USERNAME = "admin"
const LCR_PASSWORD = "secret"

// https://github.com/opencontainers/distribution-spec/blob/main/spec.md

func main() {
	// Init DB
	db, err := sql.Open("sqlite3", "./lcr.db")
	if err != nil {
		panic(err)
	}

	err = service.InitDB(db)
	if err != nil {
		panic(err)
	}

	defer db.Close()

	// Init HTTP
	mux := http.NewServeMux()

	// Page
	mux.HandleFunc("/", controller.RenderHomePage(db))

	// Registry
	mux.HandleFunc("GET /v2/", controller.Handshake)

	// Push
	mux.HandleFunc("HEAD /v2/{name}/blobs/{digest}", controller.GetBlobByDigest)
	mux.HandleFunc("POST /v2/{name}/blobs/uploads/", controller.StartBlobUpload(db))
	mux.HandleFunc("PATCH /v2/{name}/blobs/uploads/{uuid}", controller.HandleBlobUpload(db))
	mux.HandleFunc("PUT /v2/{name}/blobs/uploads/{uuid}", controller.CompleteBlobUpload(db))
	mux.HandleFunc("PUT /v2/{name}/manifests/{reference}", controller.HandleManifest(db))

	// Pull
	mux.HandleFunc("GET /v2/{name}/manifests/{reference}", controller.GetManifest(db))
	mux.HandleFunc("GET /v2/{name}/blobs/{digest}", controller.SendBlobByDigest)

	// TODO misc
	// tags list route: GET /v2/<name>/tags/list

	fmt.Printf("LCR Listening on %d\n", PORT)
	if err := http.ListenAndServe(":"+strconv.Itoa(PORT), mux); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
