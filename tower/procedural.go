// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package tower

import (
	av "github.com/mmp/vice/aviation"
	"github.com/mmp/vice/math"
	"github.com/mmp/vice/radar"
	"github.com/mmp/vice/renderer"
)

// ProceduralAirportVisualSource adapts the existing runway renderer to the
// AirportVisualSource interface. This is the reliable fallback when no imported
// airport layout is available.
type ProceduralAirportVisualSource struct {
	airportID string
	airport   av.FAAAirport
	found     bool
	fallback  math.Point2LL
}

func NewProceduralAirportVisualSource(
	airportID string,
	fallback math.Point2LL,
) *ProceduralAirportVisualSource {
	airport, found := av.DB.LookupAirport(airportID)
	return &ProceduralAirportVisualSource{
		airportID: airportID,
		airport:   airport,
		found:     found,
		fallback:  fallback,
	}
}

func (s *ProceduralAirportVisualSource) Name() string {
	return "Procedural"
}

func (s *ProceduralAirportVisualSource) Center() math.Point2LL {
	if s.found {
		return s.airport.Location
	}
	return s.fallback
}

func (s *ProceduralAirportVisualSource) Draw(
	_ float32,
	transforms radar.ScopeTransformations,
	fills *renderer.ColoredTrianglesDrawBuilder,
	lines *renderer.LinesDrawBuilder,
	text *renderer.TextDrawBuilder,
	font *renderer.Font,
) {
	if s.found {
		drawAirport(s.airport, transforms, fills, lines, text, font)
		return
	}

	drawReferencePoint(transforms.WindowFromLatLongP(s.fallback), lines)
}
