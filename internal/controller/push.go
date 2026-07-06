package controller

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

	"lcr/internal/service"
)

func GetBlobByDigest(w http.ResponseWriter, r *http.Request) {
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

func StartBlobUpload(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")

		id, err := service.CreateUpload(db, name)
		if err != nil {
			w.WriteHeader(500)
			return
		}

		w.Header().Set("Location", fmt.Sprintf("/v2/%s/blobs/uploads/%s", name, id))
		w.Header().Set("Docker-Upload-UUID", id)
		w.WriteHeader(202)
	}
}

func HandleBlobUpload(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		id := r.PathValue("uuid")

		rangeEnd, err := service.GetUploadRangeEnd(db, id)
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

		newRangeEnd := len(buffer) + rangeEnd
		err = service.UpdateUploadRangeEnd(db, id, newRangeEnd)
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

func CompleteBlobUpload(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		id := r.PathValue("uuid")
		digest := r.URL.Query().Get("digest")

		if digest == "" {
			w.WriteHeader(400)
			return
		}

		err := service.UploadExists(db, id)
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

		err = service.DeleteUpload(db, id)
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

func HandleManifest(db *sql.DB) func(w http.ResponseWriter, r *http.Request) {
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

		err = service.InsertImage(db, digestString, string(buffer))
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(500)
			return
		}

		if !strings.HasPrefix(reference, "sha256:") {
			err = service.UpsertTag(db, name, reference, digestString)
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
