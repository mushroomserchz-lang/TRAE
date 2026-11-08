package main

import (
	"math"
	"math/rand"
)

// TextImage is a grayscale coverage map for a rendered string.
type TextImage struct {
	W, H int
	A    []uint8
}

// textRenderer is installed by the platform layer (GDI on Windows).
var textRenderer func(s string, maxW int) *TextImage

const (
	sceneW = 300
	sceneH = 290
)

type particle struct {
	X, Y   float64
	VX, VY float64
	Life   float64
	Max    float64
	Size   float64
	Kind   int
}

type Scene struct {
	W, H   int
	canvas *Canvas
	layer  *Canvas
	t      float64
	theme  int

	blinkNext float64
	blinking  bool
	blinkProg float64
	blink     float64

	gazeX, gazeY   float64
	gazeTX, gazeTY float64

	happyUntil float64
	Sleep      bool
	Walking    bool
	Dragging   bool
	Airborne   bool
	AirLift    float64
	lean       float64
	leanT      float64
	bob        float64

	bubble      string
	bubbleStart float64
	bubbleUntil float64
	textImg     *TextImage
	textFor     string

	hearts    []particle
	zs        []particle
	nextHeart float64
	nextZ     float64

	spinT   float64
	spinAng float64
	squash  float64
	squashV float64
}

func NewScene(w, h int) *Scene {
	return &Scene{
		W: w, H: h,
		canvas:    NewCanvas(w, h),
		layer:     NewCanvas(catLayerW, catLayerH),
		squash:    1,
		blinkNext: 2.2 + rand.Float64()*2.5,
	}
}

func (s *Scene) SetTheme(i int) { s.theme = i % len(themes) }

func (s *Scene) Look(x, y float64) { s.gazeTX, s.gazeTY = x, y }

func (s *Scene) Say(text string, dur, now float64) {
	s.bubble = text
	s.bubbleStart = now
	s.bubbleUntil = now + dur
	s.textFor = ""
}

func (s *Scene) clearSpeech() {
	s.bubble = ""
	s.textFor = ""
}

func (s *Scene) Pet(now float64) {
	s.happyUntil = now + 2.4
	if len(s.hearts) < 60 {
		for i := 0; i < 6; i++ {
			s.hearts = append(s.hearts, particle{
				X:    float64(s.W)/2 + (rand.Float64()-0.5)*70,
				Y:    float64(s.H) - 166 + (rand.Float64()-0.5)*26,
				VX:   (rand.Float64() - 0.5) * 26,
				VY:   -18 - rand.Float64()*22,
				Life: 0.9 + rand.Float64()*0.6,
				Max:  1.5,
				Size: 7 + rand.Float64()*6,
			})
		}
	}
}

func (s *Scene) Squash(v float64) { s.squash = v }

func (s *Scene) Spin() {
	s.spinT = 0.62
	s.spinAng = 0
}

func (s *Scene) Update(dt, now float64) {
	s.t = now

	// blink
	if s.Sleep || s.Dragging {
		s.blinking = false
		s.blink = 0
	} else if s.blinking {
		s.blinkProg += dt / 0.15
		if s.blinkProg >= 1 {
			s.blinking = false
			s.blink = 0
			s.blinkNext = now + 2.0 + rand.Float64()*4.0
		} else {
			s.blink = math.Sin(math.Pi * s.blinkProg)
		}
	} else if now >= s.blinkNext {
		s.blinking = true
		s.blinkProg = 0
	}

	// gaze smoothing
	gx, gy := s.gazeTX, s.gazeTY
	if s.Sleep {
		gx, gy = 0, 0
	}
	k := math.Min(1, dt*10)
	s.gazeX += (gx - s.gazeX) * k
	s.gazeY += (gy - s.gazeY) * k

	// lean smoothing
	s.lean += (s.leanT - s.lean) * math.Min(1, dt*6)

	// squash spring
	kk, dd := 240.0, 15.0
	s.squashV += (-kk*(s.squash-1) - dd*s.squashV) * dt
	s.squash += s.squashV * dt
	if s.squash < 0.6 {
		s.squash = 0.6
		s.squashV = 0
	}
	if s.squash > 1.45 {
		s.squash = 1.45
		s.squashV = 0
	}

	// spin
	if s.spinT > 0 {
		s.spinT -= dt
		s.spinAng += dt * 2 * math.Pi / 0.62
		if s.spinT <= 0 {
			s.spinT = 0
			s.spinAng = 0
		}
	}

	// particles
	for i := len(s.hearts) - 1; i >= 0; i-- {
		p := &s.hearts[i]
		p.Life -= dt
		p.X += (p.VX + math.Sin(p.Life*7)*10) * dt
		p.Y += p.VY * dt
		p.VY *= 0.995
		if p.Life <= 0 {
			s.hearts = append(s.hearts[:i], s.hearts[i+1:]...)
		}
	}
	for i := len(s.zs) - 1; i >= 0; i-- {
		p := &s.zs[i]
		p.Life -= dt
		p.X += p.VX * dt
		p.Y += p.VY * dt
		if p.Life <= 0 {
			s.zs = append(s.zs[:i], s.zs[i+1:]...)
		}
	}

	// ambient hearts while being petted
	if now < s.happyUntil && now > s.nextHeart {
		s.nextHeart = now + 0.28
		if len(s.hearts) < 50 {
			s.hearts = append(s.hearts, particle{
				X:    float64(s.W)/2 + (rand.Float64()-0.5)*64,
				Y:    float64(s.H) - 170 + (rand.Float64()-0.5)*24,
				VX:   (rand.Float64() - 0.5) * 18,
				VY:   -22 - rand.Float64()*16,
				Life: 0.8 + rand.Float64()*0.5,
				Max:  1.3,
				Size: 6 + rand.Float64()*5,
			})
		}
	}

	// zzz while asleep
	if s.Sleep && now > s.nextZ {
		s.nextZ = now + 1.1
		s.zs = append(s.zs, particle{
			X:    float64(s.W)/2 + 34,
			Y:    float64(s.H) - 168,
			VX:   10,
			VY:   -15,
			Life: 2.2,
			Max:  2.2,
			Size: 7,
		})
	}
}

func (s *Scene) walkingBob() float64 {
	if s.Walking && !s.Airborne && !s.Dragging {
		return -math.Abs(math.Sin(s.t*11)) * 3
	}
	return 0
}

func (s *Scene) pose(now float64) CatPose {
	happy := now < s.happyUntil && !s.Sleep
	breath := 0.5 + 0.5*math.Sin(s.t*1.7)
	if s.Sleep {
		breath = 0.5 + 0.5*math.Sin(s.t*0.8)
	}
	tailSpeed := 2.3
	if happy {
		tailSpeed = 7.5
	}
	if s.Sleep {
		tailSpeed = 0.7
	}
	blink := s.blink
	if s.Sleep {
		blink = 1
	}
	wave := 0.0
	if happy {
		wave = 0.55 + 0.45*math.Sin(s.t*9)
	}
	return CatPose{
		Blink:  blink,
		GazeX:  s.gazeX,
		GazeY:  s.gazeY,
		Tail:   s.t * tailSpeed,
		Breath: breath,
		Happy:  happy,
		Sleep:  s.Sleep,
		Wave:   wave,
	}
}

func (s *Scene) Render() *Canvas {
	c := s.canvas
	c.Clear()
	now := s.t

	headCX := float64(s.W)/2 + s.lean
	headCY := float64(s.H) - 100 + s.bob + s.walkingBob()

	// shadow
	shA := 0.20 * (1 - math.Min(1, s.AirLift)*0.75)
	if shA > 0.02 {
		c.FillEllipse(float64(s.W)/2, float64(s.H)-7, 44, 7, Col{25, 20, 30, shA})
	}

	// cat
	s.layer.Clear()
	PaintCat(s.layer, s.pose(now), themes[s.theme])
	sq := math.Sqrt(s.squash)
	sx, sy := sq, 1/sq
	c.BlitTransform(s.layer, headCX, headCY+40, catHeadX, catHeadY+40, s.spinAng, sx, sy)

	// zzz
	for _, p := range s.zs {
		a := p.Life / p.Max
		if a > 1 {
			a = 1
		}
		sz := p.Size * (1 + (1-a)*0.8)
		col := Col{120, 110, 160, a * 0.85}
		c.StrokePath([]Vec{{p.X - sz, p.Y - sz}, {p.X + sz, p.Y - sz}, {p.X - sz, p.Y + sz}, {p.X + sz, p.Y + sz}}, 2.2, col)
	}

	// hearts
	for _, p := range s.hearts {
		a := p.Life / p.Max
		if a > 1 {
			a = 1
		}
		c.Heart(p.X, p.Y, p.Size, Col{255, 105, 140, a * 0.9})
	}

	// speech bubble
	s.drawBubble(c, now, headCX, headCY)

	return c
}

func (s *Scene) drawBubble(c *Canvas, now float64, headCX, headCY float64) {
	if s.bubble == "" || now > s.bubbleUntil {
		return
	}
	a := 1.0
	if now > s.bubbleUntil-0.35 {
		a = (s.bubbleUntil - now) / 0.35
	}
	if now-s.bubbleStart < 0.12 {
		a = math.Min(a, (now-s.bubbleStart)/0.12)
	}
	if a <= 0 {
		return
	}
	if s.textFor != s.bubble {
		s.textFor = s.bubble
		if textRenderer != nil {
			s.textImg = textRenderer(s.bubble, 210)
		} else {
			s.textImg = nil
		}
	}
	tw, th := 0, 0
	if s.textImg != nil {
		tw, th = s.textImg.W, s.textImg.H
	} else {
		// preview fallback: rough estimate so the layout looks right
		n := 0
		for range s.bubble {
			n++
		}
		if n > 14 {
			n = 14
		}
		tw = n*15 + 6
		th = 20
	}
	bw := float64(tw) + 30
	bh := float64(th) + 16
	if bw > 262 {
		bw = 262
	}
	cx := float64(s.W) / 2
	by := headCY - 78 - bh
	if by < 5 {
		by = 5
	}
	bx0 := cx - bw/2
	bx1 := cx + bw/2
	by1 := by + bh

	// tail
	tailOuter := []Vec{{cx - 10, by1 - 1}, {cx + 10, by1 - 1}, {cx + 1, by1 + 14}}
	tailInner := []Vec{{cx - 6.5, by1 - 1}, {cx + 6.5, by1 - 1}, {cx + 1, by1 + 9}}
	c.FillPolygon(tailOuter, Col{214, 185, 200, a})
	c.FillPolygon(tailInner, Col{255, 253, 252, a})

	// bubble body with border
	c.FillRoundRect(bx0-1.6, by-1.6, bx1+1.6, by1+1.6, 15, Col{214, 185, 200, a})
	c.FillRoundRect(bx0, by, bx1, by1, 14, Col{255, 253, 252, a * 0.98})

	if s.textImg != nil {
		tx := int(cx - float64(tw)/2)
		ty := int(by + 8)
		drawTextImage(c, s.textImg, tx, ty, Col{76, 60, 70, a})
	}
}

func drawTextImage(c *Canvas, img *TextImage, x, y int, col Col) {
	for j := 0; j < img.H; j++ {
		for i := 0; i < img.W; i++ {
			a := float64(img.A[j*img.W+i]) / 255
			if a <= 0 {
				continue
			}
			c.blendPixel(x+i, y+j, col, a)
		}
	}
}
