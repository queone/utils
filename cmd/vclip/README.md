## vclip
Keep a section of a video — extract `START..END` to a new file — by driving `ffmpeg`.

### Why?
Pulling a clip out of a video with raw `ffmpeg` means remembering seek-flag ordering, codec flags, and the keyframe-vs-accuracy trade-off, then hand-naming the output. `vclip` takes a start and (optionally) an end, validates the range against the source up front, names the output for you, and prints a before/after summary. The mental model is simple: **`vclip` keeps the part you want.**

```bash
vclip 4:13 SOURCE1.mp4
FILE    NAME           SIZE         DURATION
input   SOURCE1.mp4    104,857,600  00:42:10
output  SOURCE_1.mp4    82,330,112  00:37:57
```

### Usage

```bash
vclip START [END] [-a] <input>
```

`START` and `END` mark the section to keep. `END` is optional: omit it (or pass the literal `end`) to keep through to the source end.

### Examples

```bash
vclip 0 SOURCE1.mp4          # keep the whole file (a straight copy)
vclip 4:13 SOURCE1.mp4       # keep from 4:13 to the end
vclip 0 8:31 SOURCE1.mp4     # keep from the start to 8:31
vclip 1:00 8:31 SOURCE1.mp4  # keep the segment 1:00..8:31
```

### Timestamps
- `MM:SS` is the default (e.g. `8:31` = 8 min 31 s).
- A bare integer is whole seconds (e.g. `90` = 1 min 30 s).
- `HH:MM:SS` (e.g. `1:08:31`) is accepted only when the source is longer than one hour.
- `END` may be omitted or given as the literal `end` to mean "to the source end".
- A range that exceeds the source duration errors out.

### Output naming
The output goes next to the input. Its name is the input's with `_` inserted before the trailing digits (`SOURCE1.mp4` → `SOURCE_1.mp4`), or `_1` appended when the name has no trailing digit (`clip.mp4` → `clip_1.mp4`). `vclip` refuses to overwrite an existing output.

### Speed vs. accuracy
By default `vclip` stream-copies (`-c copy`) — fast and lossless, but the start snaps to the nearest preceding keyframe. Pass `-a, --accurate` for a frame-accurate result; this re-encodes (`libx264`/`aac`) and is slower.

### Requirements
Requires `ffmpeg` and `ffprobe` on your `PATH`:

```bash
brew install ffmpeg
```

### See also
- [`vcut`](../vcut/README.md) — the counterpart that **removes** a section and joins the remainder.

### Getting Started
This utility is part of a collection of Go utilities. To compile and install follow the **Getting Started** instructions at the [utils repo](https://github.com/queone/utils).
