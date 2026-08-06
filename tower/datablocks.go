// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package tower

import (
	"fmt"
	gomath "math"
	"strings"

	av "github.com/mmp/vice/aviation"
	"github.com/mmp/vice/math"
	"github.com/mmp/vice/platform"
	"github.com/mmp/vice/radar"
	"github.com/mmp/vice/renderer"
	"github.com/mmp/vice/sim"
)

const towerAircraftSlewRadius = float32(22)

type DatablockState struct {
	Position int
	Distance int
	Offset   [2]float32
}

type DatablockCommand struct {
	Text string
}

type DatablockBackgroundDrawBuilder struct {
	vertices [][2]float32
	indices  []int32
	color    renderer.RGB
	alpha    float32
}

func NewDatablockBackgroundDrawBuilder() *DatablockBackgroundDrawBuilder {
	return &DatablockBackgroundDrawBuilder{}
}

func (builder *DatablockBackgroundDrawBuilder) AddQuad(
	left float32,
	right float32,
	bottom float32,
	top float32,
	color renderer.RGB,
	alpha float32,
) {
	base := int32(len(builder.vertices))
	builder.vertices = append(builder.vertices,
		[2]float32{left, bottom},
		[2]float32{right, bottom},
		[2]float32{right, top},
		[2]float32{left, top},
	)
	builder.indices = append(builder.indices,
		base, base+1, base+2,
		base, base+2, base+3,
	)
	builder.color = color
	builder.alpha = alpha
}

func (builder *DatablockBackgroundDrawBuilder) GenerateCommands(cb *renderer.CommandBuffer) {
	if len(builder.vertices) == 0 {
		return
	}

	vertexBuffer := cb.Float2Buffer(builder.vertices)
	indexBuffer := cb.IntBuffer(builder.indices)

	cb.Blend()
	cb.SetRGBA(renderer.RGBA{
		R: builder.color.R,
		G: builder.color.G,
		B: builder.color.B,
		A: builder.alpha,
	})
	cb.VertexArray(vertexBuffer, 2, 2*4)
	cb.DrawTriangles(indexBuffer, len(builder.indices))
	cb.DisableVertexArray()
}

type datablockBounds struct {
	Left   float32
	Right  float32
	Bottom float32
	Top    float32
}

func defaultDatablockState() DatablockState {
	return DatablockState{Position: 6, Distance: 1}
}

func datablockStateFor(states map[string]DatablockState, callsign string) DatablockState {
	if state, ok := states[callsign]; ok {
		return normalizeDatablockState(state)
	}
	return defaultDatablockState()
}

func normalizeDatablockState(state DatablockState) DatablockState {
	switch state.Position {
	case 1, 2, 3, 4, 6, 7, 8, 9:
	default:
		state.Position = 6
	}
	if state.Distance < 0 || state.Distance > 5 {
		state.Distance = 1
	}
	return state
}

func (command *DatablockCommand) AcceptRune(r rune) {
	switch {
	case r == '/':
		command.Text = "/"
	case command.Text == "/" && r >= '0' && r <= '5':
		command.Text = "/" + string(r)
	case r == '1' || r == '2' || r == '3' || r == '4' ||
		r == '6' || r == '7' || r == '8' || r == '9':
		command.Text = string(r)
	}
}

func (command DatablockCommand) Valid() bool {
	if len(command.Text) == 1 {
		return strings.ContainsRune("12346789", rune(command.Text[0]))
	}
	return len(command.Text) == 2 &&
		command.Text[0] == '/' &&
		command.Text[1] >= '0' && command.Text[1] <= '5'
}

func (command *DatablockCommand) Clear() {
	command.Text = ""
}

func (command DatablockCommand) Apply(state DatablockState) DatablockState {
	state = normalizeDatablockState(state)
	if !command.Valid() {
		return state
	}

	if command.Text[0] == '/' {
		state.Distance = int(command.Text[1] - '0')
	} else {
		state.Position = int(command.Text[0] - '0')
	}
	return state
}

func findTowerAircraftAt(
	tracks map[av.ADSBCallsign]*sim.Track,
	airportCenter math.Point2LL,
	transforms radar.ScopeTransformations,
	mouse *platform.MouseState,
) string {
	if mouse == nil {
		return ""
	}

	bestCallsign := ""
	bestDistanceSquared := towerAircraftSlewRadius * towerAircraftSlewRadius

	for callsign, track := range tracks {
		if track == nil || strings.HasPrefix(callsign.String(), "__") {
			continue
		}
		if math.NMDistance2LL(airportCenter, track.Location) > towerTrafficRadius {
			continue
		}

		position := transforms.WindowFromLatLongP(track.Location)
		dx := mouse.Pos[0] - position[0]
		dy := mouse.Pos[1] - position[1]
		distanceSquared := dx*dx + dy*dy
		if distanceSquared <= bestDistanceSquared {
			bestDistanceSquared = distanceSquared
			bestCallsign = callsign.String()
		}
	}

	return bestCallsign
}

type towerDatablockLayout struct {
	Lines         []string
	TextPositions [][2]float32
	Bounds        datablockBounds
}

func buildTowerDatablockLayout(
	lines []string,
	aircraftPosition [2]float32,
	state DatablockState,
	font *renderer.Font,
	config *TowerCabConfig,
) towerDatablockLayout {
	const horizontalPadding = float32(1.0)
	const verticalPadding = float32(0.0)

	// LayoutBounds is reliable for line width, but its height can reflect only
	// the glyphs present in the measured string. Use the selected font size for
	// the full text-run height so letters, numbers, and descenders always fit.
	textRunHeight := float32(config.DataBlockFontSize)
	lineAdvance := textRunHeight * 0.70

	textWidth := float32(0)
	for _, line := range lines {
		textWidth = max(textWidth, font.LayoutBounds(line, 0).Width())
	}

	textHeight := textRunHeight
	if len(lines) > 1 {
		textHeight += lineAdvance * float32(len(lines)-1)
	}

	// towerDatablockPosition returns the top-left text origin. The background
	// is then derived from that exact origin and the completed text layout.
	textOrigin := towerDatablockPosition(
		aircraftPosition,
		normalizeDatablockState(state),
		textWidth,
		textHeight,
	)
	textOrigin[0] += state.Offset[0]
	textOrigin[1] += state.Offset[1]

	positions := make([][2]float32, len(lines))
	for i := range lines {
		positions[i] = [2]float32{
			textOrigin[0],
			textOrigin[1] - lineAdvance*float32(i),
		}
	}

	return towerDatablockLayout{
		Lines:         lines,
		TextPositions: positions,
		Bounds: datablockBounds{
			Left:   textOrigin[0] - horizontalPadding,
			Right:  textOrigin[0] + textWidth + horizontalPadding,
			Bottom: textOrigin[1] - textHeight - verticalPadding,
			Top:    textOrigin[1] + verticalPadding,
		},
	}
}

func findTowerDatablockAt(
	tracks map[av.ADSBCallsign]*sim.Track,
	airportCenter math.Point2LL,
	transforms radar.ScopeTransformations,
	mouse *platform.MouseState,
	font *renderer.Font,
	config *TowerCabConfig,
	states map[string]DatablockState,
) string {
	if mouse == nil || !config.ShowDataBlocks {
		return ""
	}

	for callsign, track := range tracks {
		if track == nil || strings.HasPrefix(callsign.String(), "__") {
			continue
		}
		if math.NMDistance2LL(airportCenter, track.Location) > towerTrafficRadius {
			continue
		}

		callsignString := callsign.String()
		lines := towerDatablockLines(callsignString, track, config)
		if len(lines) == 0 {
			continue
		}

		position := transforms.WindowFromLatLongP(track.Location)
		layout := buildTowerDatablockLayout(
			lines,
			position,
			datablockStateFor(states, callsignString),
			font,
			config,
		)

		if mouse.Pos[0] >= layout.Bounds.Left &&
			mouse.Pos[0] <= layout.Bounds.Right &&
			mouse.Pos[1] >= layout.Bounds.Bottom &&
			mouse.Pos[1] <= layout.Bounds.Top {
			return callsignString
		}
	}

	return ""
}

func cleanupDatablockStates(
	tracks map[av.ADSBCallsign]*sim.Track,
	states map[string]DatablockState,
) {
	for callsign := range states {
		found := false
		for trackCallsign, track := range tracks {
			if track != nil && trackCallsign.String() == callsign {
				found = true
				break
			}
		}
		if !found {
			delete(states, callsign)
		}
	}
}

func drawTowerDatablock(
	callsign string,
	track *sim.Track,
	aircraftPosition [2]float32,
	state DatablockState,
	font *renderer.Font,
	backgrounds *DatablockBackgroundDrawBuilder,
	geometry *renderer.ColoredTrianglesDrawBuilder,
	text *renderer.TextDrawBuilder,
	config *TowerCabConfig,
) {
	lines := towerDatablockLines(callsign, track, config)
	if len(lines) == 0 {
		return
	}

	layout := buildTowerDatablockLayout(
		lines,
		aircraftPosition,
		state,
		font,
		config,
	)

	if config.ShowDataBlockBackground {
		backgrounds.AddQuad(
			layout.Bounds.Left,
			layout.Bounds.Right,
			layout.Bounds.Bottom,
			layout.Bounds.Top,
			renderer.RGB{R: 0, G: 0, B: 0},
			config.DataBlockBackgroundAlpha,
		)
	}

	state = normalizeDatablockState(state)
	if config.ShowDataBlockLeaderLines && state.Distance != 0 {
		end := nearestPointOnDatablock(aircraftPosition, layout.Bounds)
		addDatablockSegment(
			geometry,
			aircraftPosition,
			end,
			1.0,
			config.DataBlockColor,
		)
	}

	for i, line := range layout.Lines {
		text.AddText(line, layout.TextPositions[i], renderer.TextStyle{
			Font:  font,
			Color: config.DataBlockColor,
		})
	}
}

func towerDatablockLines(
	callsign string,
	track *sim.Track,
	config *TowerCabConfig,
) []string {
	var lines []string

	if callsignText := formatTowerCallsign(callsign, config.CallsignMode); callsignText != "" {
		lines = append(lines, callsignText)
	}

	var details []string
	if config.ShowAircraftType && track.FlightPlan != nil &&
		track.FlightPlan.AircraftType != "" {
		details = append(details, track.FlightPlan.AircraftType)
	}
	if config.ShowAltitude {
		details = append(details, fmt.Sprintf("%03d", int(track.TransponderAltitude/100)))
	}
	if len(details) > 0 {
		lines = append(lines, strings.Join(details, "  "))
	}

	return lines
}

func towerDatablockPosition(
	aircraft [2]float32,
	state DatablockState,
	width float32,
	height float32,
) [2]float32 {
	// CRC-style spacing. /0 places the box directly against/over the target;
	// /5 produces the long leader shown in the CRC reference.
	distances := [...]float32{0, 28, 52, 78, 106, 136}
	distance := distances[state.Distance]

	xDirection, yDirection := datablockDirection(state.Position)

	var x float32
	switch {
	case xDirection < 0:
		x = aircraft[0] - distance - width
	case xDirection > 0:
		x = aircraft[0] + distance
	default:
		x = aircraft[0] - width/2
	}

	var y float32
	switch {
	case yDirection > 0:
		y = aircraft[1] + distance + height
	case yDirection < 0:
		y = aircraft[1] - distance
	default:
		y = aircraft[1] + height/2
	}

	return [2]float32{x, y}
}

func nearestPointOnDatablock(point [2]float32, bounds datablockBounds) [2]float32 {
	return [2]float32{
		min(max(point[0], bounds.Left), bounds.Right),
		min(max(point[1], bounds.Bottom), bounds.Top),
	}
}

func addDatablockSegment(
	geometry *renderer.ColoredTrianglesDrawBuilder,
	start [2]float32,
	end [2]float32,
	width float32,
	color renderer.RGB,
) {
	dx := end[0] - start[0]
	dy := end[1] - start[1]
	length := float32(gomath.Sqrt(float64(dx*dx + dy*dy)))
	if length <= 0.001 {
		return
	}

	half := width / 2
	perpendicularX := -dy / length * half
	perpendicularY := dx / length * half

	geometry.AddQuad(
		[2]float32{start[0] + perpendicularX, start[1] + perpendicularY},
		[2]float32{end[0] + perpendicularX, end[1] + perpendicularY},
		[2]float32{end[0] - perpendicularX, end[1] - perpendicularY},
		[2]float32{start[0] - perpendicularX, start[1] - perpendicularY},
		color,
	)
}

func datablockDirection(position int) (float32, float32) {
	const diagonal = float32(1 / gomath.Sqrt2)

	switch position {
	case 1:
		return -diagonal, -diagonal
	case 2:
		return 0, -1
	case 3:
		return diagonal, -diagonal
	case 4:
		return -1, 0
	case 6:
		return 1, 0
	case 7:
		return -diagonal, diagonal
	case 8:
		return 0, 1
	case 9:
		return diagonal, diagonal
	default:
		return 1, 0
	}
}
