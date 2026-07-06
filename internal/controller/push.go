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
	if os.IsNotExist(err) {
		writeError(w, 404, ErrCodeBlobUnknown, "blob unknown to registry")
		return
	}
	if err != nil {
		fmt.Println(err.Error())
		writeError(w, 500, ErrCodeUnknown, "internal server error")
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
			fmt.Println(err.Error())
			writeError(w, 500, ErrCodeUnknown, "internal server error")
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
			writeError(w, 404, ErrCodeBlobUploadUnknown, "blob upload unknown")
			return
		}
		if err != nil {
			fmt.Println(err.Error())
			writeError(w, 500, ErrCodeUnknown, "internal server error")
			return
		}

		file, err := os.OpenFile(path.Join("blobs", "uploads", id), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println(err.Error())
			writeError(w, 500, ErrCodeUnknown, "internal server error")
			return
		}

		defer file.Close()

		buffer, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err.Error())
			writeError(w, 500, ErrCodeUnknown, "internal server error")
			return
		}

		_, err = file.Write(buffer)
		if err != nil {
			fmt.Println(err.Error())
			writeError(w, 500, ErrCodeUnknown, "internal server error")
			return
		}

		newRangeEnd := len(buffer) + rangeEnd
		err = service.UpdateUploadRangeEnd(db, id, newRangeEnd)
		if err != nil {
			fmt.Println(err.Error())
			writeError(w, 500, ErrCodeUnknown, "internal server error")
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
			writeError(w, 400, ErrCodeDigestInvalid, "digest not provided")
			return
		}

		err := service.UploadExists(db, id)
		if err == sql.ErrNoRows {
			writeError(w, 404, ErrCodeBlobUploadUnknown, "blob upload unknown")
			return
		}
		if err != nil {
			fmt.Println(err.Error())
			writeError(w, 500, ErrCodeUnknown, "internal server error")
			return
		}

		file, err := os.Open(path.Join("blobs", "uploads", id))
		if err != nil {
			fmt.Println(err.Error())
			writeError(w, 500, ErrCodeUnknown, "internal server error")
			return
		}
		defer file.Close()

		bytes, err := io.ReadAll(file)
		if err != nil {
			fmt.Println(err.Error())
			writeError(w, 500, ErrCodeUnknown, "internal server error")
			return
		}

		fileDigest := sha256.Sum256(bytes)
		digestString := fmt.Sprintf("sha256:%x", fileDigest)

		if digestString != digest {
			writeError(w, 400, ErrCodeDigestInvalid, "provided digest did not match uploaded content")
			return
		}

		err = os.Rename(path.Join("blobs", "uploads", id), path.Join("blobs", digestString))
		if err != nil {
			fmt.Println(err.Error())
			writeError(w, 500, ErrCodeUnknown, "internal server error")
			return
		}

		err = service.DeleteUpload(db, id)
		if err != nil {
			fmt.Println(err.Error())
			writeError(w, 500, ErrCodeUnknown, "internal server error")
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
			writeError(w, 500, ErrCodeUnknown, "internal server error")
			return
		}

		bodyDigest := sha256.Sum256(buffer)
		digestString := fmt.Sprintf("sha256:%x", bodyDigest)

		err = service.InsertImage(db, digestString, string(buffer))
		if err != nil {
			fmt.Println(err.Error())
			writeError(w, 500, ErrCodeUnknown, "internal server error")
			return
		}

		if !strings.HasPrefix(reference, "sha256:") {
			err = service.UpsertTag(db, name, reference, digestString)
			if err != nil {
				fmt.Println(err.Error())
				writeError(w, 500, ErrCodeUnknown, "internal server error")
				return
			}
		}

		w.Header().Add("Docker-Content-Digest", digestString)
		w.Header().Set("Location", fmt.Sprintf("/v2/%s/manifests/%s", name, reference))
		w.WriteHeader(201)
	}
}
