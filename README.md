# av

Local audio/video streaming thingy written in Go.

## Synopsis

TBD

## Getting started

To start up the web server, you can run the shell script

```sh
./run.sh
```

or run

```sh
go run main.go -d=<directory to be served> [-ffmpeg]
```

## Useful ffmpeg commands

* Break a/v files to HLS (HTTP Live Stream) format

```sh
ffmpeg \
	-i <audio/video file>  \
	-c:a libmp3lame  \
	-b:a 128k  \
	-map 0:0  \
	-f segment  \
	-segment_time 10  \
	-segment_list outputlist.m3u8  \
	-segment_format mpegts output%03d.ts
```

* MKV to MP4 w/o re-encoding

```sh
ffmpeg \
	-i <video file>.mkv  \
	-codec copy <video file>.mp4 
```

* MKV to MP4 w/ re-encoding (slow for movies)

```sh
ffmpeg -i <video>.mkv -c:v mpeg4 -c:a libvorbis <output>.mp4
```

* Add subtitles to MP4

```sh
ffmpeg \
	-i <video file>.mp4  \
	-i <subtitle>.srt  \
	-c copy \
	-c:s mov_text \
	outfile.mp4
```

* combine (different) audio files

```sh
ffmpeg \
	-i "1.mp3" \
	-i "2.wav" \
	-i "3.flac" \
	-filter_complex 'concat=n=${x}:v=0:a=1[a]' \ # ${x}, where x is nr of input, ie how many '-i' flags
	-map '[a]' \
	-codec:a libmp3lame \
	-b:a 256k \
	output.mp3
```
