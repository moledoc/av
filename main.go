package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	_ "text/template"
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
	http.Handle("/public", addHeaders(http.FileServer(http.Dir("./public"))))
	// fmt.Fprintf(os.Stdout, "Serving %v at %v\n", dir, *port)
	// http.ListenAndServe(*port, nil)
}

// html file
type static struct {
	Html string
}

func New() static {
	return static{
		Html: "<!DOCTYPE html> <html> <head> <style> span { display: flex; align-items: center;} </style> </head> <body>",
	}
}

const music string = "<span> <img src=\"goava.jpg\" alt=\"goava\" width=50px height=50px /><audio controls loop src=\"{{.file}}\" type=\"audio/mpeg\"> </audio><font size=\"5\"> - {{.file}}</font> </span><br>"

const video string = "<span> <img src=\"goava.jpg\" alt=\"goava\" width=50px height=50px /><video controls width=\"320\" height=\"240\" src=\"{{.file}}\" type=\"video/mp4\"></video><font size=\"5\"> - {{.file}}</font> </span><br>"

func (s static) String() string {
// 	wd, _ := os.Getwd()
// 	fmt.Println(wd+ "/resources/goava.jpg")
	return s.Html+"</body><br></html>"
}

func (s static) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, s.String())
	// http.ServeFile(w, r, fmt.Sprintf("./tst.html"))

}

func main() {
	port = flag.String("p", ":8081", "port where fileserver will be served")
	simple := flag.Bool("simple", false, "simply serve provided dir as a file server")
	dir := flag.String("d", "", "directory to serve")
	flag.Parse()
	_ = simple
//	if *simple {
		sfs(*dir)
//	} 
//else {
		entries, err := os.ReadDir(*dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR]: could not open directory '%v'\n", *dir)
			os.Exit(1)
		}
		st := New()
		fmt.Println(entries)
		for _, e := range entries {
			eName := e.Name()
			ext := filepath.Ext(eName)
			var ehtml string
			switch ext {
			case ".mp3":
				ehtml = music
				continue
			case ".mp4":
				ehtml = video
			case ".mkv":
				ehtml = video
			default:
				fmt.Fprintf(os.Stdout, "[INFO]: skipping file %v\n", eName)
				continue
			}
			ehtml = strings.ReplaceAll(ehtml, "{{.file}}", eName)
			st.Html += ehtml
			fmt.Fprintf(os.Stdout, "[INFO]: handled file %v\n", eName)
		}
		http.Handle("/st", st)
		fmt.Fprintf(os.Stdout, "Serving %v at %v\n", *dir, *port)
		http.ListenAndServe(*port, nil)
//	}
}
