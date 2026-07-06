package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strconv"

	_ "github.com/mattn/go-sqlite3"

	"lcr/internal/controller"
	"lcr/internal/db"
)

const PORT = 6769
const LCR_USERNAME = "admin"
const LCR_PASSWORD = "secret"

// https://github.com/opencontainers/distribution-spec/blob/main/spec.md

func main() {
	// Init DB
	database, err := sql.Open("sqlite3", "./lcr.db")
	if err != nil {
		panic(err)
	}

	err = db.Init(database)
	if err != nil {
		panic(err)
	}

	defer database.Close()

	// Init HTTP
	mux := http.NewServeMux()

	// Page
	mux.HandleFunc("/", controller.RenderHomePage(database))

	// Registry
	mux.HandleFunc("GET /v2/", controller.Handshake)

	// Push
	mux.HandleFunc("HEAD /v2/{name}/blobs/{digest}", controller.GetBlobByDigest)
	mux.HandleFunc("POST /v2/{name}/blobs/uploads/", controller.StartBlobUpload(database))
	mux.HandleFunc("PATCH /v2/{name}/blobs/uploads/{uuid}", controller.HandleBlobUpload(database))
	mux.HandleFunc("PUT /v2/{name}/blobs/uploads/{uuid}", controller.CompleteBlobUpload(database))
	mux.HandleFunc("PUT /v2/{name}/manifests/{reference}", controller.HandleManifest(database))

	// Pull
	mux.HandleFunc("GET /v2/{name}/manifests/{reference}", controller.GetManifest(database))
	mux.HandleFunc("GET /v2/{name}/blobs/{digest}", controller.SendBlobByDigest)

	// TODO misc
	// tags list route: GET /v2/<name>/tags/list

	fmt.Printf("LCR Listening on %d\n", PORT)
	if err := http.ListenAndServe(":"+strconv.Itoa(PORT), mux); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
