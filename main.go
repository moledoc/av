package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	ffmpeg   *bool
	verbose  *bool
	vverbose *bool
	dir      *string
)

type audvid struct {
	html string
}

func new() audvid {
	return audvid{
		html: "<!DOCTYPE html> <html> <head> <style> span { display: flex; align-items: center;} </style> </head> <body>",
	}
}

const music string = "<span><audio controls loop src=\"{{.file}}\" type=\"audio/mpeg\"> </audio><font size=\"5\"> - {{.file}}</font></span><br>"

const video string = "<span><video controls width=\"320\" height=\"240\" src=\"{{.file}}\" type=\"video/mp4\"></video><font size=\"5\"> - {{.file}}</font></span><br>"

func (av audvid) String() string {
	return av.html + "</body><br></html>"
}

func (av audvid) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	av.parse(dir)
	fmt.Fprintf(w, av.String())
}

func errlog(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "[ERROR]: "+format+"\n", a...)
}

func warnlog(format string, a ...any) {
	if *verbose || *vverbose {
		fmt.Fprintf(os.Stdout, "[WARNING]: "+format+"\n", a...)
	}
}

func infolog(format string, a ...any) {
	fmt.Fprintf(os.Stdout, "[INFO]: "+format+"\n", a...)
}

func debuglog(format string, a ...any) {
	if *vverbose {
		fmt.Fprintf(os.Stdout, "[DEBUG]: "+format+"\n", a...)
	}
}

// concatAudio is a func that follows directories recursively and all same level audio files are concatenated to a single mp3 file at the parent directory, which is the directory served by the web server.
// NOTE: there is an external dependency on 'ffmpeg'.
func concatAudio(parentDir string, fnames map[string]struct{}, dirPath string) {
	// concat audio files - https://superuser.com/questions/809623/how-to-join-audio-files-of-different-formats-in-ffmpeg
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		errlog("could not open directory '%v'", dirPath)
		return
	}
	infolog("concatenating '%v'", dirPath)
	dirElems := strings.Split(dirPath, "/")
	leafDir := dirElems[len(dirElems)-1]
	catName := parentDir + "/" + leafDir + ".mp3"
	var args []string
	var count uint8
	var wg sync.WaitGroup
	for _, e := range entries {
		eName := e.Name()
		if e.IsDir() && *ffmpeg {
			// check if dir already concated
			_, ok := fnames[e.Name()+".mp3"]
			debuglog("has dir '%v' been concatenated: %v", e.Name(), ok)
			if ok {
				debuglog("skipping %v", e.Name())
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				concatAudio(parentDir, fnames, dirPath+"/"+eName)
			}()
			continue
		}
		wg.Wait()

		// compose ffmpeg command
		ext := filepath.Ext(eName)
		if ext != catName && ext == ".mp3" || ext == ".flac" || ext == ".wav" || ext == ".webm" {
			args = append(args, "-i")
			args = append(args, fmt.Sprintf("%v/%v", dirPath, eName))
			count++
		} else {
			infolog("skipping file '%v'", eName)
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
	debuglog("cmd: %v", cmd.String())
	err = cmd.Run()
	if err != nil {
		errlog("encountered error while running command:\n\tcmd - %v\n\terr - %v", cmd.String(), err)
	}
}

func ifFfmpeg(dir string, entries []os.DirEntry) bool {
	// collect filenames, so we could ignore already concatenated directories.
	var once sync.Once
	reReadDir := false
	fnames := make(map[string]struct{})
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fnames[e.Name()] = struct{}{}
	}
	// concat audio files
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		_, ok := fnames[e.Name()+".mp3"]
		debuglog("has dir '%v' been concatenated: %v", e.Name(), ok)
		if ok {
			debuglog("skipping", e.Name())
			continue
		}
		once.Do(func() { reReadDir = true })
		concatAudio(dir, fnames, dir+"/"+e.Name())
	}
	return reReadDir
}

// parse is a function that parses the directory to be served by the file server.
// It takes audio and video files and adds html element to the audvid.html value.
// If -ffmpeg is specified, then it recursively concats each level audio files to a new mp3 file to the directory being served.
func (av *audvid) parse(dir *string) {
	entries, err := os.ReadDir(*dir)
	if err != nil {
		errlog("could not open directory '%v'", *dir)
		return
	}

	if *ffmpeg {
		// if we created any mp3 files, re-read the directory being served to enable serving also the new mp3 files
		if ifFfmpeg(*dir, entries) {
			entries, err = os.ReadDir(*dir)
			if err != nil {
				errlog("could not open directory '%v'", *dir)
				return
			}
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
			infolog("skipping file '%v'", eName)
			continue
		}
		ehtml = strings.ReplaceAll(ehtml, "{{.file}}", eName)
		av.html += ehtml
		infolog("handled file '%v'", eName)
	}
}

func addHeaders(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h.ServeHTTP(w, r)
	}
}

// getLocalIP returns the non loopback local IP of the host
func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ""
	}
	for _, address := range addrs {
		// check the address type and if it is not a loopback the display it
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}

func printHelp() {
	fmt.Printf("%v -d <dir> [-p port {8080}] [-v] [-vv] [-ffmpeg]\n", os.Args[0])
	flag.PrintDefaults()
}

func main() {
	help := flag.Bool("h", false, "this help")
	port := flag.String("p", "8080", "port where fileserver will be served")
	dir = flag.String("d", "", "directory to serve")
	verbose = flag.Bool("v", false, "verbose application")
	vverbose = flag.Bool("vv", false, "very verbose application")
	ffmpeg = flag.Bool("ffmpeg", false, "concat media files recursively from each level to the directory being served; EXTERNAL DEPENDENCY ON FFMPEG")
	flag.Parse()
	if *help {
		printHelp()
		return
	}
	if *dir == "" {
		errlog("-d not provided")
		printHelp()
		return
	}
	if *vverbose {
		*verbose = true
	}
	av := new()
	http.Handle("/", addHeaders(http.FileServer(http.Dir(*dir))))
	http.Handle("/av", av)
	ip := getLocalIP()
	infolog("Serving '%v' at 'http://%v:%v/av'", *dir, ip, *port)
	http.ListenAndServe(":"+*port, nil)
}
