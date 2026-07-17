// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package tower

import (
	"strings"

	av "github.com/mmp/vice/aviation"
	"github.com/mmp/vice/math"
	"github.com/mmp/vice/radar"
	"github.com/mmp/vice/renderer"
)

const nominalRunwayWidthFeet = float32(150)

func drawAirport(ap av.FAAAirport, transforms radar.ScopeTransformations,
	fills *renderer.ColoredTrianglesDrawBuilder, lines *renderer.LinesDrawBuilder,
	text *renderer.TextDrawBuilder, font *renderer.Font) {
	labelStyle := renderer.TextStyle{Font: font, Color: renderer.RGB{R: 0.86, G: 0.88, B: 0.90}}
	seen := make(map[string]bool)

	for _, runway := range ap.Runways {
		base := av.RunwayID(runway.Id).Base()
		if seen[base] {
			continue
		}

		opposite, ok := av.LookupOppositeRunway(ap.Id, base)
		if !ok {
			continue
		}

		oppBase := av.RunwayID(opposite.Id).Base()
		seen[base] = true
		seen[oppBase] = true

		p0 := transforms.WindowFromLatLongP(runway.Threshold)
		p1 := transforms.WindowFromLatLongP(opposite.Threshold)
		direction := math.Normalize2f(math.Sub2f(p1, p0))
		perpendicular := [2]float32{-direction[1], direction[0]}

		// Scale the nominal FAA runway width against the runway's real length,
		// so pavement grows and shrinks naturally with camera zoom.
		lengthNM := math.NMDistance2LL(runway.Threshold, opposite.Threshold)
		lengthPixels := math.Distance2f(p0, p1)
		halfWidthPixels := float32(4)
		if lengthNM > 0 {
			widthNM := nominalRunwayWidthFeet / math.NauticalMilesToFeet
			halfWidthPixels = math.Max(float32(2), lengthPixels*widthNM/lengthNM/2)
		}
		offset := math.Scale2f(perpendicular, halfWidthPixels)
		q0 := math.Add2f(p0, offset)
		q1 := math.Add2f(p1, offset)
		q2 := math.Sub2f(p1, offset)
		q3 := math.Sub2f(p0, offset)

		fills.AddQuad(q0, q1, q2, q3, renderer.RGB{R: 0.18, G: 0.19, B: 0.20})

		// Pavement edges and centerline.
		lines.AddLine(q0, q1)
		lines.AddLine(q3, q2)
		lines.AddLine(p0, p1)

		drawThresholdBar(p0, p1, halfWidthPixels, lines)
		drawThresholdBar(p1, p0, halfWidthPixels, lines)

		text.AddTextCentered(strings.TrimSpace(base), runwayLabelPosition(p0, p1, halfWidthPixels), labelStyle)
		text.AddTextCentered(strings.TrimSpace(oppBase), runwayLabelPosition(p1, p0, halfWidthPixels), labelStyle)
	}

	drawReferencePoint(transforms.WindowFromLatLongP(ap.Location), lines)
	text.AddTextCentered(ap.Id, transforms.WindowFromLatLongP(ap.Location),
		renderer.TextStyle{Font: font, Color: renderer.RGB{R: 0.96, G: 0.96, B: 0.96}})
}

func drawThresholdBar(threshold, opposite [2]float32, halfWidth float32, lines *renderer.LinesDrawBuilder) {
	direction := math.Normalize2f(math.Sub2f(opposite, threshold))
	perpendicular := [2]float32{-direction[1], direction[0]}
	width := math.Max(float32(5), halfWidth*0.9)
	lines.AddLine(
		math.Add2f(threshold, math.Scale2f(perpendicular, -width)),
		math.Add2f(threshold, math.Scale2f(perpendicular, width)),
	)
}

func runwayLabelPosition(threshold, opposite [2]float32, halfWidth float32) [2]float32 {
	direction := math.Normalize2f(math.Sub2f(opposite, threshold))
	return math.Add2f(threshold, math.Scale2f(direction, math.Max(float32(18), halfWidth*2.2)))
}

func drawReferencePoint(center [2]float32, lines *renderer.LinesDrawBuilder) {
	const radius = float32(7)
	lines.AddLine([2]float32{center[0] - radius, center[1]}, [2]float32{center[0] + radius, center[1]})
	lines.AddLine([2]float32{center[0], center[1] - radius}, [2]float32{center[0], center[1] + radius})
}
