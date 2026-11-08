//go:build windows

package main

import (
	"syscall"
	"unsafe"
)

var gFont uintptr

func getFont() uintptr {
	if gFont != 0 {
		return gFont
	}
	face, _ := syscall.UTF16PtrFromString("Microsoft YaHei UI")
	fontHeight := int32(-17)
	f, _, _ := pCreateFontW.Call(
		uintptr(fontHeight), // height
		0,                   // width
		0,                   // escapement
		0,                   // orientation
		600,                 // weight
		0,                   // italic
		0,                   // underline
		0,                   // strikeout
		1,                   // DEFAULT_CHARSET
		5,                   // OUT_TT_PRECIS
		0,                   // CLIP_DEFAULT_PRECIS
		4,                   // ANTIALIASED_QUALITY
		0,                   // pitch and family
		uintptr(unsafe.Pointer(face)),
	)
	gFont = f
	return f
}

// renderTextWin draws white text into a 32bpp DIB and converts the
// result to a coverage map. Runs on the UI thread only.
func renderTextWin(s string, maxW int) *TextImage {
	if s == "" || maxW < 24 {
		return nil
	}
	face := getFont()
	if face == 0 {
		return nil
	}
	txt, err := syscall.UTF16PtrFromString(s)
	if err != nil {
		return nil
	}

	screenDC, _, _ := pGetDC.Call(0)
	if screenDC == 0 {
		return nil
	}
	mem, _, _ := pCreateCompatibleDC.Call(screenDC)
	pReleaseDC.Call(0, screenDC)
	if mem == 0 {
		return nil
	}
	defer pDeleteDC.Call(mem)

	oldFont, _, _ := pSelectObject.Call(mem, face)

	// measure
	gTextRect = RECT{0, 0, int32(maxW), 0}
	pDrawTextW.Call(mem, uintptr(unsafe.Pointer(txt)), ^uintptr(0),
		uintptr(unsafe.Pointer(&gTextRect)), uintptr(DT_CALCRECT|DT_WORDBREAK|DT_NOPREFIX))

	w := int(gTextRect.Right-gTextRect.Left) + 4
	h := int(gTextRect.Bottom-gTextRect.Top) + 4
	if w < 8 {
		w = 8
	}
	if h < 8 {
		h = 8
	}
	if w > maxW+4 {
		w = maxW + 4
	}

	gTextBmi = BITMAPINFO{}
	gTextBmi.BmiHeader.BiSize = 40
	gTextBmi.BmiHeader.BiWidth = int32(w)
	gTextBmi.BmiHeader.BiHeight = int32(-h)
	gTextBmi.BmiHeader.BiPlanes = 1
	gTextBmi.BmiHeader.BiBitCount = 32
	gTextBmi.BmiHeader.BiCompression = 0

	var bits unsafe.Pointer
	bmp, _, _ := pCreateDIBSection.Call(mem, uintptr(unsafe.Pointer(&gTextBmi)), 0,
		uintptr(unsafe.Pointer(&bits)), 0, 0)
	if bmp == 0 || bits == nil {
		pSelectObject.Call(mem, oldFont)
		return nil
	}
	oldBmp, _, _ := pSelectObject.Call(mem, bmp)

	buf := unsafe.Slice((*byte)(bits), w*h*4)
	for i := range buf {
		buf[i] = 0
	}

	pSetBkMode.Call(mem, 1) // TRANSPARENT
	pSetTextColor.Call(mem, 0x00FFFFFF)
	gTextRect = RECT{0, 0, int32(w), int32(h)}
	pDrawTextW.Call(mem, uintptr(unsafe.Pointer(txt)), ^uintptr(0),
		uintptr(unsafe.Pointer(&gTextRect)),
		uintptr(DT_CENTER|DT_WORDBREAK|DT_NOPREFIX))

	out := &TextImage{W: w, H: h, A: make([]uint8, w*h)}
	for i := 0; i < w*h; i++ {
		b := buf[i*4]
		g := buf[i*4+1]
		r := buf[i*4+2]
		m := r
		if g > m {
			m = g
		}
		if b > m {
			m = b
		}
		out.A[i] = m
	}

	pSelectObject.Call(mem, oldBmp)
	pSelectObject.Call(mem, oldFont)
	pDeleteObject.Call(bmp)
	return out
}
