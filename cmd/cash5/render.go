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
	colText    = color.RGBA{0xcc, 0xcc, 0xcc, 0xff}
	colWinText = color.RGBA{0x0d, 0x0d, 0x0d, 0xff} // dark text on green bg
	colWinBg   = color.RGBA{0x3a, 0xd5, 0x68, 0xff} // bright green background
	colRing    = color.RGBA{0x3c, 0x3c, 0x3c, 0xff}
	colSpoke   = color.RGBA{0x44, 0xee, 0x77, 0xff}
)

const (
	circleSize = 450 // circle geometry image is always this square
	fontSize   = 13.5
)

// isITerm2 reports whether the current terminal is iTerm2.
func isITerm2() bool {
	return os.Getenv("TERM_PROGRAM") == "iTerm.app"
}

// emitITerm2Image writes an inline image escape sequence for iTerm2.
func emitITerm2Image(img image.Image) {
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())
	fmt.Printf("\033]1337;File=inline=1;preserveAspectRatio=1:%s\a\n", b64)
}

// newFace returns a gomono font face at the given point size.
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

// displayCircleImage renders the winning-circle inline image for iTerm2.
// No-op outside iTerm2 — callers gate on isITerm2() before calling.
func displayCircleImage(winners []int, indent string) {
	face, err := newFace(fontSize)
	if err != nil {
		return
	}
	defer face.Close()

	hl := make(map[int]bool)
	for _, n := range winners {
		hl[n] = true
	}

	img := renderCircle(hl, face)
	fmt.Print(indent)
	emitITerm2Image(img)
}
