## vtrim
Trim a video — from the left, right, middle, or by cutting a middle section out — by driving `ffmpeg`.

### Why?
Trimming a video with raw `ffmpeg` means remembering seek-flag ordering, codec flags, and the keyframe-vs-accuracy trade-off, then hand-naming the output. `vtrim` wraps the common cases in four terse subcommands, validates the trim against the source up front, names the output for you, and prints a before/after summary.

```bash
vtrim l 8:31 april1.mp4
FILE    NAME          SIZE         DURATION
input   april1.mp4    104,857,600  00:42:10
output  april_1.mp4    82,330,112  00:33:39
```

### Timestamps
- `MM:SS` is the default (e.g. `8:31` = 8 min 31 s).
- A bare integer is whole seconds (e.g. `90` = 1 min 30 s).
- `HH:MM:SS` (e.g. `1:08:31`) is accepted only when the source is longer than one hour.
- A trim that reaches or exceeds the source duration errors out.

### Output naming
The output goes next to the input. Its name is the input's with `_` inserted before the trailing digits (`april1.mp4` → `april_1.mp4`), or `_1` appended when the name has no trailing digit (`clip.mp4` → `clip_1.mp4`). `vtrim` refuses to overwrite an existing output.

### Speed vs. accuracy
By default `vtrim` stream-copies (`-c copy`) — fast and lossless, but the start snaps to the nearest preceding keyframe. Pass `-a, --accurate` for a frame-accurate result; this re-encodes (`libx264`/`aac`) and is slower.

### Requirements
Requires `ffmpeg` and `ffprobe` on your `PATH`:

```bash
brew install ffmpeg
```

### Getting Started
This utility is part of a collection of Go utilities. To compile and install follow the **Getting Started** instructions at the [utils repo](https://github.com/queone/utils).

### Usage

```bash
$ vtrim
vtrim v0.1.0
Trim a video by driving ffmpeg — https://github.com/queone/utils/blob/main/cmd/vtrim/README.md
Overview
  Trims a video to a new file. Timestamps are MM:SS by default (e.g. 8:31); a bare
  integer is whole seconds (e.g. 90); HH:MM:SS is allowed only when the source is
  longer than one hour. The trim cannot exceed the source duration. The output is
  named by inserting '_' before the input's trailing digits (april1.mp4 -> april_1.mp4),
  or appending '_1' when the name has no trailing digit; vtrim refuses to overwrite an
  existing output.

Usage
  vtrim <mode> [-a] <args> <input>

Modes
  l START         Left   — keep from START to the end       (vtrim l 8:31 april1.mp4)
  r END           Right  — keep from the start to END        (vtrim r 8:31 april1.mp4)
  m START END     Middle — keep the segment START..END       (vtrim m 1:00 8:31 april1.mp4)
  x START END     Cut    — remove START..END, join the rest  (vtrim x 1:00 8:31 april1.mp4)

Options
  -a, --accurate  Frame-accurate trim via re-encode (default is a fast, lossless,
                  keyframe-snapped stream copy)
  -v, --version   Show this help message and exit
  -h, -?, --help  Show this help message and exit

Notes
  Requires ffmpeg and ffprobe on PATH (brew install ffmpeg).
```
