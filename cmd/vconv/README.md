## vconv
Convert a video, typically WebM, to an H.264/AAC MP4 by driving `ffmpeg`.

### Why?
Screen recorders and browsers hand out WebM files that many players and editors will not open. `vconv` re-encodes one file with the flags that play everywhere and names the output for you.

```bash
vconv talk.webm
==> Converting talk.webm
FILE    NAME        SIZE        DURATION
input   talk.webm   48,213,004  00:12:40
output  talk.mp4    39,880,112  00:12:40
```

### Usage

```bash
vconv [flags] INPUT
```

`INPUT` is any video `ffmpeg` can read. The output is `INPUT` with an `.mp4` extension, written next to it. `vconv` refuses to overwrite an existing file and refuses an input that is already `.mp4`.

### Encoding
Video is H.264 at CRF 23 with the `fast` preset; audio is AAC. These are the same flags the retired `webm2mp4.sh` script used.

### Requirements
Requires `ffmpeg` and `ffprobe` on your `PATH`:

```bash
brew install ffmpeg
```

### See also
- [`vshrink`](../vshrink/README.md) — re-encode an MP4 at a high compression level to cut its size.
- [`vkeep`](../vkeep/README.md), [`vdrop`](../vdrop/README.md), [`vjoin`](../vjoin/README.md) — keep, drop, or join sections.

### Getting Started
This utility is part of a collection of Go utilities. To compile and install follow the **Getting Started** instructions at the [gkit repo](https://github.com/queone/gkit).
