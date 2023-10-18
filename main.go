package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	ffmpeg *bool
	verbose *bool
	dir *string
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
	s.parse(dir)
	fmt.Fprintf(w, s.String())
}

func logs(verbose *bool, w io.Writer, format string, a ...any) {
	if !(*verbose) {
		return
	}
	fmt.Fprintf(w, format, a...)
}

func concatAudio(parentDir string, dirPath string) {
	// concat audio files - https://superuser.com/questions/809623/how-to-join-audio-files-of-different-formats-in-ffmpeg

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		logs(verbose, os.Stderr, "[ERROR]: could not open directory '%v'\n", dirPath)
		return
	}
	dirElems := strings.Split(dirPath, "/")
	leafDir := dirElems[len(dirElems)-1]
	catName := parentDir + "/" + leafDir + ".mp3"
	var args []string
	var count uint8
	for _, e := range entries {
		eName := e.Name()
		if e.IsDir() && *ffmpeg {
			go concatAudio(parentDir, dirPath + "/" + eName)
			continue
		}
		ext := filepath.Ext(eName)
		if ext != catName && ext == ".mp3" || ext == ".flac" || ext == ".wav" {
				args = append(args, fmt.Sprintf("-i '%v/%v'", dirPath, eName))
				count++
		} else {
			logs(verbose, os.Stdout, "[INFO]: skipping file '%v'\n", eName)
			continue
		}
	}
	if count == 0 {
		return
	}
	args = append(args, fmt.Sprintf("-filter_complex 'concat=n=%v:v=0:a=1[a]'", count))
	args = append(args, "-map '[a]'")
	args = append(args, "-codec:a libmp3lame")
	args = append(args, "-b:a 256k")
	args = append(args, fmt.Sprintf("'%v'", catName))
	// args = append(args, "> /dev/null 2>&1 &")
	cmd := exec.Command("ffmpeg", args...)
	logs(verbose, os.Stdout, "[INFO]: cmd: %v\n", cmd.String())
	err = cmd.Start()
	if err != nil {
		logs(verbose, os.Stderr, "[ERROR]: encountered error while starting command:\n\tcmd - %v\n\terr - %v\n", cmd.String(), err)
		return
	}
	err = cmd.Wait()
	if err != nil {
		logs(verbose, os.Stderr, "[ERROR]: encountered error while waiting for command:\n\tcmd - %v\n\terr - %v\n", cmd.String(), err)
		return
	}
}

func (st *static) parse(dir *string) {
	entries, err := os.ReadDir(*dir)
	if err != nil {
		logs(verbose, os.Stderr, "[ERROR]: could not open directory '%v'\n", *dir)
		return
	}
	for _, e := range entries {
		eName := e.Name()
		if e.IsDir() && *ffmpeg {
			go concatAudio(*dir, *dir+"/" +eName)
			continue
		}
		ext := filepath.Ext(eName)
		var ehtml string
		if ext == ".mp3" || ext == ".flac" || ext == ".wav" {
				ehtml = music
		} else if ext == ".mp4" || ext == ".mkv" {
				ehtml = video
		} else {
			logs(verbose, os.Stdout, "[INFO]: skipping file '%v'\n", eName)
			continue
		}
		ehtml = strings.ReplaceAll(ehtml, "{{.file}}", eName)
		st.Html += ehtml
		logs(verbose, os.Stdout, "[INFO]: handled file %v\n", eName)
	}
}

func main() {
	port := flag.String("p", ":8082", "port where fileserver will be served")
	dir = flag.String("d", "", "directory to serve")
	verbose = flag.Bool("v", false, "verbose application")
	ffmpeg = flag.Bool("ffmpeg", false, "concat media files")
	flag.Parse()
	if *dir == "" {
		logs(verbose, os.Stderr, "[ERROR]: -d not provided")
		return
	}
	st := New()
	http.Handle("/", addHeaders(http.FileServer(http.Dir(*dir))))
	http.Handle("/st", st)
	logs(verbose, os.Stdout, "Serving %v at %v\n", *dir, *port)
	http.ListenAndServe(*port, nil)
}
