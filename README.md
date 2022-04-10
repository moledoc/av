## go-av

Audio/Video streaming to http server.

## Dependencies

* ffmpeg
* go

## Running

Break a/v files to HLS (HTTP Live Stream) format

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

## MKV to MP4

```sh
ffmpeg \
	-i <video file>.mkv  \
	-codec copy <video file>.mp4 
```

## Add subtitles to MP4

```sh
ffmpeg \
	-i <video file>.mp4  \
	-i <subtitle>.srt  \
	-c copy \
	-c:s mov_text \
	outfile.mp4
```
