package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func addHeaders(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	}
}

type static struct {
	Html string
}

func New() static {
	return static{
		Html: "<!DOCTYPE html> <html> <head> <style> span { display: flex; align-items: center;} </style> </head> <body>",
	}
}

const music string = "<span><audio controls loop src=\"{{.file}}\" type=\"audio/mpeg\"> </audio><font size=\"5\"> - {{.file}}</font></span><br>"

const video string = "<span><video controls width=\"320\" height=\"240\" src=\"{{.file}}\" type=\"video/mp4\"></video><font size=\"5\"> - {{.file}}</font></span><br>"

func (s static) String() string {
	return s.Html + "</body><br></html>"
}

func (s static) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, s.String())
}

func logs(verbose bool, w io.Writer, format string, a ...any) {
	if !verbose {
		return
	}
	fmt.Fprintf(w, format, a...)
}

func main() {
	port := flag.String("p", ":8081", "port where fileserver will be served")
	dir := flag.String("d", "", "directory to serve")
	verbose := flag.Bool("v", false, "verbose application")
	flag.Parse()
	if *dir == "" {
		logs(*verbose, os.Stderr, "[ERROR]: -d not provided")
		return
	}
	entries, err := os.ReadDir(*dir)
	if err != nil {
		logs(*verbose, os.Stderr, "[ERROR]: could not open directory '%v'\n", *dir)
		os.Exit(1)
	}
	st := New()
	for _, e := range entries {
		eName := e.Name()
		ext := filepath.Ext(eName)
		var ehtml string
		switch ext {
		case ".mp3":
			ehtml = music
		case ".mp4":
			ehtml = video
		case ".mkv":
			ehtml = video
		default:
			logs(*verbose, os.Stdout, "[INFO]: skipping file '%v'\n", eName)
			continue
		}
		ehtml = strings.ReplaceAll(ehtml, "{{.file}}", eName)
		st.Html += ehtml
		logs(*verbose, os.Stdout, "[INFO]: handled file %v\n", eName)
	}
	http.Handle("/", addHeaders(http.FileServer(http.Dir(*dir))))
	http.Handle("/st", st)
	logs(*verbose, os.Stdout, "Serving %v at %v\n", *dir, *port)
	http.ListenAndServe(*port, nil)
}
