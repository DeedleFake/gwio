package ui

import (
	"fmt"
	"image/color"

	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
)

const (
	MinWidth  = 128
	MinHeight = 24

	WindowBorder = 5
)

var (
	ColorBackground          = color.NRGBA{0x77, 0x77, 0x77, 0xFF}
	ColorSelectionBox        = color.NRGBA{0xFF, 0x0, 0x0, 0xFF}
	ColorSelectionBackground = color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF / 100}
	ColorActiveBorder        = color.NRGBA{0x50, 0xA1, 0xAD, 0xFF}
	ColorInactiveBorder      = color.NRGBA{0x9C, 0xE9, 0xE9, 0xFF}
	ColorMenuSelected        = color.NRGBA{0x3D, 0x7D, 0x42, 0xFF}
	ColorMenuUnselected      = color.NRGBA{0xEB, 0xFF, 0xEC, 0xFF}
	ColorMenuBorder          = color.NRGBA{0x78, 0xAD, 0x84, 0xFF}
	ColorSurface             = color.NRGBA{0xEE, 0xEE, 0xEE, 0xFF}
)

var (
	fontOptions = opentype.FaceOptions{
		Size: 14,
		DPI:  72,
	}

	gomonoFont *sfnt.Font
	gomonoFace font.Face
)

func init() {
	var err error
	gomonoFont, err = opentype.Parse(gomono.TTF)
	if err != nil {
		panic(fmt.Errorf("parse font: %w", err))
	}

	gomonoFace, err = opentype.NewFace(gomonoFont, &fontOptions)
	if err != nil {
		panic(fmt.Errorf("create font face: %w", err))
	}
}