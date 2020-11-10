package main

import (
	"net/http"
	"os"
)

func pong(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusTeapot)
}

func main() {
	http.HandleFunc("/", pong)
	http.ListenAndServe(os.Getenv("ADDR"), nil)
}
