package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	ffmpeg *bool
	verbose *bool
	vverbose *bool
	dir *string
)

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

func errlog(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "[ERROR]: " + format, a...)
}

func warnlog(format string, a ...any) {
	if *verbose || *vverbose {
		fmt.Fprintf(os.Stdout, "[WARNING]: " + format, a...)
	}
}

func infolog(format string, a ...any) {
	fmt.Fprintf(os.Stdout, "[INFO]: " + format, a...)
}

func debuglog(format string, a ...any) {
	if *vverbose {
		fmt.Fprintf(os.Stdout, "[DEBUG]: " + format, a...)
	}
}

// concatAudio is a func that follows directories recursively and all same level audio files are concatenated to a single mp3 file at the parent directory, which is the directory served by the web server.
// NOTE: there is an external dependency on 'ffmpeg'.
func concatAudio(parentDir string, dirPath string) {
	// concat audio files - https://superuser.com/questions/809623/how-to-join-audio-files-of-different-formats-in-ffmpeg
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		errlog("could not open directory '%v'\n", dirPath)
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
		if ext != catName && ext == ".mp3" || ext == ".flac" || ext == ".wav" || ext == ".webm" {
				args = append(args, "-i")
				args = append(args, fmt.Sprintf("%v/%v", dirPath, eName))
				count++
		} else {
			infolog("skipping file '%v'\n", eName)
			continue
		}
	}
	if count == 0 {
		return
	}
	args = append(args, "-filter_complex")
	args = append(args, fmt.Sprintf("concat=n=%v:v=0:a=1[a]", count))
	args = append(args, "-map")
	args = append(args, "[a]")
	args = append(args, "-codec:a")
	args = append(args, "libmp3lame")
	args = append(args, "-b:a")
	args = append(args, "256k")
	args = append(args, catName)
	cmd := exec.Command("ffmpeg", args...)
	debuglog("cmd: %v\n", cmd.String())
	err = cmd.Run()
	if err != nil {
		errlog("encountered error while running command:\n\tcmd - %v\n\terr - %v\n", cmd.String(), err)
		return
	}
}

// parse is a function that parses the directory to be served by the file server.
// It takes audio and video files and adds html element to the static.Html value.
// If -ffmpeg is specified, then it recursively concats each level audio files to a new mp3 file to the directory being served.
func (st *static) parse(dir *string) {
	entries, err := os.ReadDir(*dir)
	if err != nil {
		errlog("could not open directory '%v'\n", *dir)
		return
	}

	var reReadDir bool
	var once sync.Once
	if *ffmpeg {
		// collect filenames, so we could ignore already concatenated directories. 
		fnames := make(map[string]struct{})
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			fnames[e.Name()] = struct{}{}
		}
		// concat audio files
		var wg sync.WaitGroup
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			_, ok := fnames[e.Name()+".mp3"]
			if ok {
				continue
			}
			wg.Add(1)
			once.Do(func() {reReadDir = true})
			go func() {
				defer wg.Done()
				concatAudio(*dir, *dir + "/" + e.Name())
			}()
		}
		wg.Wait()
	}
	// if we created any mp3 files, re-read the directory being served to enable serving also the new mp3 files
	if reReadDir{
		entries, err = os.ReadDir(*dir)
		if err != nil {
			errlog("could not open directory '%v'\n", *dir)
			return
		}
	}
	// compose html to be served
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		eName := e.Name()
		ext := filepath.Ext(eName)
		var ehtml string
		if ext == ".mp3" || ext == ".flac" || ext == ".wav" || ext == ".webm" {
				ehtml = music
		} else if ext == ".mp4" || ext == ".mkv" {
				ehtml = video
		} else {
			infolog("skipping file '%v'\n", eName)
			continue
		}
		ehtml = strings.ReplaceAll(ehtml, "{{.file}}", eName)
		st.Html += ehtml
		infolog("handled file %v\n", eName)
	}
}

func addHeaders(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	}
}

func main() {
	port := flag.String("p", ":8080", "port where fileserver will be served")
	dir = flag.String("d", "", "directory to serve")
	verbose = flag.Bool("v", false, "verbose application")
	vverbose = flag.Bool("vv", false, "very verbose application")
	ffmpeg = flag.Bool("ffmpeg", false, "concat media files recursively from each level to the directory being served; EXTERNAL DEPENDENCY ON FFMPEG")
	flag.Parse()
	if *dir == "" {
		errlog("-d not provided")
		return
	}
	st := New()
	http.Handle("/", addHeaders(http.FileServer(http.Dir(*dir))))
	http.Handle("/st", st)
	infolog("Serving %v at %v\n", *dir, *port)
	http.ListenAndServe(*port, nil)
}
