package controller

import (
	"fmt"
	"net/http"
)

func Handshake(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "{}")
}
