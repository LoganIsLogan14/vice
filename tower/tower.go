// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

// Package tower implements top-down tower-controller displays.
package tower

import (
	"fmt"

	av "github.com/mmp/vice/aviation"
	"github.com/mmp/vice/client"
	"github.com/mmp/vice/log"
	"github.com/mmp/vice/math"
	"github.com/mmp/vice/panes"
	"github.com/mmp/vice/platform"
	"github.com/mmp/vice/radar"
	"github.com/mmp/vice/renderer"
)

// TowerCabPane is an early top-down tower-cab prototype. It intentionally
// renders only aircraft already present in VICE; ground movement is not yet
// simulated.
type TowerCabPane struct {
	Center  math.Point2LL
	RangeNM float32

	font *renderer.Font
}

func NewTowerCabPane() *TowerCabPane {
	return &TowerCabPane{RangeNM: 8}
}

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
	tp.RangeNM = 8
	if ap, ok := av.DB.LookupAirport(c.State.PrimaryAirport); ok {
		tp.Center = ap.Location
	} else {
		tp.Center = c.State.Center
	}
}

func (tp *TowerCabPane) CanTakeKeyboardFocus() bool { return false }

func (tp *TowerCabPane) Draw(ctx *panes.Context, cb *renderer.CommandBuffer) {
	ctx.SetWindowCoordinateMatrices(cb)

	if ctx.Mouse != nil {
		if ctx.Mouse.Wheel[1] != 0 {
			// Positive wheel motion zooms in.
			factor := float32(1)
			if ctx.Mouse.Wheel[1] > 0 {
				factor = 0.85
			} else {
				factor = 1.18
			}
			tp.RangeNM = math.Clamp(tp.RangeNM*factor, float32(1), float32(40))
		}
	}

	transforms := radar.GetScopeTransformations(ctx.PaneExtent, ctx.MagneticVariation,
		ctx.NmPerLongitude, tp.Center, tp.RangeNM, 0)

	lines := renderer.GetLinesDrawBuilder()
	defer renderer.ReturnLinesDrawBuilder(lines)
	triangles := renderer.GetTrianglesDrawBuilder()
	defer renderer.ReturnTrianglesDrawBuilder(triangles)
	text := renderer.GetTextDrawBuilder()
	defer renderer.ReturnTextDrawBuilder(text)

	// Crosshair marks the primary airport reference point until airport
	// geometry is added.
	center := [2]float32{ctx.PaneExtent.Width() / 2, ctx.PaneExtent.Height() / 2}
	lines.AddLine([2]float32{center[0] - 12, center[1]}, [2]float32{center[0] + 12, center[1]})
	lines.AddLine([2]float32{center[0], center[1] - 12}, [2]float32{center[0], center[1] + 12})

	style := renderer.TextStyle{Font: tp.font, Color: renderer.RGB{R: 0.9, G: 0.9, B: 0.9}}
	text.AddText(fmt.Sprintf("TOWER CAB  %s  %.1f NM", ctx.Client.State.PrimaryAirport, tp.RangeNM),
		[2]float32{12, ctx.PaneExtent.Height() - 24}, style)

	for _, trk := range ctx.Client.State.Tracks {
		p := transforms.WindowFromLatLongP(trk.Location)
		if !ctx.PaneExtent.Inside(p) {
			continue
		}

		// Top-down aircraft triangle. Heading is magnetic, matching the map.
		h := math.Radians(float32(trk.Heading))
		forward := [2]float32{math.Sin(h), math.Cos(h)}
		right := [2]float32{forward[1], -forward[0]}
		nose := math.Add2f(p, math.Scale2f(forward, 9))
		left := math.Add2f(math.Add2f(p, math.Scale2f(forward, -6)), math.Scale2f(right, -5))
		rightPt := math.Add2f(math.Add2f(p, math.Scale2f(forward, -6)), math.Scale2f(right, 5))
		triangles.AddTriangle(nose, left, rightPt)

		text.AddText(string(trk.ADSBCallsign), math.Add2f(p, [2]float32{10, 8}), style)
	}

	cb.SetRGB(renderer.RGB{R: 0.25, G: 0.35, B: 0.25})
	lines.GenerateCommands(cb)
	cb.SetRGB(renderer.RGB{R: 0.95, G: 0.95, B: 0.95})
	triangles.GenerateCommands(cb)
	text.GenerateCommands(cb)
}
