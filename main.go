package main

import (
	"fmt"
	"net/http"
)

func addHeaders(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	}
}
func main() {
	http.Handle("/", addHeaders(http.FileServer(http.Dir("./av"))))
	serving := ":8081"
	fmt.Printf("Serving %v\n", serving)
	http.ListenAndServe(serving, nil)
}
