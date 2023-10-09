package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

var (
	port *string
)

func addHeaders(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	}
}

// sfs - simple file server
func sfs(dir string) {
	http.Handle("/", addHeaders(http.FileServer(http.Dir(dir))))
	fmt.Fprintf(os.Stdout, "Serving %v at %v\n", dir, *port)
	http.ListenAndServe(*port, nil)
}

func main() {
	port = flag.String("p", ":8081", "port where fileserver will be served")
	simple := flag.Bool("simple", false, "simply serve provided dir as a file server")
	dir := flag.String("dir", "", "simply serve provided dir as a file server")
	flag.Parse()
	if *dir == "" && *simple {
		fmt.Fprintf(os.Stderr, "simple %v requires '-dir' flag to be provided\n", os.Args[0])
		os.Exit(1)
	}
	if *simple {
		sfs(*dir)
	}
}
