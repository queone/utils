package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// palette
var (
	colBg      = color.RGBA{0x14, 0x14, 0x14, 0xff}
	colBorder  = color.RGBA{0x3c, 0x3c, 0x3c, 0xff}
	colText    = color.RGBA{0xcc, 0xcc, 0xcc, 0xff}
	colWinText = color.RGBA{0x0d, 0x0d, 0x0d, 0xff} // dark text on green bg
	colWinBg   = color.RGBA{0x3a, 0xd5, 0x68, 0xff} // bright green background
	colRing    = color.RGBA{0x3c, 0x3c, 0x3c, 0xff}
	colSpoke   = color.RGBA{0x44, 0xee, 0x77, 0xff}
)

const (
	cellW      = 54
	cellH      = 36
	bw         = 1   // border width in px
	gridGap    = 18  // gap between grids in px
	circleSize = 450 // circle geometry image is always this square
	fontSize   = 13.5
)

func isITerm2() bool {
	return os.Getenv("TERM_PROGRAM") == "iTerm.app"
}

func emitITerm2Image(img image.Image) {
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	fmt.Printf("\033]1337;File=inline=1;preserveAspectRatio=1:%s\a\n", b64)
}

func newFace(size float64) (font.Face, error) {
	f, err := opentype.Parse(gomono.TTF)
	if err != nil {
		return nil, err
	}
	return opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72})
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for dy := range h {
		for dx := range w {
			img.SetRGBA(x+dx, y+dy, c)
		}
	}
}

// textAt draws s centered at pixel (cx, cy).
func textAt(img *image.RGBA, face font.Face, s string, cx, cy int, c color.Color) {
	m := face.Metrics()
	adv := font.MeasureString(face, s)
	px := cx - adv.Round()/2
	py := cy + (m.Ascent.Round()-m.Descent.Round())/2
	d := &font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face, Dot: fixed.P(px, py)}
	d.DrawString(s)
}

// drawLine draws a 2×2-pixel-wide line from (x0,y0) to (x1,y1).
func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx, dy := float64(x1-x0), float64(y1-y0)
	steps := int(math.Max(math.Abs(dx), math.Abs(dy))) + 1
	for i := range steps {
		t := float64(i) / float64(max(steps-1, 1))
		x := int(math.Round(float64(x0) + t*dx))
		y := int(math.Round(float64(y0) + t*dy))
		img.SetRGBA(x, y, c)
		img.SetRGBA(x+1, y, c)
		img.SetRGBA(x, y+1, c)
		img.SetRGBA(x+1, y+1, c)
	}
}

// drawRing draws a ~2px-thick circle.
func drawRing(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	b := img.Bounds()
	fr := float64(r)
	for y := max(b.Min.Y, cy-r-2); y <= min(b.Max.Y-1, cy+r+2); y++ {
		for x := max(b.Min.X, cx-r-2); x <= min(b.Max.X-1, cx+r+2); x++ {
			d := math.Sqrt(float64((x-cx)*(x-cx) + (y-cy)*(y-cy)))
			if d >= fr-0.5 && d < fr+1.5 {
				img.SetRGBA(x, y, c)
			}
		}
	}
}

// renderBoxGrid renders a rows×cols numbered grid (numbers start at 1).
func renderBoxGrid(rows, cols int, hl map[int]bool, face font.Face) *image.RGBA {
	imgW := cols*(cellW+bw) + bw
	imgH := rows*(cellH+bw) + bw
	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	// Fill with border color; cell interiors will overwrite with bg.
	draw.Draw(img, img.Bounds(), &image.Uniform{colBorder}, image.Point{}, draw.Src)
	for r := range rows {
		for c := range cols {
			n := r*cols + c + 1
			cx := bw + c*(cellW+bw)
			cy := bw + r*(cellH+bw)
			bg, tc := colBg, colText
			if hl[n] {
				bg, tc = colWinBg, colWinText
			}
			fillRect(img, cx, cy, cellW, cellH, bg)
			textAt(img, face, fmt.Sprintf("%02d", n), cx+cellW/2, cy+cellH/2, tc)
		}
	}
	return img
}

// renderHexGrid renders the shield-shaped grid.
// Rows 0 and 6 hold 5 cells (offset by 1 column); rows 1–5 hold 7 cells.
// Per-row border strips are drawn before cell interiors so the "corners" of
// the short rows remain as background, giving the shield silhouette.
func renderHexGrid(hl map[int]bool, face font.Face) *image.RGBA {
	imgW := 7*(cellW+bw) + bw
	imgH := 7*(cellH+bw) + bw
	img := image.NewRGBA(image.Rect(0, 0, imgW, imgH))
	draw.Draw(img, img.Bounds(), &image.Uniform{colBg}, image.Point{}, draw.Src)

	type rowDef struct {
		nums []int
		off  int // column offset (0 or 1)
	}
	layout := []rowDef{
		{[]int{1, 2, 3, 4, 5}, 1},
		{[]int{6, 7, 8, 9, 10, 11, 12}, 0},
		{[]int{13, 14, 15, 16, 17, 18, 19}, 0},
		{[]int{20, 21, 22, 23, 24, 25, 26}, 0},
		{[]int{27, 28, 29, 30, 31, 32, 33}, 0},
		{[]int{34, 35, 36, 37, 38, 39, 40}, 0},
		{[]int{41, 42, 43, 44, 45}, 1},
	}
	for r, row := range layout {
		// Draw border strip spanning this row's cells.
		x0 := row.off * (cellW + bw)
		rowW := len(row.nums)*(cellW+bw) + bw
		y0 := r * (cellH + bw)
		fillRect(img, x0, y0, rowW, cellH+2*bw, colBorder)
		// Draw cell interiors.
		for ci, n := range row.nums {
			cx := bw + (ci+row.off)*(cellW+bw)
			cy := bw + r*(cellH+bw)
			bg, tc := colBg, colText
			if hl[n] {
				bg, tc = colWinBg, colWinText
			}
			fillRect(img, cx, cy, cellW, cellH, bg)
			textAt(img, face, fmt.Sprintf("%02d", n), cx+cellW/2, cy+cellH/2, tc)
		}
	}
	return img
}

// renderCircle places numbers 1–45 evenly around a circle (8° apart,
// starting at 12 o'clock). Winning numbers get a green spoke from the center.
func renderCircle(hl map[int]bool, face font.Face) *image.RGBA {
	size := circleSize
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), &image.Uniform{colBg}, image.Point{}, draw.Src)

	cx, cy := size/2, size/2
	ringR := int(float64(size) * 0.38) // radius of visible ring
	numR := int(float64(size) * 0.44)  // radius at which numbers are centered
	spokeR := ringR - 2                // spokes end just inside the ring

	drawRing(img, cx, cy, ringR, colRing)

	for n := 1; n <= 45; n++ {
		theta := (-90.0 + float64(n-1)*8.0) * math.Pi / 180.0
		cosT, sinT := math.Cos(theta), math.Sin(theta)
		nx := cx + int(math.Round(float64(numR)*cosT))
		ny := cy + int(math.Round(float64(numR)*sinT))
		tc := colText
		if hl[n] {
			sx := cx + int(math.Round(float64(spokeR)*cosT))
			sy := cy + int(math.Round(float64(spokeR)*sinT))
			drawLine(img, cx, cy, sx, sy, colSpoke)
			tc = colWinText
			// Green background patch behind the number label.
			m := face.Metrics()
			tw := font.MeasureString(face, "00").Round()
			th := m.Ascent.Round() + m.Descent.Round()
			pad := 3
			fillRect(img, nx-tw/2-pad, ny-th/2-pad, tw+2*pad, th+2*pad, colWinBg)
		}
		textAt(img, face, fmt.Sprintf("%02d", n), nx, ny, tc)
	}
	return img
}

// renderGeometries composes all four geometries into one wide image.
func renderGeometries(winners []int) (*image.RGBA, error) {
	face, err := newFace(fontSize)
	if err != nil {
		return nil, err
	}
	defer face.Close()

	hl := make(map[int]bool)
	for _, n := range winners {
		hl[n] = true
	}

	grids := []image.Image{
		renderBoxGrid(5, 9, hl, face),
		renderBoxGrid(9, 5, hl, face),
		renderHexGrid(hl, face),
		renderCircle(hl, face),
	}

	maxH := 0
	totalW := gridGap * (len(grids) - 1)
	for _, g := range grids {
		if g.Bounds().Dy() > maxH {
			maxH = g.Bounds().Dy()
		}
		totalW += g.Bounds().Dx()
	}

	out := image.NewRGBA(image.Rect(0, 0, totalW, maxH))
	draw.Draw(out, out.Bounds(), &image.Uniform{colBg}, image.Point{}, draw.Src)

	x := 0
	for _, g := range grids {
		yOff := (maxH - g.Bounds().Dy()) / 2
		r := image.Rect(x, yOff, x+g.Bounds().Dx(), yOff+g.Bounds().Dy())
		draw.Draw(out, r, g, image.Point{}, draw.Src)
		x += g.Bounds().Dx() + gridGap
	}
	return out, nil
}

func displayGeometriesImage(winners []int, indent string) {
	img, err := renderGeometries(winners)
	if err != nil {
		return
	}
	fmt.Print(indent)
	emitITerm2Image(img)
}
