// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

// Package tower implements top-down tower-controller displays.
package tower

import (
	"fmt"

	av "github.com/mmp/vice/aviation"
	"github.com/mmp/vice/client"
	"github.com/mmp/vice/log"
	"github.com/mmp/vice/panes"
	"github.com/mmp/vice/platform"
	"github.com/mmp/vice/renderer"
)

// TowerCabPane is VICE's top-down tower-cab display.
type TowerCabPane struct {
	camera Camera
	font   *renderer.Font
}

func NewTowerCabPane() *TowerCabPane { return &TowerCabPane{camera: NewCamera()} }

func (tp *TowerCabPane) Activate(r renderer.Renderer, p platform.Platform, lg *log.Logger) {
	tp.font = renderer.GetDefaultFont()
}

func (tp *TowerCabPane) LoadedSim(c *client.ControlClient, p platform.Platform, lg *log.Logger) {
	tp.resetView(c)
}

func (tp *TowerCabPane) ResetSim(c *client.ControlClient, p platform.Platform, lg *log.Logger) {
	tp.resetView(c)
}

func (tp *TowerCabPane) resetView(c *client.ControlClient) {
	tp.camera.Reset(c.State.Center)
	if ap, ok := av.DB.LookupAirport(c.State.PrimaryAirport); ok {
		tp.camera.Reset(ap.Location)
	}
}

func (tp *TowerCabPane) CanTakeKeyboardFocus() bool { return false }

func (tp *TowerCabPane) Draw(ctx *panes.Context, cb *renderer.CommandBuffer) {
	ctx.SetWindowCoordinateMatrices(cb)

	transforms := tp.camera.Transforms(ctx)
	if tp.camera.HandleMouse(ctx, transforms) {
		transforms = tp.camera.Transforms(ctx)
	}

	fills := renderer.GetColoredTrianglesDrawBuilder()
	defer renderer.ReturnColoredTrianglesDrawBuilder(fills)
	lines := renderer.GetLinesDrawBuilder()
	defer renderer.ReturnLinesDrawBuilder(lines)
	text := renderer.GetTextDrawBuilder()
	defer renderer.ReturnTextDrawBuilder(text)

	airportID := ctx.Client.State.PrimaryAirport
	airport, airportOK := av.DB.LookupAirport(airportID)
	if airportOK {
		drawAirport(airport, transforms, fills, lines, text, tp.font)
	} else {
		drawReferencePoint(transforms.WindowFromLatLongP(tp.camera.Center), lines)
	}

	// Pavement first, then markings and labels.
	fills.GenerateCommands(cb)
	cb.SetRGB(renderer.RGB{R: 0.66, G: 0.68, B: 0.70})
	cb.LineWidth(1.5, ctx.DPIScale)
	lines.GenerateCommands(cb)

	style := renderer.TextStyle{Font: tp.font, Color: renderer.RGB{R: 0.88, G: 0.90, B: 0.92}}
	text.AddText(fmt.Sprintf("TOWER CAB  %s  %.1f NM", airportID, tp.camera.RangeNM),
		[2]float32{12, ctx.PaneExtent.Height() - 24}, style)
	text.AddText("Mouse wheel: zoom   Right-drag: pan   Double-right-click: reset",
		[2]float32{12, 16}, renderer.TextStyle{Font: tp.font, Color: renderer.RGB{R: 0.55, G: 0.58, B: 0.60}})
	text.GenerateCommands(cb)
}
