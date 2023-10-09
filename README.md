# goava

<img src=./resources/goava.jpg width=15% height=15%/>

**WIP** Local audio/video streaming thingy written in Go.

## Getting started

Simple file server to serve.

```sh
go run main.go -simple -dir=<dir to serve>
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

* MKV to MP4

```sh
ffmpeg \
	-i <video file>.mkv  \
	-codec copy <video file>.mp4 
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
