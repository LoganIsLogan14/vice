// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

// Package tower implements top-down tower-controller displays.
package tower

import (
	"strings"

	"github.com/AllenDang/cimgui-go/imgui"
	av "github.com/mmp/vice/aviation"
	"github.com/mmp/vice/client"
	"github.com/mmp/vice/log"
	"github.com/mmp/vice/math"
	"github.com/mmp/vice/panes"
	"github.com/mmp/vice/platform"
	"github.com/mmp/vice/radar"
	"github.com/mmp/vice/renderer"
)

// TowerCabPane is VICE's top-down tower-cab display.
type TowerCabPane struct {
	camera           Camera
	font             *renderer.Font
	renderer         renderer.Renderer
	config           TowerCabConfig
	aircraftTextures map[string]uint32

	background          AirportBackgroundSource
	backgroundAirportID string

	visibility VisibilityLayer

	visualSource    AirportVisualSource
	visualAirportID string

	datablocks               map[string]DatablockState
	datablockCommand         DatablockCommand
	draggedDatablockCallsign string
}

func NewTowerCabPane() *TowerCabPane {
	return &TowerCabPane{
		camera:     NewCamera(),
		config:     DefaultTowerCabConfig(),
		datablocks: make(map[string]DatablockState),
	}
}

func (tp *TowerCabPane) Activate(r renderer.Renderer, p platform.Platform, lg *log.Logger) {
	tp.config.EnsureDefaults()
	tp.font = renderer.GetDefaultFont()

	if tp.renderer != nil {
		destroyAircraftTextures(tp.renderer, tp.aircraftTextures)
	}
	tp.renderer = r
	tp.aircraftTextures = loadAircraftTextures(r)
}

func (tp *TowerCabPane) LoadedSim(c *client.ControlClient, p platform.Platform, lg *log.Logger) {
	tp.resetView(c)
}

func (tp *TowerCabPane) ResetSim(c *client.ControlClient, p platform.Platform, lg *log.Logger) {
	tp.resetView(c)
}

func (tp *TowerCabPane) resetView(c *client.ControlClient) {
	airportID := c.State.PrimaryAirport

	tp.camera.Reset(c.State.Center)
	if ap, ok := av.DB.LookupAirport(airportID); ok {
		tp.camera.ResetView(ap.Location, initialAirportRange(ap))
	}

	tp.setVisualSource(airportID)
	tp.setBackgroundSource(airportID)

	tp.datablocks = make(map[string]DatablockState)
	tp.datablockCommand.Clear()
	tp.draggedDatablockCallsign = ""
}

func (tp *TowerCabPane) setVisualSource(airportID string) {
	if source, err := NewCRCAirportVisualSource(airportID, tp.camera.Home); err == nil {
		tp.visualSource = source
	} else {
		tp.visualSource = NewProceduralAirportVisualSource(airportID, tp.camera.Home)
	}
	tp.visualAirportID = airportID
}

func (tp *TowerCabPane) setBackgroundSource(airportID string) {
	if tp.background != nil {
		tp.background.Dispose()
		tp.background = nil
	}

	if tp.renderer != nil {
		if source, err := NewCRCAerialBackgroundSource(
			tp.renderer,
			airportID,
			&tp.config,
		); err == nil {
			tp.background = source
		}
	}
	tp.backgroundAirportID = airportID
}

func (tp *TowerCabPane) CanTakeKeyboardFocus() bool { return true }

func (tp *TowerCabPane) Draw(ctx *panes.Context, cb *renderer.CommandBuffer) {
	tp.config.EnsureDefaults()
	ctx.SetWindowCoordinateMatrices(cb)

	transforms := tp.camera.Transforms(ctx)
	if tp.camera.HandleMouse(ctx, transforms, &tp.config) {
		transforms = tp.camera.Transforms(ctx)
	}

	if tp.datablocks == nil {
		tp.datablocks = make(map[string]DatablockState)
	}
	tp.handleDatablockKeyboard(ctx)

	airportCenter := tp.visualSourceCenter(ctx)
	cleanupDatablockStates(ctx.Client.State.Tracks, tp.datablocks)
	tp.handleDatablockMouse(ctx, airportCenter, transforms)

	backgroundFill := renderer.GetColoredTrianglesDrawBuilder()
	defer renderer.ReturnColoredTrianglesDrawBuilder(backgroundFill)
	fills := renderer.GetColoredTrianglesDrawBuilder()
	defer renderer.ReturnColoredTrianglesDrawBuilder(fills)
	statusFill := renderer.GetColoredTrianglesDrawBuilder()
	defer renderer.ReturnColoredTrianglesDrawBuilder(statusFill)
	aircraftTriangles := renderer.GetColoredTrianglesDrawBuilder()
	defer renderer.ReturnColoredTrianglesDrawBuilder(aircraftTriangles)
	datablockBackgrounds := NewDatablockBackgroundDrawBuilder()
	aircraftSprites := renderer.GetTexturedQuadsDrawBuilder()
	defer renderer.ReturnTexturedQuadsDrawBuilder(aircraftSprites)
	lines := renderer.GetLinesDrawBuilder()
	defer renderer.ReturnLinesDrawBuilder(lines)
	surfaceText := renderer.GetTextDrawBuilder()
	defer renderer.ReturnTextDrawBuilder(surfaceText)
	statusText := renderer.GetTextDrawBuilder()
	defer renderer.ReturnTextDrawBuilder(statusText)

	airportID := ctx.Client.State.PrimaryAirport
	if tp.visualSource == nil || tp.visualAirportID != airportID {
		tp.setVisualSource(airportID)
	}
	if tp.backgroundAirportID != airportID {
		tp.setBackgroundSource(airportID)
	}

	tp.config.Normalize()

	backgroundFill.AddQuad(
		[2]float32{0, 0},
		[2]float32{ctx.PaneExtent.Width(), 0},
		[2]float32{ctx.PaneExtent.Width(), ctx.PaneExtent.Height()},
		[2]float32{0, ctx.PaneExtent.Height()},
		tp.config.BackgroundColor,
	)
	backgroundFill.GenerateCommands(cb)

	if tp.config.ShowAerial && tp.background != nil {
		tp.background.Draw(tp.camera.RangeNM, transforms, cb)
	}

	if tp.config.ShowCRCMap {
		tp.visualSource.Draw(tp.camera.RangeNM, transforms, fills, lines, surfaceText, tp.font)
	}

	// Draw the complete airport surface first so the visibility effect can
	// obscure both the aerial imagery and CRC geometry, matching CRC.
	fills.GenerateCommands(cb)
	cb.SetRGB(renderer.RGB{R: 0.66, G: 0.68, B: 0.70})
	cb.LineWidth(1.5, ctx.DPIScale)
	lines.GenerateCommands(cb)

	// Aircraft symbols and their data blocks are both emitted below the
	// visibility layer, matching CRC's low-visibility behavior.
	airportCenter = tp.visualSource.Center()
	aircraftCount := drawAircraft(
		ctx.Client.State.Tracks,
		airportCenter,
		tp.camera.RangeNM,
		transforms,
		aircraftTriangles,
		datablockBackgrounds,
		aircraftSprites,
		tp.aircraftTextures,
		surfaceText,
		&tp.config,
		tp.datablocks,
	)
	aircraftSprites.GenerateCommands(cb)
	datablockBackgrounds.GenerateCommands(cb)
	aircraftTriangles.GenerateCommands(cb)
	surfaceText.GenerateCommands(cb)

	visibilitySM := float32(10)
	if metar, ok := ctx.Client.State.METAR[airportID]; ok {
		if parsed, err := metar.Visibility(); err == nil {
			visibilitySM = parsed
		}
	}
	if tp.config.ShowVisibility {
		tp.visibility.Draw(
			visibilitySM,
			tp.visualSource.Center(),
			transforms,
			ctx.PaneExtent,
			cb,
			&tp.config,
		)
	}

	if tp.config.ShowStatusBar {
		drawStatusBar(ctx, airportID, aircraftCount, tp.datablockCommand.Text, statusFill, statusText, tp.font, &tp.config)
		statusFill.GenerateCommands(cb)
		statusText.GenerateCommands(cb)
	}
}

func (tp *TowerCabPane) handleDatablockMouse(
	ctx *panes.Context,
	airportCenter math.Point2LL,
	transforms radar.ScopeTransformations,
) {
	if ctx.Mouse == nil {
		return
	}

	mouse := ctx.Mouse

	if mouse.Clicked[platform.MouseButtonPrimary] {
		// Typed /0-/5 and numpad-position commands retain priority when the
		// controller clicks an aircraft target.
		if tp.datablockCommand.Valid() {
			clickedCallsign := findTowerAircraftAt(
				ctx.Client.State.Tracks,
				airportCenter,
				transforms,
				mouse,
			)
			if clickedCallsign != "" {
				state := datablockStateFor(tp.datablocks, clickedCallsign)
				tp.datablocks[clickedCallsign] = tp.datablockCommand.Apply(state)
				tp.datablockCommand.Clear()
				tp.draggedDatablockCallsign = ""
				return
			}
		}

		labelFont := renderer.GetFont(renderer.FontIdentifier{
			Name: renderer.RobotoMono,
			Size: tp.config.DataBlockFontSize,
		})
		if labelFont == nil {
			labelFont = tp.font
		}

		tp.draggedDatablockCallsign = findTowerDatablockAt(
			ctx.Client.State.Tracks,
			airportCenter,
			transforms,
			mouse,
			labelFont,
			&tp.config,
			tp.datablocks,
		)
	}

	if tp.draggedDatablockCallsign != "" &&
		mouse.Dragging[platform.MouseButtonPrimary] {
		state := datablockStateFor(
			tp.datablocks,
			tp.draggedDatablockCallsign,
		)
		state.Offset[0] += mouse.DragDelta[0]
		state.Offset[1] += mouse.DragDelta[1]
		tp.datablocks[tp.draggedDatablockCallsign] = state
	}
}

func (tp *TowerCabPane) visualSourceCenter(ctx *panes.Context) math.Point2LL {
	if tp.visualSource != nil {
		return tp.visualSource.Center()
	}
	return ctx.Client.State.Center
}

func (tp *TowerCabPane) handleDatablockKeyboard(ctx *panes.Context) {
	if !ctx.HaveFocus || ctx.Keyboard == nil {
		return
	}

	if ctx.Keyboard.WasPressed(imgui.KeyEscape) ||
		ctx.Keyboard.WasPressed(imgui.KeyBackspace) ||
		ctx.Keyboard.WasPressed(imgui.KeyDelete) {
		tp.datablockCommand.Clear()
		return
	}

	for _, r := range ctx.Keyboard.Input {
		tp.datablockCommand.AcceptRune(r)
	}
}

func drawStatusBar(
	ctx *panes.Context,
	airportID string,
	aircraftCount int,
	commandText string,
	fills *renderer.ColoredTrianglesDrawBuilder,
	text *renderer.TextDrawBuilder,
	font *renderer.Font,
	config *TowerCabConfig,
) {
	statusFont := renderer.GetFont(renderer.FontIdentifier{
		Name: renderer.RobotoMono,
		Size: config.StatusFontSize,
	})
	if statusFont == nil {
		statusFont = font
	}

	// The text builder's Y coordinate is the top edge of the text run in
	// window coordinates. Derive the bar thickness directly from the selected
	// font size, then place the text a fixed amount below the top of the bar.
	// This keeps the text inside the bar at every supported font size.
	const horizontalPadding = float32(8)
	const verticalPadding = float32(5)
	barHeight := float32(config.StatusFontSize) + 2*verticalPadding

	top := ctx.PaneExtent.Height()
	barBottom := top - barHeight
	fills.AddQuad(
		[2]float32{0, barBottom},
		[2]float32{ctx.PaneExtent.Width(), barBottom},
		[2]float32{ctx.PaneExtent.Width(), top},
		[2]float32{0, top},
		renderer.RGB{R: 0.04, G: 0.04, B: 0.04},
	)

	// AddText uses an inverted window Y axis here: the distance subtracted
	// from `top` becomes the text's screen-space offset from the top of the
	// pane. Center the actual rendered glyph height inside the bar instead of
	// estimating the offset from the requested point size.
	fontHeight := statusFont.LayoutBounds("Hg", 0).Height()
	textTopInset := max((barHeight-fontHeight)/2, float32(0))

	// Font bounds include space that does not perfectly match the visible
	// glyphs. Apply a small optical correction so the characters appear
	// centered rather than slightly high inside the bar.
	const opticalDownOffset = float32(3)
	y := top - textTopInset - opticalDownOffset
	x := horizontalPadding

	if ctx.Client.State.Paused {
		text.AddText("PAUSED", [2]float32{x, y}, renderer.TextStyle{
			Font:  statusFont,
			Color: renderer.RGB{R: 1, G: 0, B: 0},
		})
		x += statusFont.LayoutBounds("PAUSED", 0).Width() + 32
	}

	zulu := ctx.Client.State.SimTime.Time().UTC().Format("15:04:05Z")
	text.AddText(zulu, [2]float32{x, y}, renderer.TextStyle{
		Font:  statusFont,
		Color: config.StatusTextColor,
	})
	x += statusFont.LayoutBounds(zulu, 0).Width() + 32

	switch config.StatusMETAR {
	case StatusMETARSimplified:
		if metar, ok := ctx.Client.State.METAR[airportID]; ok {
			wind, altimeter := simplifiedMETAR(metar.Raw)
			if wind != "" {
				text.AddText(wind, [2]float32{x, y}, renderer.TextStyle{
					Font:  statusFont,
					Color: config.StatusTextColor,
				})
				x += statusFont.LayoutBounds(wind, 0).Width() + 32
			}
			if altimeter != "" {
				text.AddText(altimeter, [2]float32{x, y}, renderer.TextStyle{
					Font:  statusFont,
					Color: config.StatusTextColor,
				})
			}
		}

	case StatusMETARFull:
		if metar, ok := ctx.Client.State.METAR[airportID]; ok && metar.Raw != "" {
			fullMETAR := strings.TrimSpace(metar.Raw)
			fullMETAR = strings.TrimPrefix(fullMETAR, "METAR ")
			text.AddText(fullMETAR, [2]float32{x, y}, renderer.TextStyle{
				Font:  statusFont,
				Color: config.StatusTextColor,
			})
		}
	}

	if commandText != "" {
		commandTop := barBottom
		commandBottom := commandTop - barHeight
		fills.AddQuad(
			[2]float32{0, commandBottom},
			[2]float32{ctx.PaneExtent.Width(), commandBottom},
			[2]float32{ctx.PaneExtent.Width(), commandTop},
			[2]float32{0, commandTop},
			renderer.RGB{R: 0.025, G: 0.025, B: 0.025},
		)

		commandY := commandTop - textTopInset - opticalDownOffset
		text.AddText(commandText, [2]float32{horizontalPadding, commandY}, renderer.TextStyle{
			Font:  statusFont,
			Color: config.StatusTextColor,
		})
	}
}

// simplifiedMETAR returns only the wind group and altimeter setting.
// For example, "33009G20KT" and "29.99".
func simplifiedMETAR(raw string) (wind string, altimeter string) {
	for _, field := range strings.Fields(raw) {
		if wind == "" && strings.HasSuffix(field, "KT") {
			wind = field
			continue
		}
		if len(field) == 5 && field[0] == 'A' {
			digits := field[1:]
			if digits[0] >= '0' && digits[0] <= '9' &&
				digits[1] >= '0' && digits[1] <= '9' &&
				digits[2] >= '0' && digits[2] <= '9' &&
				digits[3] >= '0' && digits[3] <= '9' {
				altimeter = digits[:2] + "." + digits[2:]
			}
		}
	}
	return wind, altimeter
}
