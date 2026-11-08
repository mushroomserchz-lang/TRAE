//go:build windows

package main

import (
	"encoding/binary"
	"math"
	"math/rand"
	"runtime"
	"syscall"
	"time"
	"unicode/utf8"
	"unsafe"
)

// ---------------------------------------------------------------- constants

const (
	WS_POPUP       = 0x80000000
	WS_EX_LAYERED  = 0x00080000
	WS_EX_TOPMOST  = 0x00000008
	WS_EX_TOOLWINDOW = 0x00000080
	WS_EX_NOACTIVATE = 0x08000000

	CS_DBLCLKS = 0x0008

	WM_DESTROY     = 0x0002
	WM_CLOSE       = 0x0010
	WM_PAINT       = 0x000F
	WM_ERASEBKGND  = 0x0014
	WM_NCHITTEST   = 0x0084
	WM_MOUSEACTIVATE = 0x0021
	WM_TIMER       = 0x0113
	WM_LBUTTONDOWN = 0x0201
	WM_LBUTTONUP   = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_MOUSEMOVE   = 0x0200
	WM_RBUTTONUP   = 0x0205

	HTCLIENT      = 1
	HTTRANSPARENT = ^uintptr(0) // -1

	MA_NOACTIVATE = 3

	SW_SHOWNOACTIVATE = 4

	SWP_NOSIZE     = 0x0001
	SWP_NOMOVE     = 0x0002
	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010

	SM_CXSCREEN = 0
	SM_CYSCREEN = 1

	SPI_GETWORKAREA = 0x0030



	SND_ASYNC     = 0x0001
	SND_NODEFAULT = 0x0002
	SND_MEMORY    = 0x0004

	DT_CENTER    = 0x0001
	DT_WORDBREAK = 0x0010
	DT_CALCRECT  = 0x0400
	DT_NOPREFIX  = 0x0800
)

var (
	hwndTopmost    = ^uintptr(0)
	hwndNotTopmost = ^uintptr(1)
)

// ---------------------------------------------------------------- win types

type POINT struct{ X, Y int32 }
type SIZE struct{ CX, CY int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }
type BLENDFUNCTION struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}
type MSG struct {
	Hwnd     syscall.Handle
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       POINT
	LPrivate uint32
}
type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     syscall.Handle
	HIcon         syscall.Handle
	HCursor       syscall.Handle
	HbrBackground syscall.Handle
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       syscall.Handle
}
type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}
type BITMAPINFO struct {
	BmiHeader BITMAPINFOHEADER
	BmiColors [1]uint32
}

// ---------------------------------------------------------------- dll procs

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	winmm    = syscall.NewLazyDLL("winmm.dll")

	pRegisterClassExW      = user32.NewProc("RegisterClassExW")
	pCreateWindowExW       = user32.NewProc("CreateWindowExW")
	pDefWindowProcW        = user32.NewProc("DefWindowProcW")
	pShowWindow            = user32.NewProc("ShowWindow")
	pGetMessageW           = user32.NewProc("GetMessageW")
	pTranslateMessage      = user32.NewProc("TranslateMessage")
	pDispatchMessageW      = user32.NewProc("DispatchMessageW")
	pSetTimer              = user32.NewProc("SetTimer")
	pKillTimer             = user32.NewProc("KillTimer")
	pGetCursorPos          = user32.NewProc("GetCursorPos")
	pSetWindowPos          = user32.NewProc("SetWindowPos")
	pSetCapture            = user32.NewProc("SetCapture")
	pReleaseCapture        = user32.NewProc("ReleaseCapture")
	pGetSystemMetrics      = user32.NewProc("GetSystemMetrics")
	pSystemParametersInfoW = user32.NewProc("SystemParametersInfoW")
	pDestroyWindow         = user32.NewProc("DestroyWindow")
	pPostQuitMessage       = user32.NewProc("PostQuitMessage")
	pValidateRect          = user32.NewProc("ValidateRect")
	pSetProcessDPIAware    = user32.NewProc("SetProcessDPIAware")
	pLoadCursorW           = user32.NewProc("LoadCursorW")
	pDrawTextW             = user32.NewProc("DrawTextW")
	pGetDC                 = user32.NewProc("GetDC")
	pReleaseDC             = user32.NewProc("ReleaseDC")
	pUpdateLayeredWindow   = user32.NewProc("UpdateLayeredWindow")

	pCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	pCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	pSelectObject       = gdi32.NewProc("SelectObject")
	pDeleteObject       = gdi32.NewProc("DeleteObject")
	pDeleteDC           = gdi32.NewProc("DeleteDC")
	pCreateFontW        = gdi32.NewProc("CreateFontW")
	pSetTextColor       = gdi32.NewProc("SetTextColor")
	pSetBkMode          = gdi32.NewProc("SetBkMode")

	pGetModuleHandleW = kernel32.NewProc("GetModuleHandleW")

	pPlaySoundW = winmm.NewProc("PlaySoundW")
)

// kept on the heap so the addresses stay stable across syscalls
var (
	gMsg       MSG
	gRect      RECT
	gPoint     POINT
	gSize      SIZE
	gSrc       POINT
	gDst       POINT
	gBlend     BLENDFUNCTION
	gBmi       BITMAPINFO
	gWndClass  WNDCLASSEXW
	gTextRect  RECT
	gTextBmi   BITMAPINFO
	gClassName = "TraeDesktopCatWnd"
)

// ---------------------------------------------------------------- layered surface

type Layered struct {
	dc   uintptr
	bmp  uintptr
	old  uintptr
	bits unsafe.Pointer
	w, h int
	n    int
}

func newLayered(w, h int) *Layered {
	screenDC, _, _ := pGetDC.Call(0)
	if screenDC == 0 {
		return nil
	}
	mem, _, _ := pCreateCompatibleDC.Call(screenDC)
	pReleaseDC.Call(0, screenDC)
	if mem == 0 {
		return nil
	}
	gBmi = BITMAPINFO{}
	gBmi.BmiHeader.BiSize = 40
	gBmi.BmiHeader.BiWidth = int32(w)
	gBmi.BmiHeader.BiHeight = int32(-h)
	gBmi.BmiHeader.BiPlanes = 1
	gBmi.BmiHeader.BiBitCount = 32
	gBmi.BmiHeader.BiCompression = 0
	var bits unsafe.Pointer
	bmp, _, _ := pCreateDIBSection.Call(mem, uintptr(unsafe.Pointer(&gBmi)), 0,
		uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 || bits == nil {
		pDeleteDC.Call(mem)
		return nil
	}
	old, _, _ := pSelectObject.Call(mem, bmp)
	return &Layered{dc: mem, bmp: bmp, old: old, bits: bits, w: w, h: h, n: w * h}
}

func (l *Layered) upload(c *Canvas) {
	dst := unsafe.Slice((*byte)(l.bits), l.n*4)
	src := c.Pix
	for i := 0; i < l.n; i++ {
		a := uint32(src[i*4+3])
		j := i * 4
		if a == 0 {
			dst[j], dst[j+1], dst[j+2], dst[j+3] = 0, 0, 0, 0
			continue
		}
		dst[j] = uint8(uint32(src[i*4+2]) * a / 255)
		dst[j+1] = uint8(uint32(src[i*4+1]) * a / 255)
		dst[j+2] = uint8(uint32(src[i*4+0]) * a / 255)
		dst[j+3] = uint8(a)
	}
}

func (l *Layered) present(hwnd uintptr, x, y int) {
	screenDC, _, _ := pGetDC.Call(0)
	gSize = SIZE{int32(l.w), int32(l.h)}
	gSrc = POINT{0, 0}
	gDst = POINT{int32(x), int32(y)}
	gBlend = BLENDFUNCTION{BlendOp: 0, BlendFlags: 0, SourceConstantAlpha: 255, AlphaFormat: 1}
	pUpdateLayeredWindow.Call(hwnd, screenDC, uintptr(unsafe.Pointer(&gDst)),
		uintptr(unsafe.Pointer(&gSize)), l.dc, uintptr(unsafe.Pointer(&gSrc)),
		0, uintptr(unsafe.Pointer(&gBlend)), 2)
	pReleaseDC.Call(0, screenDC)
}

// ---------------------------------------------------------------- app

type App struct {
	hwnd       uintptr
	wndProcRef uintptr
	layer      *Layered
	scene      *Scene

	start time.Time
	last  float64
	now   float64

	wx, wy float64
	vx, vy float64

	workL, workT, workR, workB int

	dragging  bool
	dragOffX  float64
	dragOffY  float64
	dragLastX float64
	dragLastY float64
	dragLastT float64
	dragMoved float64

	airborne bool
	airLift  float64

	follow     bool
	topmost    bool
	muted      bool
	sleepUntil float64
	nextIdle   float64
	walkTarget float64
	zoomUntil  float64

	hovering    bool
	hoverSince  float64
	petCooldown float64

	menuOpen  bool
	menuSince float64
	menuLeave float64
	menuHover int
	menuPress int
	menuImgs  []*TextImage
}

var app *App

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (a *App) tnow() float64 { return time.Since(a.start).Seconds() }

func (a *App) groundY() float64 { return float64(a.workB) - 290 }

func (a *App) cursorPos() (float64, float64) {
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&gPoint)))
	return float64(gPoint.X), float64(gPoint.Y)
}

func (a *App) readWorkArea() {
	gRect = RECT{}
	pSystemParametersInfoW.Call(SPI_GETWORKAREA, 0, uintptr(unsafe.Pointer(&gRect)), 0)
	if gRect.Right <= gRect.Left || gRect.Bottom <= gRect.Top {
		w, _, _ := pGetSystemMetrics.Call(SM_CXSCREEN)
		h, _, _ := pGetSystemMetrics.Call(SM_CYSCREEN)
		gRect = RECT{0, 0, int32(w), int32(h)}
	}
	a.workL = int(gRect.Left)
	a.workT = int(gRect.Top)
	a.workR = int(gRect.Right)
	a.workB = int(gRect.Bottom)
}

func newApp() *App {
	a := &App{start: time.Now(), topmost: true, walkTarget: math.NaN()}
	a.scene = NewScene(sceneW, sceneH)
	a.readWorkArea()
	a.wx = float64(a.workR - sceneW - 48)
	a.wy = a.groundY()
	if a.wx < float64(a.workL) {
		a.wx = float64(a.workL)
	}
	a.createWindow()
	a.layer = newLayered(sceneW, sceneH)

	a.scene.Say("喵~ 你好呀！右键可以打开菜单哦", 6.5, 0)
	a.nextIdle = 9
	a.render()
	pSetTimer.Call(a.hwnd, 1, 16, 0)
	return a
}

func (a *App) createWindow() {
	hInst, _, _ := pGetModuleHandleW.Call(0)
	className, _ := syscall.UTF16PtrFromString(gClassName)
	cursor, _, _ := pLoadCursorW.Call(0, 32512)
	a.wndProcRef = syscall.NewCallback(wndProc)
	gWndClass = WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:         CS_DBLCLKS,
		LpfnWndProc:   a.wndProcRef,
		HInstance:     syscall.Handle(hInst),
		HCursor:       syscall.Handle(cursor),
		LpszClassName: className,
	}
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&gWndClass)))

	title, _ := syscall.UTF16PtrFromString("桌面小猫")
	hwnd, _, _ := pCreateWindowExW.Call(
		uintptr(WS_EX_LAYERED|WS_EX_TOPMOST|WS_EX_TOOLWINDOW|WS_EX_NOACTIVATE),
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		uintptr(WS_POPUP),
		uintptr(int64(int32(a.wx))), uintptr(int64(int32(a.wy))),
		uintptr(sceneW), uintptr(sceneH),
		0, 0, hInst, 0,
	)
	a.hwnd = hwnd
	pShowWindow.Call(hwnd, SW_SHOWNOACTIVATE)
}

func (a *App) render() {
	a.scene.Update(0.016, a.now)
	c := a.scene.Render()
	if a.menuOpen {
		a.drawMenu(c)
	}
	a.layer.upload(c)
	a.layer.present(a.hwnd, int(math.Round(a.wx)), int(math.Round(a.wy)))
}

func (a *App) tick() {
	a.now = a.tnow()
	dt := a.now - a.last
	if dt < 0.001 {
		dt = 0.001
	}
	if dt > 0.06 {
		dt = 0.06
	}
	a.last = a.now
	a.update(a.now, dt)
	a.scene.Update(dt, a.now)
	c := a.scene.Render()
	if a.menuOpen {
		a.drawMenu(c)
	}
	a.layer.upload(c)
	a.layer.present(a.hwnd, int(math.Round(a.wx)), int(math.Round(a.wy)))
}

func (a *App) update(now, dt float64) {
	cx, cy := a.cursorPos()

	// gaze follows the cursor
	headX := a.wx + sceneW/2
	headY := a.wy + 190
	gx := clamp((cx-headX)/240, -1, 1)
	gy := clamp((cy-headY)/200, -1, 1)
	if a.scene.Sleep {
		gx, gy = 0, 0
	}
	a.scene.Look(gx, gy)

	ground := a.groundY()

	switch {
	case a.dragging:
		a.scene.Dragging = true
		a.scene.Airborne = false
		a.scene.Walking = false
		a.scene.AirLift = 0.7
	case a.airborne:
		a.scene.Dragging = false
		a.scene.Airborne = true
		a.scene.Walking = false
		a.vy += 2600 * dt
		a.wx += a.vx * dt
		a.wy += a.vy * dt
		a.airLift = clamp((ground-a.wy)/340, 0, 1)
		a.scene.AirLift = a.airLift

		left := float64(a.workL)
		right := float64(a.workR - sceneW)
		if a.wx < left {
			a.wx = left
			a.vx = -a.vx * 0.55
		}
		if a.wx > right {
			a.wx = right
			a.vx = -a.vx * 0.55
		}
		if a.wy < float64(a.workT-120) {
			a.wy = float64(a.workT - 120)
			if a.vy < 0 {
				a.vy = 0
			}
		}
		if a.wy >= ground {
			a.wy = ground
			if a.vy > 140 {
				a.vy = -a.vy * 0.40
				a.vx *= 0.82
				a.scene.Squash(1.28)
				if rand.Float64() < 0.45 {
					a.say(pick(bounceLines), now)
				}
			} else {
				a.vy, a.vx = 0, 0
				a.airborne = false
				a.airLift = 0
				a.scene.Airborne = false
				a.scene.AirLift = 0
				a.scene.Squash(1.30)
				if rand.Float64() < 0.5 {
					a.say(pick(landLines), now)
				}
			}
		}
	default:
		a.scene.Dragging = false
		a.scene.Airborne = false
		a.scene.AirLift = 0
		a.wy = ground

		if a.menuOpen {
			a.scene.Walking = false
			a.scene.leanT = 0
		} else if a.scene.Sleep {
			a.scene.Walking = false
			a.scene.leanT = 0
			if now >= a.sleepUntil {
				a.wake(now)
			}
		} else if a.follow {
			target := clamp(cx-sceneW/2, float64(a.workL), float64(a.workR-sceneW))
			a.walkToward(target, dt)
		} else if !math.IsNaN(a.walkTarget) {
			a.walkToward(a.walkTarget, dt)
		} else {
			a.scene.Walking = false
			a.scene.leanT = 0
			if now > a.nextIdle && !a.menuOpen {
				a.idleAction(now)
			}
		}
	}

	// hover / petting
	over := a.hitTest(cx, cy)
	if over && !a.dragging && !a.menuOpen {
		if !a.hovering {
			a.hovering = true
			a.hoverSince = now
		}
		if a.scene.Sleep && now-a.hoverSince > 0.9 {
			a.wake(now)
		}
		if !a.scene.Sleep && now-a.hoverSince > 0.45 && now > a.petCooldown {
			a.petCooldown = now + 1.5
			a.scene.Pet(now)
			if rand.Float64() < 0.5 {
				a.sound(sndChirp)
			}
			if rand.Float64() < 0.30 {
				a.say(pick(petLines), now)
			}
		}
	} else {
		a.hovering = false
	}

	if a.menuOpen {
		if a.pointInWindow(cx, cy) {
			a.menuLeave = 0
			mx, my := int(cx-a.wx), int(cy-a.wy)
			a.menuHover = a.menuItemAt(mx, my)
		} else if a.menuLeave == 0 {
			a.menuLeave = now
		} else if now-a.menuLeave > 0.35 {
			a.closeMenu()
		}
		if a.menuOpen && now-a.menuSince > 12 {
			a.closeMenu()
		}
	}
}

func (a *App) pointInWindow(sx, sy float64) bool {
	x := int(sx - a.wx)
	y := int(sy - a.wy)
	return x >= 0 && y >= 0 && x < sceneW && y < sceneH
}

func (a *App) walkToward(target float64, dt float64) {
	dx := target - a.wx
	if math.Abs(dx) <= 2.5 {
		a.wx = target
		a.scene.Walking = false
		a.scene.leanT = 0
		return
	}
	dir := 1.0
	if dx < 0 {
		dir = -1
	}
	speed := 82.0
	lean := 2.5
	if a.now < a.zoomUntil {
		speed = 320
		lean = 5
	}
	a.wx += dir * math.Min(math.Abs(dx), speed*dt)
	a.scene.Walking = true
	a.scene.leanT = dir * lean
}

func (a *App) idleAction(now float64) {
	a.nextIdle = now + 5.5 + rand.Float64()*7.5
	r := rand.Float64()
	switch {
	case r < 0.40:
		a.say(pick(idleLines), now)
	case r < 0.62:
		lo := float64(a.workL + 20)
		hi := float64(a.workR - sceneW - 20)
		if hi > lo {
			a.walkTarget = lo + rand.Float64()*(hi-lo)
		}
	case r < 0.74:
		a.sleepNow(now, 9+rand.Float64()*12)
	case r < 0.84:
		if rand.Float64() < 0.5 {
			a.scene.Spin()
			a.say("转圈圈~", now)
		} else {
			a.jump(420)
		}
	case r < 0.93:
		lo := float64(a.workL + 20)
		hi := float64(a.workR - sceneW - 20)
		if hi > lo && math.Abs(a.wx-(lo+hi)/2) > 80 {
			a.walkTarget = lo + rand.Float64()*(hi-lo)
			a.zoomUntil = now + 1.4 + rand.Float64()
			a.say("冲呀——！", now)
		}
	default:
		a.scene.Squash(1.18)
		a.scene.Pet(now)
	}
}

func (a *App) sleepNow(now, dur float64) {
	a.scene.Sleep = true
	a.sleepUntil = now + dur
	a.walkTarget = math.NaN()
	a.scene.Walking = false
	a.scene.Look(0, 0)
	a.say("打个盹…Zzz", now)
}

func (a *App) wake(now float64) {
	if !a.scene.Sleep {
		return
	}
	a.scene.Sleep = false
	a.sleepUntil = 0
	a.scene.Pet(now)
	a.say("唔…我醒啦", now)
}

func (a *App) say(text string, now float64) {
	dur := 1.9 + float64(utf8.RuneCountInString(text))*0.14
	if dur < 2.2 {
		dur = 2.2
	}
	a.scene.Say(text, dur, now)
}

func (a *App) jump(v float64) {
	if a.scene.Sleep {
		a.wake(a.tnow())
	}
	a.vy = -v
	a.airborne = true
	a.scene.Squash(0.84)
	a.wy -= 1
}

func (a *App) hitTest(sx, sy float64) bool {
	x := int(sx - a.wx)
	y := int(sy - a.wy)
	c := a.scene.canvas
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return false
	}
	return c.Pix[(y*c.W+x)*4+3] > 24
}

// ---------------------------------------------------------------- input

func (a *App) onLDown() {
	now := a.tnow()
	a.wake(now)
	a.dragging = true
	a.dragMoved = 0
	a.hovering = false
	a.walkTarget = math.NaN()
	a.scene.Walking = false
	a.scene.Squash(0.94)
	cx, cy := a.cursorPos()
	a.dragOffX = cx - a.wx
	a.dragOffY = cy - a.wy
	a.dragLastX, a.dragLastY, a.dragLastT = cx, cy, now
	a.vx, a.vy = 0, 0
	pSetCapture.Call(a.hwnd)
}

func (a *App) onMouseMove() {
	if !a.dragging {
		return
	}
	cx, cy := a.cursorPos()
	now := a.tnow()
	dt := now - a.dragLastT
	if dt > 0.001 {
		a.vx = (cx - a.dragLastX) / dt
		a.vy = (cy - a.dragLastY) / dt
	}
	a.dragMoved += math.Hypot(cx-a.dragLastX, cy-a.dragLastY)
	a.dragLastX, a.dragLastY, a.dragLastT = cx, cy, now
	a.wx = cx - a.dragOffX
	a.wy = cy - a.dragOffY
}

func (a *App) onLUp() {
	if !a.dragging {
		return
	}
	pReleaseCapture.Call()
	a.dragging = false
	a.scene.Dragging = false
	now := a.tnow()

	if a.dragMoved < 6 {
		a.jump(440)
		a.scene.Pet(now)
		a.sound(sndMeow)
		a.say(pick(clickLines), now)
		return
	}

	mag := math.Hypot(a.vx, a.vy)
	if mag > 2600 {
		k := 2600 / mag
		a.vx *= k
		a.vy *= k
	}
	a.airborne = true
	a.scene.Airborne = true
	if mag > 900 {
		a.say(pick(throwLines), now)
		a.sound(sndChirp)
	}
}

func (a *App) onDblClick() {
	now := a.tnow()
	a.wake(now)
	a.jump(700)
	a.scene.Spin()
	a.scene.Pet(now)
	a.sound(sndMeow)
	a.say(pick(dblLines), now)
}

const (
	menuX0      = 24.0
	menuY0      = 5.0
	menuW       = 252.0
	menuRowH    = 29.0
	menuPadTop  = 7.0
	menuCellPad = 6.0
	menuRows    = 4
	menuCols    = 2
)

var menuItems = []string{"摸摸头", "跟着鼠标", "去睡觉", "声音", "换毛色", "跳一下", "总在最前", "再见啦"}

func menuHeight() float64 { return menuPadTop*2 + menuRowH*menuRows }

func (a *App) menuChecked(i int) bool {
	switch i {
	case 1:
		return a.follow
	case 2:
		return a.scene.Sleep
	case 3:
		return !a.muted
	case 6:
		return a.topmost
	}
	return false
}

func (a *App) openMenu() {
	now := a.tnow()
	a.wake(now)
	a.menuOpen = true
	a.menuSince = now
	a.menuLeave = 0
	a.menuHover = -1
	a.menuPress = -1
	a.scene.clearSpeech()
	a.scene.Walking = false
	a.walkTarget = math.NaN()
	if a.menuImgs == nil {
		a.menuImgs = make([]*TextImage, len(menuItems))
		for i, label := range menuItems {
			if textRenderer != nil {
				a.menuImgs[i] = textRenderer(label, 150)
			}
		}
	}
}

func (a *App) closeMenu() {
	a.menuOpen = false
	a.menuHover = -1
	a.menuPress = -1
}

func (a *App) menuItemAt(mx, my int) int {
	cellW := (menuW - menuCellPad*2) / float64(menuCols)
	for i := range menuItems {
		r, col := i/menuCols, i%menuCols
		cx := menuX0 + menuCellPad + float64(col)*cellW
		cy := menuY0 + menuPadTop + float64(r)*menuRowH
		if float64(mx) >= cx && float64(mx) < cx+cellW &&
			float64(my) >= cy && float64(my) < cy+menuRowH {
			return i
		}
	}
	return -1
}

func (a *App) clientCursor() (int, int) {
	cx, cy := a.cursorPos()
	return int(cx - a.wx), int(cy - a.wy)
}

func (a *App) drawMenu(c *Canvas) {
	x0, y0 := menuX0, menuY0
	x1, y1 := x0+menuW, y0+menuHeight()

	c.FillRoundRect(x0+2, y0+4, x1+2, y1+4, 16, Col{25, 20, 30, 0.15})
	c.FillRoundRect(x0-1.6, y0-1.6, x1+1.6, y1+1.6, 17, Col{215, 186, 201, 1})
	c.FillRoundRect(x0, y0, x1, y1, 16, Col{255, 253, 252, 1})

	cellW := (menuW - menuCellPad*2) / float64(menuCols)
	for i := range menuItems {
		r, col := i/menuCols, i%menuCols
		cx := x0 + menuCellPad + float64(col)*cellW
		cy := y0 + menuPadTop + float64(r)*menuRowH
		if i == a.menuHover {
			c.FillRoundRect(cx+1, cy+1, cx+cellW-1, cy+menuRowH-1, 8, Col{255, 233, 242, 1})
		}
		if i < len(a.menuImgs) && a.menuImgs[i] != nil {
			img := a.menuImgs[i]
			tx := int(cx + (cellW-float64(img.W))/2)
			ty := int(cy + (menuRowH-float64(img.H))/2)
			tc := Col{88, 72, 82, 1}
			if i == a.menuHover {
				tc = Col{210, 82, 124, 1}
			}
			drawTextImage(c, img, tx, ty, tc)
		}
		if a.menuChecked(i) {
			c.FillEllipse(cx+cellW-11, cy+menuRowH/2, 3, 3, Col{234, 118, 158, 1})
		}
	}
}

func (a *App) runMenuAction(i int) {
	now := a.tnow()
	a.closeMenu()
	switch i {
	case 0:
		a.scene.Pet(now)
		a.sound(sndMeow)
		a.say(pick(petLines), now)
	case 1:
		a.follow = !a.follow
		if a.follow {
			a.scene.Sleep = false
			a.sleepUntil = 0
			a.walkTarget = math.NaN()
			a.say("马上跟上来！", now)
		} else {
			a.scene.Walking = false
			a.say("那我就在这儿待着~", now)
		}
	case 2:
		if a.scene.Sleep {
			a.wake(now)
		} else {
			a.sleepNow(now, 600)
		}
	case 3:
		a.muted = !a.muted
		if !a.muted {
			a.sound(sndChirp)
			a.say("喵~（开麦）", now)
		} else {
			a.say("好的，我安静一点", now)
		}
	case 4:
		a.scene.SetTheme(a.scene.theme + 1)
		a.scene.Pet(now)
		a.say(pick(colorLines), now)
	case 5:
		a.jump(620)
		a.sound(sndChirp)
	case 6:
		a.topmost = !a.topmost
		h := hwndNotTopmost
		if a.topmost {
			h = hwndTopmost
		}
		pSetWindowPos.Call(a.hwnd, h, 0, 0, 0, 0, SWP_NOMOVE|SWP_NOSIZE|SWP_NOACTIVATE)
		if a.topmost {
			a.say("我会一直在最上面哦", now)
		} else {
			a.say("好，我躲到后面去", now)
		}
	case 7:
		pDestroyWindow.Call(a.hwnd)
	}
}

// ---------------------------------------------------------------- sound

var (
	sndMeow  []byte
	sndChirp []byte
)

func wav(samples []int16, sr int) []byte {
	dataLen := len(samples) * 2
	buf := make([]byte, 44+dataLen)
	copy(buf[0:], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:], uint32(36+dataLen))
	copy(buf[8:], "WAVE")
	copy(buf[12:], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:], 16)
	binary.LittleEndian.PutUint16(buf[20:], 1)
	binary.LittleEndian.PutUint16(buf[22:], 1)
	binary.LittleEndian.PutUint32(buf[24:], uint32(sr))
	binary.LittleEndian.PutUint32(buf[28:], uint32(sr*2))
	binary.LittleEndian.PutUint16(buf[32:], 2)
	binary.LittleEndian.PutUint16(buf[34:], 16)
	copy(buf[36:], "data")
	binary.LittleEndian.PutUint32(buf[40:], uint32(dataLen))
	for i, s := range samples {
		binary.LittleEndian.PutUint16(buf[44+i*2:], uint16(s))
	}
	return buf
}

func synthMeow() []byte {
	const sr = 22050
	n := int(sr * 0.26)
	out := make([]int16, n)
	var phase float64
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n)
		f := 540 + 280*math.Sin(math.Pi*math.Min(1, t*1.2)) - 130*t
		phase += 2 * math.Pi * f / sr
		v := math.Sin(phase)*0.6 + math.Sin(phase*2)*0.18 + math.Sin(phase*3)*0.06
		env := math.Min(1, t*22) * math.Pow(1-t, 1.5)
		out[i] = int16(v * env * 8200)
	}
	return wav(out, sr)
}

func synthChirp() []byte {
	const sr = 22050
	n := int(sr * 0.18)
	out := make([]int16, n)
	var phase float64
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n)
		note := 980.0
		if t > 0.45 {
			note = 1320.0
		}
		phase += 2 * math.Pi * note / sr
		v := math.Sin(phase)*0.55 + math.Sin(phase*2)*0.12
		env := math.Min(1, t*30) * math.Pow(1-t, 2.0)
		out[i] = int16(v * env * 7000)
	}
	return wav(out, sr)
}

func (a *App) sound(b []byte) {
	if a.muted || len(b) == 0 {
		return
	}
	pPlaySoundW.Call(uintptr(unsafe.Pointer(&b[0])), 0, SND_MEMORY|SND_ASYNC|SND_NODEFAULT)
}

// ---------------------------------------------------------------- lines

var idleLines = []string{
	"喵~",
	"今天也要开心呀",
	"摸鱼中，请勿打扰",
	"你工作好认真哦",
	"要不要休息一下？喝口水吧",
	"我在这儿陪你",
	"喵喵喵？",
	"偷偷告诉你：你已经很棒了",
	"记得眨眼睛哦",
	"困了就去睡吧",
	"不要久坐呀，起来动一动",
	"喵呜~（打了个滚）",
	"我在看着你哦",
	"这条鱼是我的（护食）",
	"摸摸我嘛",
	"我才不是在偷懒",
	"嘿嘿",
	"喵星信号接收中…",
	"累了就靠一会儿吧",
	"窗外的风好像很舒服",
	"要不要听我讲个冷笑话？",
	"今天也辛苦啦",
}

var petLines = []string{
	"呼噜呼噜…",
	"好舒服呀~",
	"再摸一下嘛",
	"嘿嘿，好痒",
	"最喜欢你了",
	"喵~（眯眼睛）",
}

var clickLines = []string{
	"喵？",
	"哇！",
	"嘿嘿，好痒",
	"再摸一下嘛",
	"喵呜~",
	"吓我一跳！",
}

var dblLines = []string{
	"喵呜——！",
	"转圈圈~",
	"我飞起来啦！",
	"看我的绝招！",
}

var dragLines = []string{
	"放我下来！",
	"喵——！",
	"要飞起来啦",
}

var throwLines = []string{
	"我飞了！",
	"啊啊啊啊——",
	"抓稳我！",
}

var bounceLines = []string{
	"哎哟！",
	"弹起来了！",
	"喵！",
}

var landLines = []string{
	"安全着陆~",
	"喵…",
	"屁股有点疼",
}

var colorLines = []string{
	"换新衣服啦~",
	"这个颜色适合我吗？",
	"喵~好看吗",
}

func pick(list []string) string {
	if len(list) == 0 {
		return ""
	}
	return list[rand.Intn(len(list))]
}

// ---------------------------------------------------------------- window proc

func wndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	if app == nil {
		r, _, _ := pDefWindowProcW.Call(hwnd, msg, wparam, lparam)
		return r
	}
	switch msg {
	case WM_TIMER:
		app.tick()
		return 0
	case WM_NCHITTEST:
		if app.dragging {
			return HTCLIENT
		}
		x := float64(int(int16(lparam & 0xffff)))
		y := float64(int(int16((lparam >> 16) & 0xffff)))
		if app.hitTest(x, y) {
			return HTCLIENT
		}
		return HTTRANSPARENT
	case WM_MOUSEACTIVATE:
		return MA_NOACTIVATE
	case WM_LBUTTONDOWN:
		if app.menuOpen {
			mx, my := app.clientCursor()
			app.menuPress = app.menuItemAt(mx, my)
			return 0
		}
		app.onLDown()
		return 0
	case WM_LBUTTONUP:
		if app.menuOpen {
			mx, my := app.clientCursor()
			i := app.menuItemAt(mx, my)
			if i >= 0 && i == app.menuPress {
				app.runMenuAction(i)
			} else {
				app.closeMenu()
			}
			app.menuPress = -1
			return 0
		}
		app.onLUp()
		return 0
	case WM_LBUTTONDBLCLK:
		if !app.menuOpen {
			app.onDblClick()
		}
		return 0
	case WM_MOUSEMOVE:
		if app.menuOpen {
			mx, my := app.clientCursor()
			app.menuHover = app.menuItemAt(mx, my)
			return 0
		}
		app.onMouseMove()
		return 0
	case WM_RBUTTONUP:
		if app.menuOpen {
			app.closeMenu()
		} else {
			app.openMenu()
		}
		return 0
	case WM_ERASEBKGND:
		return 1
	case WM_PAINT:
		pValidateRect.Call(hwnd, 0)
		return 0
	case WM_DESTROY:
		pKillTimer.Call(hwnd, 1)
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return r
}

// ---------------------------------------------------------------- main

func main() {
	runtime.LockOSThread()
	pSetProcessDPIAware.Call()
	sndMeow = synthMeow()
	sndChirp = synthChirp()
	textRenderer = renderTextWin

	app = newApp()
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&gMsg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&gMsg)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&gMsg)))
	}
}
