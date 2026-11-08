package main

import "math"

// ---------------------------------------------------------------- colors

type Col struct {
	R, G, B uint8
	A       float64
}

func rgb(r, g, b uint8) Col      { return Col{r, g, b, 1} }
func (c Col) alpha(a float64) Col { c.A *= a; return c }

type Vec struct{ X, Y float64 }

// ---------------------------------------------------------------- canvas

// Canvas holds straight-alpha RGBA pixels, row major.
type Canvas struct {
	W, H int
	Pix  []uint8
}

func NewCanvas(w, h int) *Canvas {
	return &Canvas{W: w, H: h, Pix: make([]uint8, w*h*4)}
}

func (c *Canvas) Clear() {
	for i := range c.Pix {
		c.Pix[i] = 0
	}
}

func (c *Canvas) blendPixel(x, y int, col Col, cov float64) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H || cov <= 0 {
		return
	}
	a := col.A * cov
	if a <= 0.0009 {
		return
	}
	if a > 1 {
		a = 1
	}
	i := (y*c.W + x) * 4
	da := float64(c.Pix[i+3]) / 255
	oa := a + da*(1-a)
	if oa <= 0 {
		return
	}
	inv := 1 / oa
	k := da * (1 - a)
	c.Pix[i] = uint8((float64(col.R)*a+float64(c.Pix[i])*k)*inv + 0.5)
	c.Pix[i+1] = uint8((float64(col.G)*a+float64(c.Pix[i+1])*k)*inv + 0.5)
	c.Pix[i+2] = uint8((float64(col.B)*a+float64(c.Pix[i+2])*k)*inv + 0.5)
	c.Pix[i+3] = uint8(oa*255 + 0.5)
}

// ---------------------------------------------------------------- shapes

func (c *Canvas) FillEllipse(cx, cy, rx, ry float64, col Col) {
	if rx <= 0 || ry <= 0 || col.A <= 0 {
		return
	}
	soft := math.Min(rx, ry)
	if soft < 1 {
		soft = 1
	}
	x0 := int(math.Floor(cx - rx - 1))
	x1 := int(math.Ceil(cx + rx + 1))
	y0 := int(math.Floor(cy - ry - 1))
	y1 := int(math.Ceil(cy + ry + 1))
	irx, iry := 1/rx, 1/ry
	for y := y0; y <= y1; y++ {
		dy := (float64(y) + 0.5 - cy) * iry
		for x := x0; x <= x1; x++ {
			dx := (float64(x) + 0.5 - cx) * irx
			s := math.Hypot(dx, dy)
			cov := 0.5 - (s-1)*soft
			if cov <= 0 {
				continue
			}
			if cov > 1 {
				cov = 1
			}
			c.blendPixel(x, y, col, cov)
		}
	}
}

func (c *Canvas) FillRect(x0, y0, x1, y1 float64, col Col) {
	c.FillRoundRect(x0, y0, x1, y1, 0, col)
}

func (c *Canvas) FillRoundRect(x0, y0, x1, y1, r float64, col Col) {
	if col.A <= 0 {
		return
	}
	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	if r > (x1-x0)/2 {
		r = (x1 - x0) / 2
	}
	if r > (y1-y0)/2 {
		r = (y1 - y0) / 2
	}
	ix0 := int(math.Floor(x0 - 1))
	ix1 := int(math.Ceil(x1 + 1))
	iy0 := int(math.Floor(y0 - 1))
	iy1 := int(math.Ceil(y1 + 1))
	for y := iy0; y <= iy1; y++ {
		py := float64(y) + 0.5
		for x := ix0; x <= ix1; x++ {
			px := float64(x) + 0.5
			qx := math.Max(x0+r, math.Min(px, x1-r))
			qy := math.Max(y0+r, math.Min(py, y1-r))
			d := math.Hypot(px-qx, py-qy) - r
			cov := 0.5 - d
			if cov <= 0 {
				continue
			}
			if cov > 1 {
				cov = 1
			}
			c.blendPixel(x, y, col, cov)
		}
	}
}

func polygonContains(poly []Vec, x, y float64) bool {
	in := false
	j := len(poly) - 1
	for i := 0; i < len(poly); i++ {
		yi, yj := poly[i].Y, poly[j].Y
		if (yi > y) != (yj > y) {
			xi, xj := poly[i].X, poly[j].X
			if x < (xj-xi)*(y-yi)/(yj-yi)+xi {
				in = !in
			}
		}
		j = i
	}
	return in
}

func (c *Canvas) FillPolygon(poly []Vec, col Col) {
	if len(poly) < 3 || col.A <= 0 {
		return
	}
	minX, maxX := poly[0].X, poly[0].X
	minY, maxY := poly[0].Y, poly[0].Y
	for _, p := range poly {
		minX = math.Min(minX, p.X)
		maxX = math.Max(maxX, p.X)
		minY = math.Min(minY, p.Y)
		maxY = math.Max(maxY, p.Y)
	}
	x0, x1 := int(math.Floor(minX-1)), int(math.Ceil(maxX+1))
	y0, y1 := int(math.Floor(minY-1)), int(math.Ceil(maxY+1))
	const ss = 3
	step := 1.0 / ss
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			hits := 0
			for sy := 0; sy < ss; sy++ {
				for sx := 0; sx < ss; sx++ {
					px := float64(x) + (float64(sx)+0.5)*step
					py := float64(y) + (float64(sy)+0.5)*step
					if polygonContains(poly, px, py) {
						hits++
					}
				}
			}
			if hits > 0 {
				c.blendPixel(x, y, col, float64(hits)/(ss*ss))
			}
		}
	}
}

func (c *Canvas) StrokePath(pts []Vec, width float64, col Col) {
	if len(pts) < 2 || width <= 0 {
		return
	}
	step := math.Max(0.4, width*0.12)
	for i := 0; i < len(pts)-1; i++ {
		a, b := pts[i], pts[i+1]
		d := math.Hypot(b.X-a.X, b.Y-a.Y)
		n := int(math.Ceil(d / step))
		if n < 1 {
			n = 1
		}
		for k := 0; k <= n; k++ {
			t := float64(k) / float64(n)
			c.FillEllipse(a.X+(b.X-a.X)*t, a.Y+(b.Y-a.Y)*t, width/2, width/2, col)
		}
	}
}

func (c *Canvas) TaperedStroke(pts []Vec, w0, w1 float64, col Col) {
	if len(pts) < 2 {
		return
	}
	for i := 0; i < len(pts); i++ {
		t := float64(i) / float64(len(pts)-1)
		w := w0 + (w1-w0)*t
		c.FillEllipse(pts[i].X, pts[i].Y, w/2, w/2, col)
	}
}

func Bezier(p0, p1, p2, p3 Vec, n int) []Vec {
	out := make([]Vec, 0, n+1)
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		u := 1 - t
		x := u*u*u*p0.X + 3*u*u*t*p1.X + 3*u*t*t*p2.X + t*t*t*p3.X
		y := u*u*u*p0.Y + 3*u*u*t*p1.Y + 3*u*t*t*p2.Y + t*t*t*p3.Y
		out = append(out, Vec{x, y})
	}
	return out
}

func Arc(cx, cy, rx, ry, a0, a1 float64, n int) []Vec {
	out := make([]Vec, 0, n+1)
	for i := 0; i <= n; i++ {
		t := float64(i) / float64(n)
		a := a0 + (a1-a0)*t
		out = append(out, Vec{cx + math.Cos(a)*rx, cy + math.Sin(a)*ry})
	}
	return out
}

func (c *Canvas) Heart(cx, cy, size float64, col Col) {
	poly := make([]Vec, 0, 48)
	for i := 0; i < 48; i++ {
		t := -math.Pi/2 + float64(i)/48*math.Pi*2
		x := 16 * math.Pow(math.Sin(t), 3)
		y := -(13*math.Cos(t) - 5*math.Cos(2*t) - 2*math.Cos(3*t) - math.Cos(4*t))
		poly = append(poly, Vec{cx + x*size/16, cy + y*size/16})
	}
	c.FillPolygon(poly, col)
}

// ---------------------------------------------------------------- transform

func (c *Canvas) sample(x, y float64) (Col, float64) {
	if x < 0 || y < 0 || x >= float64(c.W)-1 || y >= float64(c.H)-1 {
		return Col{}, 0
	}
	x0, y0 := int(x), int(y)
	fx, fy := x-float64(x0), y-float64(y0)
	i00 := (y0*c.W + x0) * 4
	i10 := i00 + 4
	i01 := i00 + c.W*4
	i11 := i01 + 4
	w00 := (1 - fx) * (1 - fy)
	w10 := fx * (1 - fy)
	w01 := (1 - fx) * fy
	w11 := fx * fy
	r := float64(c.Pix[i00])*w00 + float64(c.Pix[i10])*w10 + float64(c.Pix[i01])*w01 + float64(c.Pix[i11])*w11
	g := float64(c.Pix[i00+1])*w00 + float64(c.Pix[i10+1])*w10 + float64(c.Pix[i01+1])*w01 + float64(c.Pix[i11+1])*w11
	b := float64(c.Pix[i00+2])*w00 + float64(c.Pix[i10+2])*w10 + float64(c.Pix[i01+2])*w01 + float64(c.Pix[i11+2])*w11
	a := float64(c.Pix[i00+3])*w00 + float64(c.Pix[i10+3])*w10 + float64(c.Pix[i01+3])*w01 + float64(c.Pix[i11+3])*w11
	return Col{uint8(r), uint8(g), uint8(b), a / 255}, a / 255
}

func (c *Canvas) Blit(src *Canvas, dx, dy int) {
	for y := 0; y < src.H; y++ {
		for x := 0; x < src.W; x++ {
			i := (y*src.W + x) * 4
			a := float64(src.Pix[i+3]) / 255
			if a <= 0 {
				continue
			}
			c.blendPixel(dx+x, dy+y, Col{src.Pix[i], src.Pix[i+1], src.Pix[i+2], 1}, a)
		}
	}
}

// BlitTransform places src so that src point (scx,scy) lands on (dcx,dcy).
func (c *Canvas) BlitTransform(src *Canvas, dcx, dcy, scx, scy, angle, sx, sy float64) {
	if sx <= 0 || sy <= 0 {
		return
	}
	if angle == 0 && sx == 1 && sy == 1 {
		c.Blit(src, int(math.Round(dcx-scx)), int(math.Round(dcy-scy)))
		return
	}
	rad := math.Hypot(float64(src.W), float64(src.H)) * 0.6 * math.Max(sx, sy)
	x0 := int(dcx - rad)
	x1 := int(dcx + rad)
	y0 := int(dcy - rad)
	y1 := int(dcy + rad)
	cosA, sinA := math.Cos(angle), math.Sin(angle)
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			px := (float64(x) + 0.5 - dcx) / sx
			py := (float64(y) + 0.5 - dcy) / sy
			lx := cosA*px + sinA*py + scx
			ly := -sinA*px + cosA*py + scy
			if lx < 0 || ly < 0 || lx >= float64(src.W)-1 || ly >= float64(src.H)-1 {
				continue
			}
			col, a := src.sample(lx, ly)
			if a <= 0.002 {
				continue
			}
			c.blendPixel(x, y, Col{col.R, col.G, col.B, 1}, a)
		}
	}
}

// ---------------------------------------------------------------- cat art

type Theme struct {
	Body, Outline, Belly, EarInner, Blush, Eye, Nose, Collar, Bell Col
}

var themes = []Theme{
	{ // apricot
		Body: rgb(255, 217, 168), Outline: rgb(214, 155, 106), Belly: rgb(255, 244, 224),
		EarInner: rgb(255, 178, 195), Blush: rgb(255, 150, 173), Eye: rgb(62, 44, 44),
		Nose: rgb(233, 121, 145), Collar: rgb(255, 138, 158), Bell: rgb(255, 206, 92),
	},
	{ // gray
		Body: rgb(228, 233, 241), Outline: rgb(161, 174, 191), Belly: rgb(250, 252, 255),
		EarInner: rgb(245, 184, 198), Blush: rgb(255, 158, 178), Eye: rgb(55, 60, 78),
		Nose: rgb(236, 130, 152), Collar: rgb(124, 199, 227), Bell: rgb(255, 206, 92),
	},
	{ // pink
		Body: rgb(255, 199, 219), Outline: rgb(226, 143, 172), Belly: rgb(255, 243, 248),
		EarInner: rgb(255, 158, 191), Blush: rgb(255, 128, 160), Eye: rgb(85, 48, 62),
		Nose: rgb(226, 92, 128), Collar: rgb(178, 156, 219), Bell: rgb(255, 209, 102),
	},
	{ // midnight
		Body: rgb(109, 106, 133), Outline: rgb(74, 71, 96), Belly: rgb(138, 134, 163),
		EarInner: rgb(176, 140, 168), Blush: rgb(196, 112, 141), Eye: rgb(31, 29, 42),
		Nose: rgb(230, 128, 158), Collar: rgb(78, 205, 196), Bell: rgb(255, 209, 102),
	},
}

// cat layer geometry
const (
	catLayerW = 280
	catLayerH = 230
	catHeadX  = 130.0
	catHeadY  = 105.0
)

type CatPose struct {
	Blink  float64 // 0 open .. 1 closed
	GazeX  float64 // -1..1
	GazeY  float64
	Tail   float64 // phase in radians
	Breath float64 // 0..1
	Happy  bool
	Sleep  bool
	Wave   float64 // 0..1 paw wave
}

// PaintCat renders the cat into layer, centred on catHeadX/catHeadY.
func PaintCat(l *Canvas, p CatPose, th Theme) {
	ox, oy := catHeadX, catHeadY
	br := 1 + p.Breath*0.02

	// ---- tail
	wag := math.Sin(p.Tail)
	tail := Bezier(
		Vec{ox + 30, oy + 62},
		Vec{ox + 66, oy + 64 + 10*wag},
		Vec{ox + 78, oy + 30 + 22*wag},
		Vec{ox + 62, oy + 6 + 15*wag},
		34,
	)
	l.TaperedStroke(tail, 22, 12, th.Outline)
	l.TaperedStroke(tail, 15, 6, th.Body)
	l.FillEllipse(ox+62, oy+6+15*wag, 7, 7, th.Body)

	// ---- body
	l.FillEllipse(ox, oy+54, 45*br, 43*br, th.Outline)
	l.FillEllipse(ox, oy+54, 42*br, 40*br, th.Body)
	l.FillEllipse(ox, oy+62, 25*br, 26*br, th.Belly)

	// ---- back feet
	for _, fx := range []float64{-24, 22} {
		l.FillEllipse(ox+fx, oy+90, 17, 11, th.Outline)
		l.FillEllipse(ox+fx, oy+89, 14, 9, th.Body)
	}

	// ---- collar
	l.FillRoundRect(ox-30, oy+25, ox+30, oy+38, 6, th.Collar)
	// bell
	l.FillEllipse(ox, oy+43, 8.5, 8.5, th.Outline)
	l.FillEllipse(ox, oy+43, 7, 7, th.Bell)
	l.FillEllipse(ox-2.2, oy+40.5, 2.2, 2.2, rgb(255, 255, 255).alpha(0.85))
	l.StrokePath([]Vec{{ox - 2, oy + 42}, {ox, oy + 46}, {ox + 2, oy + 42}}, 1.2, th.Outline)

	// ---- ears
	earL := []Vec{{ox - 39, oy - 40}, {ox - 32, oy - 73}, {ox - 5, oy - 49}}
	earR := []Vec{{ox + 39, oy - 40}, {ox + 32, oy - 73}, {ox + 5, oy - 49}}
	l.FillPolygon(earL, th.Outline)
	l.FillPolygon(earR, th.Outline)
	innerL := []Vec{{ox - 32, oy - 43}, {ox - 28, oy - 63}, {ox - 13, oy - 49}}
	innerR := []Vec{{ox + 32, oy - 43}, {ox + 28, oy - 63}, {ox + 13, oy - 49}}
	l.FillPolygon(innerL, th.EarInner)
	l.FillPolygon(innerR, th.EarInner)

	// ---- head
	l.FillEllipse(ox, oy-14, 43, 43, th.Outline)
	l.FillEllipse(ox, oy-14, 40, 40, th.Body)
	l.FillEllipse(ox-14, oy-34, 17, 11, rgb(255, 255, 255).alpha(0.13))

	// ---- eyes
	eyeY := oy - 22.0
	for _, side := range []float64{-1, 1} {
		ex := ox + side*17 + p.GazeX*3
		ey := eyeY + p.GazeY*2.5
		if p.Sleep {
			arc := Arc(ex, ey+1, 9, 6, 0.15*math.Pi, 0.85*math.Pi, 12)
			l.StrokePath(arc, 2.4, th.Eye)
			continue
		}
		if p.Happy {
			arc := Arc(ex, ey+3, 9, 6, 1.15*math.Pi, 1.85*math.Pi, 12)
			l.StrokePath(arc, 2.6, th.Eye)
			continue
		}
		if p.Blink > 0.82 {
			l.StrokePath([]Vec{{ex - 8, ey}, {ex, ey + 2.5}, {ex + 8, ey}}, 2.4, th.Eye)
			continue
		}
		ry := 11 * (1 - p.Blink*0.92)
		l.FillEllipse(ex, ey, 9, ry, th.Eye)
		if ry > 4 {
			l.FillEllipse(ex-3.2+p.GazeX*1.5, ey-4.5+p.GazeY*1.2, 3.1, 3.1, rgb(255, 255, 255).alpha(0.95))
			l.FillEllipse(ex+3.2+p.GazeX*1.5, ey+3.6+p.GazeY*1.2, 1.5, 1.5, rgb(255, 255, 255).alpha(0.7))
		}
	}

	// ---- blush
	if !p.Sleep {
		l.FillEllipse(ox-28, oy+2, 9.5, 5.5, th.Blush.alpha(0.55))
		l.FillEllipse(ox+28, oy+2, 9.5, 5.5, th.Blush.alpha(0.55))
	}

	// ---- nose + mouth
	l.FillPolygon([]Vec{{ox - 4.5, oy - 6}, {ox + 4.5, oy - 6}, {ox, oy + 0.5}}, th.Nose)
	if p.Sleep {
		l.FillEllipse(ox, oy+8, 3.2, 3.2, th.Nose)
	} else if p.Happy {
		l.FillEllipse(ox, oy+9, 6.5, 5.5, th.Nose)
		l.FillEllipse(ox, oy+11.5, 4.2, 3, th.Nose.mix(rgb(255, 255, 255), 0.25))
	} else {
		l.StrokePath([]Vec{{ox - 7, oy + 6}, {ox - 3.5, oy + 10}, {ox, oy + 6.5}}, 2.0, th.Nose)
		l.StrokePath([]Vec{{ox, oy + 6.5}, {ox + 3.5, oy + 10}, {ox + 7, oy + 6}}, 2.0, th.Nose)
	}

	// ---- whiskers
	wc := th.Outline.alpha(0.55)
	for _, s := range []float64{-1, 1} {
		for i, dy := range []float64{-3, 2, 7} {
			x0 := ox + s*16
			y0 := oy + dy + 4
			x1 := ox + s*(42+float64(i)*2)
			y1 := oy + dy + 1 + (float64(i)-1)*3
			l.StrokePath([]Vec{{x0, y0}, {x1, y1}}, 1.4, wc)
		}
	}

	// ---- front paws
	pawY := oy + 78.0
	if p.Wave > 0.01 {
		// raise left paw
		px := ox - 20
		py := pawY - 22*p.Wave
		l.FillEllipse(px, py, 13, 11, th.Outline)
		l.FillEllipse(px, py, 11, 9, th.Body)
		l.StrokePath([]Vec{{px - 3, py + 3}, {px, py + 6}, {px + 3, py + 3}}, 1.3, th.Outline)
	} else {
		for _, fx := range []float64{-18, 18} {
			l.FillEllipse(ox+fx, pawY, 13, 10, th.Outline)
			l.FillEllipse(ox+fx, pawY, 11, 8, th.Body)
			l.StrokePath([]Vec{{ox + fx - 3, oy + 82}, {ox + fx, oy + 85}, {ox + fx + 3, oy + 82}}, 1.2, th.Outline.alpha(0.6))
		}
	}
}

func (c Col) mix(o Col, t float64) Col {
	return Col{
		uint8(float64(c.R)*(1-t) + float64(o.R)*t),
		uint8(float64(c.G)*(1-t) + float64(o.G)*t),
		uint8(float64(c.B)*(1-t) + float64(o.B)*t),
		c.A*(1-t) + o.A*t,
	}
}
