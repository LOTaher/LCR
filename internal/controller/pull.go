package controller

import (
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"lcr/internal/service"
)

func GetManifest(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		reference := r.PathValue("reference")

		var digest string
		var err error

		if !strings.HasPrefix(reference, "sha256:") {
			digest, err = service.GetDigestByTag(db, reference)
			if err == sql.ErrNoRows {
				writeError(w, 404, ErrCodeManifestUnknown, "manifest unknown")
				return
			}
			if err != nil {
				fmt.Println(err.Error())
				writeError(w, 500, ErrCodeUnknown, "internal server error")
				return
			}
		} else {
			digest = reference
		}

		manifest, err := service.GetManifestByDigest(db, digest)
		if err == sql.ErrNoRows {
			writeError(w, 404, ErrCodeManifestUnknown, "manifest unknown")
			return
		}
		if err != nil {
			fmt.Println(err.Error())
			writeError(w, 500, ErrCodeUnknown, "internal server error")
			return
		}

		err = service.IncrementTagDownload(db, reference)
		if err != nil {
			fmt.Println(err.Error())
			writeError(w, 500, ErrCodeUnknown, "internal server error")
			return
		}

		w.Header().Add("Docker-Content-Digest", digest)
		w.Header().Add("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		w.WriteHeader(200)
		fmt.Fprint(w, manifest)
	}
}

func SendBlobByDigest(w http.ResponseWriter, r *http.Request) {
	digest := r.PathValue("digest")

	fileInfo, err := os.Stat(filepath.Join("blobs", digest))
	if os.IsNotExist(err) {
		writeError(w, 404, ErrCodeBlobUnknown, "blob unknown to registry")
		return
	}
	if err != nil {
		fmt.Println(err.Error())
		writeError(w, 500, ErrCodeUnknown, "internal server error")
		return
	}

	file, err := os.Open(filepath.Join("blobs", digest))
	if err != nil {
		fmt.Println(err.Error())
		writeError(w, 500, ErrCodeUnknown, "internal server error")
		return
	}

	defer file.Close()

	w.Header().Add("Docker-Content-Digest", digest)
	w.Header().Add("Content-Type", "application/octet-stream")
	w.Header().Add("Content-Length", strconv.Itoa(int(fileInfo.Size())))
	w.WriteHeader(200)
	io.Copy(w, file)
}
