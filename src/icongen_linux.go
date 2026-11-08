//go:build !windows

package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

func downscale(src *Canvas, w int) *Canvas {
	dst := NewCanvas(w, w)
	scale := float64(src.W) / float64(w)
	for y := 0; y < w; y++ {
		y0 := int(float64(y) * scale)
		y1 := int(float64(y+1) * scale)
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < w; x++ {
			x0 := int(float64(x) * scale)
			x1 := int(float64(x+1) * scale)
			if x1 <= x0 {
				x1 = x0 + 1
			}
			var r, g, b, a float64
			n := 0
			for yy := y0; yy < y1 && yy < src.H; yy++ {
				for xx := x0; xx < x1 && xx < src.W; xx++ {
					i := (yy*src.W + xx) * 4
					aa := float64(src.Pix[i+3]) / 255
					r += float64(src.Pix[i]) * aa
					g += float64(src.Pix[i+1]) * aa
					b += float64(src.Pix[i+2]) * aa
					a += aa
					n++
				}
			}
			if n == 0 || a <= 0 {
				continue
			}
			j := (y*w + x) * 4
			dst.Pix[j] = uint8(r/a + 0.5)
			dst.Pix[j+1] = uint8(g/a + 0.5)
			dst.Pix[j+2] = uint8(b/a + 0.5)
			dst.Pix[j+3] = uint8(a/float64(n)*255 + 0.5)
		}
	}
	return dst
}

func main() {
	os.MkdirAll("/tmp/icon", 0755)

	base := NewCanvas(512, 512)
	layer := NewCanvas(catLayerW, catLayerH)
	PaintCat(layer, CatPose{Tail: 0.9, GazeX: 0.15, GazeY: 0.05, Breath: 0.6}, themes[0])
	// head + chest crop, centred
	base.BlitTransform(layer, 256, 260, 150, 102, 0, 2.45, 2.45)

	sizes := []int{16, 32, 48, 64, 128, 256}
	for _, n := range sizes {
		c := downscale(base, n)
		f, err := os.Create(fmt.Sprintf("/tmp/icon/%d.rgba", n))
		if err != nil {
			panic(err)
		}
		binary.Write(f, binary.LittleEndian, c.Pix)
		f.Close()
	}
}
