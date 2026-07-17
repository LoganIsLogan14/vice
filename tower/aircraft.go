// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package tower

import (
	"fmt"
	gomath "math"
	"strings"

	av "github.com/mmp/vice/aviation"
	"github.com/mmp/vice/math"
	"github.com/mmp/vice/radar"
	"github.com/mmp/vice/renderer"
	"github.com/mmp/vice/sim"
)

const (
	aircraftNosePixels = float32(10)
	aircraftTailPixels = float32(6)
	aircraftHalfWidth  = float32(5)
	towerTrafficRadius = float32(15)
)

// drawAircraft renders radar-visible traffic within the Tower Cab traffic
// radius. Ground simulation will later feed targets through this same path.
func drawAircraft(tracks map[av.ADSBCallsign]*sim.Track, airportCenter math.Point2LL,
	transforms radar.ScopeTransformations, triangles *renderer.ColoredTrianglesDrawBuilder,
	text *renderer.TextDrawBuilder, font *renderer.Font) int {
	count := 0
	for callsign, track := range tracks {
		if track == nil || strings.HasPrefix(callsign.String(), "__") {
			continue
		}
		if math.NMDistance2LL(airportCenter, track.Location) > towerTrafficRadius {
			continue
		}

		position := transforms.WindowFromLatLongP(track.Location)
		headingRadians := float64(track.Heading) * gomath.Pi / 180
		forward := [2]float32{float32(gomath.Sin(headingRadians)), float32(gomath.Cos(headingRadians))}
		right := [2]float32{forward[1], -forward[0]}

		nose := addScaled(position, forward, aircraftNosePixels)
		tailCenter := addScaled(position, forward, -aircraftTailPixels)
		leftTail := addScaled(tailCenter, right, -aircraftHalfWidth)
		rightTail := addScaled(tailCenter, right, aircraftHalfWidth)

		color := renderer.RGB{R: 0.92, G: 0.94, B: 0.96}
		triangles.AddTriangle(nose, leftTail, rightTail, color)

		labelPosition := [2]float32{position[0] + 11, position[1] + 9}
		labelStyle := renderer.TextStyle{Font: font, Color: color}
		text.AddText(callsign.String(), labelPosition, labelStyle)
		text.AddText(fmt.Sprintf("%03d  %03d", int(track.TransponderAltitude/100), int(track.Groundspeed+0.5)),
			[2]float32{labelPosition[0], labelPosition[1] - 12},
			renderer.TextStyle{Font: font, Color: renderer.RGB{R: 0.68, G: 0.72, B: 0.75}})
		count++
	}
	return count
}

func addScaled(p, v [2]float32, scale float32) [2]float32 {
	return [2]float32{p[0] + v[0]*scale, p[1] + v[1]*scale}
}
