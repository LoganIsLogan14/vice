// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package tower

import (
	"github.com/mmp/vice/math"
	"github.com/mmp/vice/radar"
	"github.com/mmp/vice/renderer"
)

// AirportVisualSource supplies the static visual layer beneath aircraft targets.
//
// A source answers only what the airport should look like. Taxi routing,
// hold-short logic, gates, and other operational behavior belong in a separate
// ground-network model.
type AirportVisualSource interface {
	Name() string
	Center() math.Point2LL
	Draw(
		rangeNM float32,
		transforms radar.ScopeTransformations,
		fills *renderer.ColoredTrianglesDrawBuilder,
		lines *renderer.LinesDrawBuilder,
		text *renderer.TextDrawBuilder,
		font *renderer.Font,
	)
}
