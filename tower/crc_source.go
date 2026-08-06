// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package tower

import (
	"encoding/json"
	"fmt"
	math64 "math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/mmp/vice/math"
	"github.com/mmp/vice/radar"
	"github.com/mmp/vice/renderer"
	"github.com/rclancey/earcut"
)

const (
	crcPolygonSimplifyTolerance = 0.000003
	crcFullDetailRangeNM        = float32(0.5)
)

type crcDrawableKind int

const (
	crcDrawablePolygon crcDrawableKind = iota
	crcDrawableLineString
)

type crcPoint64 [2]float64

type crcVisualTriangle struct {
	points [3]crcPoint64
}

type crcDrawable struct {
	kind                crcDrawableKind
	color               renderer.RGB
	thickness           float32
	zIndex              int
	order               int
	simplifiedTriangles []crcVisualTriangle
	fullTriangles       []crcVisualTriangle
	points              []crcPoint64
}

type crcCachedMap struct {
	drawables []crcDrawable
}

var crcMapCache = struct {
	sync.RWMutex
	maps map[string]*crcCachedMap
}{
	maps: make(map[string]*crcCachedMap),
}

// CRCAirportVisualSource renders cached polygon and LineString features from a
// CRC Cab or ASDEX map.
type CRCAirportVisualSource struct {
	name   string
	center math.Point2LL
	layout *crcCachedMap
}

func NewCRCAirportVisualSource(
	airportID string,
	fallback math.Point2LL,
) (*CRCAirportVisualSource, error) {
	root, err := crcDataDirectory()
	if err != nil {
		return nil, err
	}

	match, err := findCRCMap(root, airportID)
	if err != nil {
		return nil, err
	}

	path := filepath.Join(root, "VideoMaps", match.ARTCC, match.Map.ID+".geojson")
	layout, err := loadCachedCRCMap(path)
	if err != nil {
		return nil, fmt.Errorf("load CRC map %q: %w", match.Map.Name, err)
	}
	if len(layout.drawables) == 0 {
		return nil, fmt.Errorf("CRC map %q contained no drawable features", match.Map.Name)
	}

	return &CRCAirportVisualSource{
		name:   "CRC " + match.Map.Name,
		center: fallback,
		layout: layout,
	}, nil
}

func (s *CRCAirportVisualSource) Name() string {
	return s.name
}

func (s *CRCAirportVisualSource) Center() math.Point2LL {
	return s.center
}

func (s *CRCAirportVisualSource) Draw(
	rangeNM float32,
	transforms radar.ScopeTransformations,
	fills *renderer.ColoredTrianglesDrawBuilder,
	lines *renderer.LinesDrawBuilder,
	text *renderer.TextDrawBuilder,
	font *renderer.Font,
) {
	for _, drawable := range s.layout.drawables {
		switch drawable.kind {
		case crcDrawablePolygon:
			triangles := drawable.simplifiedTriangles
			if rangeNM <= crcFullDetailRangeNM {
				triangles = drawable.fullTriangles
			}
			for _, triangle := range triangles {
				fills.AddTriangle(
					transforms.WindowFromLatLong64(triangle.points[0], s.center),
					transforms.WindowFromLatLong64(triangle.points[1], s.center),
					transforms.WindowFromLatLong64(triangle.points[2], s.center),
					drawable.color,
				)
			}

		case crcDrawableLineString:
			drawCRCLineString(
				drawable.points,
				s.center,
				drawable.thickness,
				drawable.color,
				transforms,
				fills,
			)
		}
	}
}

func loadCachedCRCMap(path string) (*crcCachedMap, error) {
	crcMapCache.RLock()
	cached := crcMapCache.maps[path]
	crcMapCache.RUnlock()
	if cached != nil {
		return cached, nil
	}

	loaded, err := loadCRCMap(path)
	if err != nil {
		return nil, err
	}

	crcMapCache.Lock()
	if existing := crcMapCache.maps[path]; existing != nil {
		crcMapCache.Unlock()
		return existing, nil
	}
	crcMapCache.maps[path] = loaded
	crcMapCache.Unlock()

	return loaded, nil
}

func loadCRCMap(path string) (*crcCachedMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read GeoJSON: %w", err)
	}

	var collection crcFeatureCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		return nil, fmt.Errorf("decode GeoJSON: %w", err)
	}

	drawables := make([]crcDrawable, 0, len(collection.Features))
	for featureIndex, feature := range collection.Features {
		color, err := parseCRCHexColor(feature.Properties.Color)
		if err != nil {
			return nil, fmt.Errorf("feature %d: %w", featureIndex, err)
		}

		switch feature.Geometry.Type {
		case "Polygon":
			simplified, full, err := loadCRCPolygonFeature(feature.Geometry.Coordinates)
			if err != nil {
				return nil, fmt.Errorf("feature %d Polygon: %w", featureIndex, err)
			}
			if len(full) == 0 {
				continue
			}
			drawables = append(drawables, crcDrawable{
				kind:                crcDrawablePolygon,
				color:               color,
				zIndex:              feature.Properties.ZIndex,
				order:               featureIndex,
				simplifiedTriangles: simplified,
				fullTriangles:       full,
			})

		case "LineString":
			points, err := loadCRCLineFeature(feature.Geometry.Coordinates)
			if err != nil {
				return nil, fmt.Errorf("feature %d LineString: %w", featureIndex, err)
			}
			if len(points) < 2 {
				continue
			}

			thickness := float32(1)
			if feature.Properties.Thickness != nil && *feature.Properties.Thickness > 0 {
				thickness = float32(*feature.Properties.Thickness)
			}

			drawables = append(drawables, crcDrawable{
				kind:      crcDrawableLineString,
				color:     color,
				thickness: thickness,
				zIndex:    feature.Properties.ZIndex,
				order:     featureIndex,
				points:    points,
			})
		}
	}

	sort.SliceStable(drawables, func(i, j int) bool {
		if drawables[i].zIndex != drawables[j].zIndex {
			return drawables[i].zIndex < drawables[j].zIndex
		}
		return drawables[i].order < drawables[j].order
	})

	return &crcCachedMap{drawables: drawables}, nil
}

func loadCRCPolygonFeature(
	raw json.RawMessage,
) ([]crcVisualTriangle, []crcVisualTriangle, error) {
	var coordinates [][][]float64
	if err := json.Unmarshal(raw, &coordinates); err != nil {
		return nil, nil, err
	}

	simplified, err := triangulateCRCPolygon(coordinates, true)
	if err != nil {
		return nil, nil, err
	}
	full, err := triangulateCRCPolygon(coordinates, false)
	if err != nil {
		return nil, nil, err
	}
	return simplified, full, nil
}

func triangulateCRCPolygon(
	coordinates [][][]float64,
	simplify bool,
) ([]crcVisualTriangle, error) {
	points, flatCoordinates, holeIndices :=
		prepareCRCPolygonForEarcut(coordinates, simplify)
	if len(points) < 3 {
		return nil, nil
	}

	indices, err := earcut.Earcut(flatCoordinates, holeIndices, 2)
	if err != nil {
		return nil, err
	}

	triangles := make([]crcVisualTriangle, 0, len(indices)/3)
	for i := 0; i+2 < len(indices); i += 3 {
		a, b, c := indices[i], indices[i+1], indices[i+2]
		if a < 0 || b < 0 || c < 0 ||
			a >= len(points) || b >= len(points) || c >= len(points) {
			return nil, fmt.Errorf("triangulation returned an invalid vertex index")
		}
		triangles = append(triangles, crcVisualTriangle{
			points: [3]crcPoint64{points[a], points[b], points[c]},
		})
	}
	return triangles, nil
}

func loadCRCLineFeature(raw json.RawMessage) ([]crcPoint64, error) {
	var coordinates [][]float64
	if err := json.Unmarshal(raw, &coordinates); err != nil {
		return nil, err
	}
	return crcCoordinatesToPoints(coordinates), nil
}

func drawCRCLineString(
	points []crcPoint64,
	origin math.Point2LL,
	thickness float32,
	color renderer.RGB,
	transforms radar.ScopeTransformations,
	fills *renderer.ColoredTrianglesDrawBuilder,
) {
	if len(points) < 2 {
		return
	}

	// CRC thickness values are screen-oriented. Keep very thin markings
	// readable while preventing unusually large values from overwhelming the
	// airport at wide zoom levels.
	halfWidth := min(max(float32(0.45), thickness*0.5), float32(4))

	windowPoints := make([][2]float32, len(points))
	for i, point := range points {
		windowPoints[i] = transforms.WindowFromLatLong64(point, origin)
	}

	drewSegment := false
	for i := 0; i+1 < len(windowPoints); i++ {
		start := windowPoints[i]
		end := windowPoints[i+1]
		delta := math.Sub2f(end, start)
		length := math.Length2f(delta)
		if length <= 0.001 {
			continue
		}

		direction := math.Scale2f(delta, 1/length)
		perpendicular := [2]float32{-direction[1], direction[0]}
		offset := math.Scale2f(perpendicular, halfWidth)

		fills.AddQuad(
			math.Add2f(start, offset),
			math.Add2f(end, offset),
			math.Sub2f(end, offset),
			math.Sub2f(start, offset),
			color,
		)
		drewSegment = true
	}

	if !drewSegment {
		return
	}

	// A small triangle fan at every vertex gives the polyline round joins and
	// round end caps. This removes the square blocks used by the first line
	// renderer and hides cracks between adjoining segment quads.
	const circleSegments = 8
	for _, center := range windowPoints {
		addCRCRoundJoin(center, halfWidth, circleSegments, color, fills)
	}
}

func addCRCRoundJoin(
	center [2]float32,
	radius float32,
	segments int,
	color renderer.RGB,
	fills *renderer.ColoredTrianglesDrawBuilder,
) {
	if radius <= 0 || segments < 3 {
		return
	}

	const twoPi = float32(6.283185307179586)
	previous := [2]float32{
		center[0] + radius,
		center[1],
	}
	for i := 1; i <= segments; i++ {
		angle := twoPi * float32(i) / float32(segments)
		next := [2]float32{
			center[0] + radius*float32(cosCRC(angle)),
			center[1] + radius*float32(sinCRC(angle)),
		}
		fills.AddTriangle(center, previous, next, color)
		previous = next
	}
}

func sinCRC(x float32) float64 {
	// The standard library trig functions operate on float64. Keeping these
	// wrappers local avoids scattering casts throughout the renderer.
	return math64.Sin(float64(x))
}

func cosCRC(x float32) float64 {
	return math64.Cos(float64(x))
}

func prepareCRCPolygonForEarcut(
	coordinates [][][]float64,
	simplify bool,
) ([]crcPoint64, []float64, []int) {
	var points []crcPoint64
	var flatCoordinates []float64
	var holeIndices []int
	validRingCount := 0

	for _, rawRing := range coordinates {
		ring := crcCoordinatesToPoints(rawRing)
		if simplify {
			ring = simplifyClosedCRCRing(ring, crcPolygonSimplifyTolerance)
			ring = removeCRCCollinearPoints(ring)
		}
		if len(ring) < 3 {
			continue
		}

		if validRingCount > 0 {
			holeIndices = append(holeIndices, len(points))
		}
		validRingCount++

		for _, point := range ring {
			points = append(points, point)
			flatCoordinates = append(flatCoordinates, point[0], point[1])
		}
	}
	return points, flatCoordinates, holeIndices
}

func crcCoordinatesToPoints(coordinates [][]float64) []crcPoint64 {
	points := make([]crcPoint64, 0, len(coordinates))
	for _, coordinate := range coordinates {
		if len(coordinate) < 2 {
			continue
		}

		point := crcPoint64{coordinate[0], coordinate[1]}
		if len(points) == 0 || points[len(points)-1] != point {
			points = append(points, point)
		}
	}

	if len(points) > 1 && points[0] == points[len(points)-1] {
		points = points[:len(points)-1]
	}
	return points
}

func simplifyClosedCRCRing(points []crcPoint64, tolerance float64) []crcPoint64 {
	if len(points) < 4 {
		return points
	}

	split := 1
	maxDistance := float64(0)
	for i := 1; i < len(points); i++ {
		distance := crcDistanceSquared(points[i], points[0])
		if distance > maxDistance {
			maxDistance = distance
			split = i
		}
	}

	first := simplifyCRCPolyline(points[:split+1], tolerance)
	secondInput := append([]crcPoint64(nil), points[split:]...)
	secondInput = append(secondInput, points[0])
	second := simplifyCRCPolyline(secondInput, tolerance)

	result := append([]crcPoint64(nil), first...)
	if len(second) > 2 {
		result = append(result, second[1:len(second)-1]...)
	}
	return result
}

func simplifyCRCPolyline(points []crcPoint64, tolerance float64) []crcPoint64 {
	if len(points) <= 2 {
		return append([]crcPoint64(nil), points...)
	}

	toleranceSquared := tolerance * tolerance
	index := -1
	maxDistance := float64(0)
	for i := 1; i < len(points)-1; i++ {
		distance := crcPointSegmentDistanceSquared(
			points[i],
			points[0],
			points[len(points)-1],
		)
		if distance > maxDistance {
			maxDistance = distance
			index = i
		}
	}

	if index < 0 || maxDistance <= toleranceSquared {
		return []crcPoint64{points[0], points[len(points)-1]}
	}

	left := simplifyCRCPolyline(points[:index+1], tolerance)
	right := simplifyCRCPolyline(points[index:], tolerance)
	return append(left[:len(left)-1], right...)
}

func crcPointSegmentDistanceSquared(p, a, b crcPoint64) float64 {
	px, py := p[0], p[1]
	ax, ay := a[0], a[1]
	bx, by := b[0], b[1]

	dx, dy := bx-ax, by-ay
	if dx == 0 && dy == 0 {
		dx, dy = px-ax, py-ay
		return dx*dx + dy*dy
	}

	t := ((px-ax)*dx + (py-ay)*dy) / (dx*dx + dy*dy)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}

	qx, qy := ax+t*dx, ay+t*dy
	dx, dy = px-qx, py-qy
	return dx*dx + dy*dy
}

func removeCRCCollinearPoints(points []crcPoint64) []crcPoint64 {
	if len(points) < 4 {
		return points
	}

	result := make([]crcPoint64, 0, len(points))
	for i := range points {
		previous := points[(i+len(points)-1)%len(points)]
		current := points[i]
		next := points[(i+1)%len(points)]

		if crcAbs(crcCross(previous, current, next)) > 1e-12 {
			result = append(result, current)
		}
	}
	return result
}

func crcCross(a, b, c crcPoint64) float64 {
	return (b[0]-a[0])*(c[1]-a[1]) -
		(b[1]-a[1])*(c[0]-a[0])
}

func crcDistanceSquared(a, b crcPoint64) float64 {
	dx := a[0] - b[0]
	dy := a[1] - b[1]
	return dx*dx + dy*dy
}

func crcAbs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func parseCRCHexColor(value string) (renderer.RGB, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(trimmed) != 6 {
		return renderer.RGB{}, fmt.Errorf("unsupported CRC color %q", value)
	}

	number, err := strconv.ParseUint(trimmed, 16, 24)
	if err != nil {
		return renderer.RGB{}, fmt.Errorf("parse CRC color %q: %w", value, err)
	}

	return renderer.RGB{
		R: float32((number>>16)&0xff) / 255,
		G: float32((number>>8)&0xff) / 255,
		B: float32(number&0xff) / 255,
	}, nil
}
