// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package tower

import (
	"github.com/mmp/vice/math"

	"github.com/AllenDang/cimgui-go/imgui"
	"github.com/mmp/vice/panes"
	"github.com/mmp/vice/platform"
	"github.com/mmp/vice/radar"
)

const (
	defaultRangeNM = float32(8)

	// Allow a much closer surface view, comparable to CRC.
	minimumRangeNM = float32(0.05)
	maximumRangeNM = float32(40)

	// Normal wheel movement is deliberately fine-grained. Holding Ctrl uses
	// larger steps for quickly moving between airport-wide and close views.
	zoomFineInFactor    = float32(0.95)
	zoomFineOutFactor   = float32(1.0526316)
	zoomCoarseInFactor  = float32(0.85)
	zoomCoarseOutFactor = float32(1.18)
)

// Camera stores the Tower Cab view independently of the STARS view.
type Camera struct {
	Center      math.Point2LL
	Home        math.Point2LL
	RangeNM     float32
	HomeRangeNM float32
	RotationDeg float32
}

func NewCamera() Camera { return Camera{RangeNM: defaultRangeNM} }

func (c *Camera) Reset(center math.Point2LL) {
	c.ResetView(center, defaultRangeNM)
}

// ResetView establishes the current view and the view restored by a
// double-right-click.
func (c *Camera) ResetView(center math.Point2LL, rangeNM float32) {
	c.Center = center
	c.Home = center
	c.RangeNM = math.Clamp(rangeNM, minimumRangeNM, maximumRangeNM)
	c.HomeRangeNM = c.RangeNM
	c.RotationDeg = 0
}

func (c *Camera) Transforms(ctx *panes.Context) radar.ScopeTransformations {
	return radar.GetScopeTransformations(ctx.PaneExtent, ctx.MagneticVariation,
		ctx.NmPerLongitude, c.Center, c.RangeNM, c.RotationDeg)
}

// HandleMouse updates zoom and pan. It returns true when the transforms need
// to be rebuilt before drawing.
func (c *Camera) HandleMouse(ctx *panes.Context, transforms radar.ScopeTransformations, config *TowerCabConfig) bool {
	if ctx.Mouse == nil {
		return false
	}

	changed := false

	if config.MousePanZoomEnabled {
		if wheel := ctx.Mouse.Wheel[1]; wheel != 0 {
			before := transforms.LatLongFromWindowP(ctx.Mouse.Pos)

			inFactor := zoomFineInFactor
			outFactor := zoomFineOutFactor
			if imgui.CurrentIO().KeyCtrl() {
				inFactor = zoomCoarseInFactor
				outFactor = zoomCoarseOutFactor
			}

			factor := inFactor
			if wheel > 0 {
				// Preserve the existing Tower Cab wheel direction.
				factor = outFactor
			}
			c.RangeNM = math.Clamp(c.RangeNM*factor, minimumRangeNM, maximumRangeNM)

			if config.ZoomToCursor {
				// Keep the point under the cursor stationary while zooming.
				afterTransforms := c.Transforms(ctx)
				after := afterTransforms.LatLongFromWindowP(ctx.Mouse.Pos)
				c.Center = math.Add2f(c.Center, math.Sub2f(before, after))
			}
			changed = true
		}

		if ctx.Mouse.Dragging[platform.MouseButtonSecondary] {
			llDelta := transforms.LatLongFromWindowV(ctx.Mouse.DragDelta)
			c.Center = math.Sub2f(c.Center, llDelta)
			changed = true
		}
	}

	if ctx.Mouse.DoubleClicked[platform.MouseButtonSecondary] {
		c.Center = c.Home
		c.RangeNM = c.HomeRangeNM
		changed = true
	}

	return changed
}
