## vcut
Remove a section of a video — cut `START..END` and join the remainder — by driving `ffmpeg`.

### Why?
Cutting a section out of a video with raw `ffmpeg` means splitting the source, re-listing the pieces, and concatenating them — or wrestling with a `filter_complex` graph for a frame-accurate result — then hand-naming the output. `vcut` takes a start and (optionally) an end, validates the range against the source up front, joins the surrounding parts, names the output for you, and prints a before/after summary. The mental model is simple: **`vcut` removes the part you don't want.**

```bash
vcut 1:00 8:31 SOURCE1.mp4
FILE    NAME           SIZE         DURATION
input   SOURCE1.mp4    104,857,600  00:42:10
output  SOURCE_1.mp4    86,212,000  00:34:39
```

### Usage

```bash
vcut START [END] [-a] <input>
```

`START` and `END` mark the section to remove. `END` is optional: omit it (or pass the literal `end`) to remove through to the source end.

### Examples

```bash
vcut 4:13 SOURCE1.mp4        # remove from 4:13 to the end (keep the start)
vcut 0 8:31 SOURCE1.mp4      # remove from the start to 8:31 (keep the rest)
vcut 1:00 8:31 SOURCE1.mp4   # remove the segment 1:00..8:31, join the rest
```

### Timestamps
- `MM:SS` is the default (e.g. `8:31` = 8 min 31 s).
- A bare integer is whole seconds (e.g. `90` = 1 min 30 s).
- `HH:MM:SS` (e.g. `1:08:31`) is accepted only when the source is longer than one hour.
- `END` may be omitted or given as the literal `end` to mean "to the source end".
- A range that exceeds the source duration errors out, and a cut cannot remove the entire video.

### Output naming
The output goes next to the input. Its name is the input's with `_` inserted before the trailing digits (`SOURCE1.mp4` → `SOURCE_1.mp4`), or `_1` appended when the name has no trailing digit (`clip.mp4` → `clip_1.mp4`). `vcut` refuses to overwrite an existing output.

### Speed vs. accuracy
By default `vcut` stream-copies (`-c copy`) the surrounding segments and concatenates them — fast and lossless, but each segment boundary snaps to the nearest preceding keyframe. Pass `-a, --accurate` for a frame-accurate result; this re-encodes (`libx264`/`aac`) through a filter graph and is slower.

### Requirements
Requires `ffmpeg` and `ffprobe` on your `PATH`:

```bash
brew install ffmpeg
```

### See also
- [`vclip`](../vclip/README.md) — the counterpart that **keeps** a section instead of removing it.

### Getting Started
This utility is part of a collection of Go utilities. To compile and install follow the **Getting Started** instructions at the [utils repo](https://github.com/queone/utils).
