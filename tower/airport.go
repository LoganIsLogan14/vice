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

const (
	nominalRunwayWidthFeet = float32(150)
	airportFramePadding    = float32(1.35)
	minimumAirportRangeNM  = float32(1.5)
)

var (
	runwayPavementColor = renderer.RGB{R: 0.18, G: 0.19, B: 0.20}
	runwayMarkingColor  = renderer.RGB{R: 0.82, G: 0.84, B: 0.86}
)

// initialAirportRange returns a camera range that fits every runway threshold
// around the airport reference point with a little margin.
func initialAirportRange(ap av.FAAAirport) float32 {
	maxDistanceNM := float32(0)
	for _, runway := range ap.Runways {
		distanceNM := math.NMDistance2LL(ap.Location, runway.Threshold)
		if distanceNM > maxDistanceNM {
			maxDistanceNM = distanceNM
		}
	}

	return max(minimumAirportRangeNM, maxDistanceNM*airportFramePadding)
}

func drawAirport(ap av.FAAAirport, transforms radar.ScopeTransformations,
	fills *renderer.ColoredTrianglesDrawBuilder, lines *renderer.LinesDrawBuilder,
	text *renderer.TextDrawBuilder, font *renderer.Font) {
	labelStyle := renderer.TextStyle{
		Font:  font,
		Color: renderer.RGB{R: 0.90, G: 0.92, B: 0.94},
	}

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

		lengthNM := math.NMDistance2LL(runway.Threshold, opposite.Threshold)
		lengthPixels := math.Distance2f(p0, p1)
		halfWidthPixels := float32(4)
		if lengthNM > 0 {
			widthNM := nominalRunwayWidthFeet / math.NauticalMilesToFeet
			halfWidthPixels = max(float32(2), lengthPixels*widthNM/lengthNM/2)
		}

		offset := math.Scale2f(perpendicular, halfWidthPixels)
		q0 := math.Add2f(p0, offset)
		q1 := math.Add2f(p1, offset)
		q2 := math.Sub2f(p1, offset)
		q3 := math.Sub2f(p0, offset)

		fills.AddQuad(q0, q1, q2, q3, runwayPavementColor)
		lines.AddLine(q0, q1)
		lines.AddLine(q3, q2)

		drawRunwayMarkings(p0, p1, direction, perpendicular, halfWidthPixels, lengthPixels, fills)

		text.AddTextCentered(strings.TrimSpace(base),
			runwayLabelPosition(p0, p1, halfWidthPixels), labelStyle)
		text.AddTextCentered(strings.TrimSpace(oppBase),
			runwayLabelPosition(p1, p0, halfWidthPixels), labelStyle)
	}

	drawReferencePoint(transforms.WindowFromLatLongP(ap.Location), lines)
	text.AddTextCentered(ap.Id, transforms.WindowFromLatLongP(ap.Location),
		renderer.TextStyle{
			Font:  font,
			Color: renderer.RGB{R: 0.96, G: 0.96, B: 0.96},
		})
}

func drawRunwayMarkings(start, end, direction, perpendicular [2]float32,
	halfWidth, lengthPixels float32, fills *renderer.ColoredTrianglesDrawBuilder) {
	if lengthPixels < 25 {
		return
	}

	// Keep markings readable at a wide range of zoom levels without allowing
	// them to grow beyond the pavement.
	markingHalfWidth := max(float32(0.7), halfWidth*0.10)
	edgeInset := max(float32(0.8), halfWidth*0.16)

	// Runway edge stripes.
	drawStrip(start, end, perpendicular, halfWidth-edgeInset, markingHalfWidth, fills)
	drawStrip(start, end, perpendicular, -(halfWidth - edgeInset), markingHalfWidth, fills)

	// Dashed centerline.
	dashLength := max(float32(5), lengthPixels*0.025)
	gapLength := dashLength
	startInset := max(float32(12), lengthPixels*0.08)
	endInset := lengthPixels - startInset
	for distance := startInset; distance < endInset; distance += dashLength + gapLength {
		dashEnd := min(distance+dashLength, endInset)
		addOrientedRect(
			pointAlong(start, direction, distance),
			pointAlong(start, direction, dashEnd),
			perpendicular,
			markingHalfWidth,
			runwayMarkingColor,
			fills,
		)
	}

	// Threshold bars and "piano keys" at both ends.
	drawThresholdMarkings(start, direction, perpendicular, halfWidth, fills)
	drawThresholdMarkings(end, math.Scale2f(direction, -1), perpendicular, halfWidth, fills)

	// Aiming point blocks and touchdown-zone bars.
	drawApproachMarkings(start, direction, perpendicular, halfWidth, lengthPixels, fills)
	drawApproachMarkings(end, math.Scale2f(direction, -1), perpendicular, halfWidth, lengthPixels, fills)
}

func drawStrip(start, end, perpendicular [2]float32, lateralOffset, halfThickness float32,
	fills *renderer.ColoredTrianglesDrawBuilder) {
	centerStart := math.Add2f(start, math.Scale2f(perpendicular, lateralOffset))
	centerEnd := math.Add2f(end, math.Scale2f(perpendicular, lateralOffset))
	addOrientedRect(centerStart, centerEnd, perpendicular, halfThickness, runwayMarkingColor, fills)
}

func drawThresholdMarkings(threshold, inward, perpendicular [2]float32,
	halfWidth float32, fills *renderer.ColoredTrianglesDrawBuilder) {
	barDistance := max(float32(4), halfWidth*0.9)
	barHalfLength := max(float32(1.2), halfWidth*0.18)
	barCenter := pointAlong(threshold, inward, barDistance)
	addOrientedRect(
		pointAlong(barCenter, inward, -barHalfLength),
		pointAlong(barCenter, inward, barHalfLength),
		perpendicular,
		halfWidth*0.82,
		runwayMarkingColor,
		fills,
	)

	// Eight threshold stripes, four on each side of the centerline.
	stripeStart := pointAlong(threshold, inward, barDistance+barHalfLength+1)
	stripeLength := max(float32(6), halfWidth*1.3)
	stripeEnd := pointAlong(stripeStart, inward, stripeLength)
	stripeHalfWidth := max(float32(0.45), halfWidth*0.055)
	spacing := halfWidth * 0.20
	for i := 1; i <= 4; i++ {
		offset := (float32(i) - 0.5) * spacing
		for _, side := range []float32{-1, 1} {
			s := math.Add2f(stripeStart, math.Scale2f(perpendicular, offset*side))
			e := math.Add2f(stripeEnd, math.Scale2f(perpendicular, offset*side))
			addOrientedRect(s, e, perpendicular, stripeHalfWidth, runwayMarkingColor, fills)
		}
	}
}

func drawApproachMarkings(threshold, inward, perpendicular [2]float32,
	halfWidth, lengthPixels float32, fills *renderer.ColoredTrianglesDrawBuilder) {
	// Scale placement from runway length so markings remain useful on airports
	// with very different runway dimensions.
	aimDistance := max(float32(28), lengthPixels*0.18)
	aimLength := max(float32(8), lengthPixels*0.045)
	aimHalfWidth := max(float32(0.9), halfWidth*0.16)
	aimOffset := halfWidth * 0.48

	aimStart := pointAlong(threshold, inward, aimDistance)
	aimEnd := pointAlong(aimStart, inward, aimLength)
	for _, side := range []float32{-1, 1} {
		s := math.Add2f(aimStart, math.Scale2f(perpendicular, aimOffset*side))
		e := math.Add2f(aimEnd, math.Scale2f(perpendicular, aimOffset*side))
		addOrientedRect(s, e, perpendicular, aimHalfWidth, runwayMarkingColor, fills)
	}

	// Three pairs of touchdown-zone bars between the threshold and aiming point.
	barHalfWidth := max(float32(0.65), halfWidth*0.10)
	barLength := max(float32(4), lengthPixels*0.018)
	for i := 1; i <= 3; i++ {
		distance := aimDistance * float32(i) / 4
		center := pointAlong(threshold, inward, distance)
		for _, side := range []float32{-1, 1} {
			offset := halfWidth * (0.34 + float32(i-1)*0.08)
			s := math.Add2f(pointAlong(center, inward, -barLength/2),
				math.Scale2f(perpendicular, offset*side))
			e := math.Add2f(pointAlong(center, inward, barLength/2),
				math.Scale2f(perpendicular, offset*side))
			addOrientedRect(s, e, perpendicular, barHalfWidth, runwayMarkingColor, fills)
		}
	}
}

func addOrientedRect(start, end, perpendicular [2]float32, halfWidth float32,
	color renderer.RGB, fills *renderer.ColoredTrianglesDrawBuilder) {
	offset := math.Scale2f(perpendicular, halfWidth)
	fills.AddQuad(
		math.Add2f(start, offset),
		math.Add2f(end, offset),
		math.Sub2f(end, offset),
		math.Sub2f(start, offset),
		color,
	)
}

func pointAlong(origin, direction [2]float32, distance float32) [2]float32 {
	return math.Add2f(origin, math.Scale2f(direction, distance))
}

func runwayLabelPosition(threshold, opposite [2]float32, halfWidth float32) [2]float32 {
	direction := math.Normalize2f(math.Sub2f(opposite, threshold))
	return math.Add2f(threshold,
		math.Scale2f(direction, max(float32(18), halfWidth*2.2)))
}

func drawReferencePoint(center [2]float32, lines *renderer.LinesDrawBuilder) {
	const radius = float32(7)
	lines.AddLine(
		[2]float32{center[0] - radius, center[1]},
		[2]float32{center[0] + radius, center[1]},
	)
	lines.AddLine(
		[2]float32{center[0], center[1] - radius},
		[2]float32{center[0], center[1] + radius},
	)
}
