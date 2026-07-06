package controller

import (
	"encoding/json"
	"net/http"
)

const (
	ErrCodeBlobUnknown       = "BLOB_UNKNOWN"
	ErrCodeBlobUploadUnknown = "BLOB_UPLOAD_UNKNOWN"
	ErrCodeDigestInvalid     = "DIGEST_INVALID"
	ErrCodeManifestUnknown   = "MANIFEST_UNKNOWN"
	ErrCodeUnknown           = "UNKNOWN"
)

type registryError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	json.NewEncoder(w).Encode(struct {
		Error registryError `json:"error"`
	}{Error: registryError{Code: code, Message: message}})
}
