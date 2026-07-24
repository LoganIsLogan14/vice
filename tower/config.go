// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package tower

import "github.com/mmp/vice/renderer"

type CallsignDisplayMode int

const (
	CallsignDisplayFull CallsignDisplayMode = iota
	CallsignDisplayAirline
	CallsignDisplayNone
)

type StatusMETARMode int

const (
	StatusMETARNone StatusMETARMode = iota
	StatusMETARSimplified
	StatusMETARFull
)

// TowerCabConfig contains controller-facing display preferences.
type TowerCabConfig struct {
	Initialized bool

	ShowAerial     bool
	ShowCRCMap     bool
	ShowVisibility bool
	UseHighRes     bool

	AerialBrightness float32

	HighResStartNM float32
	HighResFullNM  float32

	VisibilityFadeFraction  float32
	VisibilityAnimationRate float32

	ShowDataBlocks           bool
	DataBlockFontSize        int
	CallsignMode             CallsignDisplayMode
	ShowAircraftType         bool
	ShowAltitude             bool
	ShowDataBlockBackground  bool
	DataBlockBackgroundAlpha float32
	ShowDataBlockLeaderLines bool

	AircraftScale         float32
	AircraftShadowOffsetX float32
	AircraftShadowOffsetY float32
	AircraftOutlineScale  float32

	ShowStatusBar  bool
	StatusFontSize int
	StatusMETAR    StatusMETARMode

	MousePanZoomEnabled bool
	ZoomToCursor        bool

	BackgroundColor renderer.RGB
	DataBlockColor  renderer.RGB
	AircraftColor   renderer.RGB
	StatusTextColor renderer.RGB
}

func DefaultTowerCabConfig() TowerCabConfig {
	return TowerCabConfig{
		Initialized:    true,
		ShowAerial:     true,
		ShowCRCMap:     true,
		ShowVisibility: true,
		UseHighRes:     true,

		AerialBrightness: 1.00,

		HighResStartNM: 9,
		HighResFullNM:  6,

		VisibilityFadeFraction:  0.28,
		VisibilityAnimationRate: 0.08,

		ShowDataBlocks:           true,
		DataBlockFontSize:        20,
		CallsignMode:             CallsignDisplayFull,
		ShowAircraftType:         true,
		ShowAltitude:             true,
		ShowDataBlockBackground:  true,
		DataBlockBackgroundAlpha: 0.50,
		ShowDataBlockLeaderLines: true,

		AircraftScale:         0.18,
		AircraftShadowOffsetX: 2.0,
		AircraftShadowOffsetY: -2.0,
		AircraftOutlineScale:  1.10,

		ShowStatusBar:  true,
		StatusFontSize: 20,
		StatusMETAR:    StatusMETARFull,

		MousePanZoomEnabled: true,
		ZoomToCursor:        true,

		BackgroundColor: renderer.RGB{R: 0, G: 0, B: 0},
		DataBlockColor:  renderer.RGB{R: 0, G: 1, B: 0},
		AircraftColor:   renderer.RGB{R: 0.92, G: 0.94, B: 0.96},
		StatusTextColor: renderer.RGB{R: 0, G: 1, B: 0},
	}
}

func (c *TowerCabConfig) EnsureDefaults() {
	if c.Initialized {
		return
	}
	*c = DefaultTowerCabConfig()
}

func (c *TowerCabConfig) Normalize() {
	c.AerialBrightness = min(max(c.AerialBrightness, float32(0)), float32(1))

	c.HighResStartNM = max(c.HighResStartNM, float32(0.1))
	c.HighResFullNM = max(c.HighResFullNM, float32(0))
	if c.HighResFullNM >= c.HighResStartNM {
		c.HighResFullNM = max(c.HighResStartNM-0.1, float32(0))
	}

	c.VisibilityFadeFraction = min(max(c.VisibilityFadeFraction, float32(0.01)), float32(0.95))
	c.VisibilityAnimationRate = min(max(c.VisibilityAnimationRate, float32(0.001)), float32(1))

	c.DataBlockFontSize = min(max(c.DataBlockFontSize, 8), 32)
	c.DataBlockBackgroundAlpha = min(max(c.DataBlockBackgroundAlpha, float32(0)), float32(1))
	c.StatusFontSize = min(max(c.StatusFontSize, 10), 48)
	c.AircraftScale = min(max(c.AircraftScale, float32(0.01)), float32(2.0))
	c.AircraftShadowOffsetX = min(max(c.AircraftShadowOffsetX, float32(-20)), float32(20))
	c.AircraftShadowOffsetY = min(max(c.AircraftShadowOffsetY, float32(-20)), float32(20))
	c.AircraftOutlineScale = min(max(c.AircraftOutlineScale, float32(1.0)), float32(1.5))
	c.CallsignMode = CallsignDisplayMode(min(max(int(c.CallsignMode), 0), 2))
	c.StatusMETAR = StatusMETARMode(min(max(int(c.StatusMETAR), 0), 2))

	clampColor := func(rgb *renderer.RGB) {
		rgb.R = min(max(rgb.R, float32(0)), float32(1))
		rgb.G = min(max(rgb.G, float32(0)), float32(1))
		rgb.B = min(max(rgb.B, float32(0)), float32(1))
	}
	clampColor(&c.BackgroundColor)
	clampColor(&c.DataBlockColor)
	clampColor(&c.AircraftColor)
	clampColor(&c.StatusTextColor)
}
