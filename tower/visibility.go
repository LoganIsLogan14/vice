// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package tower

import (
	"fmt"
	gomath "math"

	"github.com/mmp/vice/math"
	"github.com/mmp/vice/radar"
	"github.com/mmp/vice/renderer"
)

const (
	visibilityUnlimitedSM    = float32(10)
	visibilityOverlayAlpha   = float32(0.72)
	visibilityCircleSegments = 256
	visibilityFadeSteps      = 10
)

// VisibilityLayer draws a METAR-driven tower visibility mask. The reported
// statute-mile visibility is converted into a real geographic radius around
// the airport, projected through the same camera transformations as the map.
type VisibilityLayer struct {
	currentSM   float32
	targetSM    float32
	initialized bool
}

func (v *VisibilityLayer) Label() string {
	if !v.initialized || v.targetSM >= visibilityUnlimitedSM {
		return "10+SM"
	}
	if v.targetSM < 1 {
		return fmt.Sprintf("%.2fSM", v.targetSM)
	}
	return fmt.Sprintf("%.1fSM", v.targetSM)
}

func (v *VisibilityLayer) Draw(
	reportedSM float32,
	center math.Point2LL,
	transforms radar.ScopeTransformations,
	extent math.Extent2D,
	cb *renderer.CommandBuffer,
	config *TowerCabConfig,
) {
	reportedSM = max(reportedSM, float32(0.05))
	v.targetSM = reportedSM

	if !v.initialized {
		v.currentSM = reportedSM
		v.initialized = true
	} else {
		animationRate := float32(0.08)
		if config != nil {
			animationRate = config.VisibilityAnimationRate
		}
		v.currentSM += (reportedSM - v.currentSM) * animationRate
		if gomath.Abs(float64(reportedSM-v.currentSM)) < 0.002 {
			v.currentSM = reportedSM
		}
	}

	// CRC keeps a maximum-size visibility mask even when the METAR reports
	// 10SM or greater. Besides representing unrestricted visibility, this
	// hides the rectangular boundary of the satellite image.
	renderedSM := min(v.currentSM, visibilityUnlimitedSM)

	centerWindow := transforms.WindowFromLatLongP(center)

	// Treat the reported visibility as the OUTER edge of the feather, rather
	// than the beginning of it. At 10SM or greater, CRC appears to use an
	// approximately 10 NM maximum mask radius to hide the LowRes image edge.
	radiusNM := renderedSM * math.StatuteMilesToNauticalMiles
	if v.currentSM >= visibilityUnlimitedSM {
		radiusNM = 10
	}
	northPoint := math.Point2LL{
		center[0],
		center[1] + radiusNM/60,
	}
	northWindow := transforms.WindowFromLatLongP(northPoint)
	radiusPX := math.Length2f(math.Sub2f(northWindow, centerWindow))
	if radiusPX < 1 {
		radiusPX = 1
	}

	fadeFraction := float32(0.28)
	if config != nil {
		fadeFraction = config.VisibilityFadeFraction
	}
	drawVisibilityMask(centerWindow, radiusPX, extent, cb, fadeFraction)
}

func drawVisibilityMask(
	center [2]float32,
	fadeOuterRadius float32,
	extent math.Extent2D,
	cb *renderer.CommandBuffer,
	fadeFraction float32,
) {
	width := extent.Width()
	height := extent.Height()
	maxDX := max(center[0], width-center[0])
	maxDY := max(center[1], height-center[1])
	screenOuterRadius := float32(gomath.Hypot(float64(maxDX), float64(maxDY))) * 1.35

	// The METAR-scaled radius is the OUTER edge of the transition. Fade inward
	// from there so the image has already become solid gray before its
	// rectangular boundary can be exposed.
	fadeWidth := max(fadeOuterRadius*fadeFraction, float32(18))
	clearRadius := max(fadeOuterRadius-fadeWidth, float32(1))

	cb.Blend()
	drawVisibilityGradientAnnulus(center, clearRadius, fadeOuterRadius, cb)
	cb.DisableBlend()

	// Beyond the feather, replace everything with fully opaque gray.
	drawVisibilityAnnulus(center, fadeOuterRadius, screenOuterRadius, 1, cb)

	cb.SetRGB(renderer.RGB{R: 1, G: 1, B: 1})
}

func drawVisibilityGradientAnnulus(
	center [2]float32,
	innerRadius float32,
	outerRadius float32,
	cb *renderer.CommandBuffer,
) {
	vertices := make([][2]float32, 0, visibilityCircleSegments*2)
	colors := make([]byte, 0, visibilityCircleSegments*2*4)
	indices := make([]int32, 0, visibilityCircleSegments*6)

	const twoPi = float64(2 * gomath.Pi)
	const gray = byte(158)
	for i := 0; i < visibilityCircleSegments; i++ {
		angle := twoPi * float64(i) / visibilityCircleSegments
		cosine := float32(gomath.Cos(angle))
		sine := float32(gomath.Sin(angle))

		vertices = append(vertices,
			[2]float32{center[0] + innerRadius*cosine, center[1] + innerRadius*sine},
			[2]float32{center[0] + outerRadius*cosine, center[1] + outerRadius*sine},
		)

		// Inner edge is transparent; outer edge is fully opaque. The renderer
		// interpolates these RGBA values across the ring.
		colors = append(colors,
			gray, gray, gray, 0,
			gray, gray, gray, 255,
		)
	}

	for i := 0; i < visibilityCircleSegments; i++ {
		next := (i + 1) % visibilityCircleSegments
		inner0 := int32(2 * i)
		outer0 := inner0 + 1
		inner1 := int32(2 * next)
		outer1 := inner1 + 1
		indices = append(indices,
			inner0, outer0, outer1,
			inner0, outer1, inner1,
		)
	}

	vertexBuffer := cb.Float2Buffer(vertices)
	colorBuffer := cb.RawBuffer(colors)
	indexBuffer := cb.IntBuffer(indices)

	cb.VertexArray(vertexBuffer, 2, 2*4)
	cb.RGB8Array(colorBuffer, 4, 4)
	cb.DrawTriangles(indexBuffer, len(indices))
	cb.DisableColorArray()
	cb.DisableVertexArray()
}

func drawVisibilityAnnulus(
	center [2]float32,
	innerRadius float32,
	outerRadius float32,
	alpha float32,
	cb *renderer.CommandBuffer,
) {
	vertices := make([][2]float32, 0, visibilityCircleSegments*2)
	indices := make([]int32, 0, visibilityCircleSegments*6)

	const twoPi = float64(2 * gomath.Pi)
	for i := 0; i < visibilityCircleSegments; i++ {
		angle := twoPi * float64(i) / visibilityCircleSegments
		cosine := float32(gomath.Cos(angle))
		sine := float32(gomath.Sin(angle))
		vertices = append(vertices,
			[2]float32{center[0] + innerRadius*cosine, center[1] + innerRadius*sine},
			[2]float32{center[0] + outerRadius*cosine, center[1] + outerRadius*sine},
		)
	}

	for i := 0; i < visibilityCircleSegments; i++ {
		next := (i + 1) % visibilityCircleSegments
		inner0 := int32(2 * i)
		outer0 := inner0 + 1
		inner1 := int32(2 * next)
		outer1 := inner1 + 1
		indices = append(indices,
			inner0, outer0, outer1,
			inner0, outer1, inner1,
		)
	}

	vertexBuffer := cb.Float2Buffer(vertices)
	indexBuffer := cb.IntBuffer(indices)

	cb.SetRGBA(renderer.RGBA{
		R: 0.62,
		G: 0.62,
		B: 0.62,
		A: alpha,
	})
	cb.VertexArray(vertexBuffer, 2, 2*4)
	cb.DrawTriangles(indexBuffer, len(indices))
	cb.DisableVertexArray()
}
