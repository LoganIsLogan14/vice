// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package tower

import (
	"fmt"

	"github.com/mmp/vice/platform"
	"github.com/mmp/vice/renderer"

	"github.com/AllenDang/cimgui-go/imgui"
)

func (tp *TowerCabPane) DisplayName() string { return "Tower Cab" }

func (tp *TowerCabPane) DrawUI(p platform.Platform, globalConfig *platform.Config) {
	tp.config.EnsureDefaults()
	tp.config.ShowVisibility = true

	if imgui.CollapsingHeaderBoolPtr("Layers##tower", nil) {
		imgui.Checkbox("Show aerial imagery", &tp.config.ShowAerial)
		if tp.config.ShowAerial {
			imgui.Checkbox("Use high-resolution image", &tp.config.UseHighRes)
			imgui.SliderFloatV("Aerial brightness", &tp.config.AerialBrightness, 0, 1, "%.2f", 0)
		}
		imgui.Checkbox("Show CRC airport map", &tp.config.ShowCRCMap)
	}

	if imgui.CollapsingHeaderBoolPtr("Data blocks##tower", nil) {
		imgui.Checkbox("Show data blocks", &tp.config.ShowDataBlocks)

		if tp.config.ShowDataBlocks {
			if imgui.BeginComboV("Font size", fmt.Sprintf("%d", tp.config.DataBlockFontSize), 0) {
				for size := 8; size <= 32; size += 2 {
					label := fmt.Sprintf("%d", size)
					if imgui.SelectableBoolV(label, size == tp.config.DataBlockFontSize, 0, imgui.Vec2{}) {
						tp.config.DataBlockFontSize = size
					}
				}
				imgui.EndCombo()
			}

			callsignNames := []string{"Full", "Airline code only", "None"}
			current := callsignNames[int(tp.config.CallsignMode)]
			if imgui.BeginComboV("Callsign", current, 0) {
				for i, name := range callsignNames {
					if imgui.SelectableBoolV(name, i == int(tp.config.CallsignMode), 0, imgui.Vec2{}) {
						tp.config.CallsignMode = CallsignDisplayMode(i)
					}
				}
				imgui.EndCombo()
			}

			imgui.Checkbox("Show aircraft type", &tp.config.ShowAircraftType)
			imgui.Checkbox("Show altitude", &tp.config.ShowAltitude)
			imgui.Separator()
			imgui.Checkbox("Show data block background", &tp.config.ShowDataBlockBackground)
			if tp.config.ShowDataBlockBackground {
				imgui.SliderFloatV(
					"Background opacity",
					&tp.config.DataBlockBackgroundAlpha,
					0,
					1,
					"%.2f",
					0,
				)
			}
			imgui.Checkbox("Show leader lines", &tp.config.ShowDataBlockLeaderLines)

		}
	}

	if imgui.CollapsingHeaderBoolPtr("Status bar##tower", nil) {
		imgui.Checkbox("Show status bar", &tp.config.ShowStatusBar)
		if tp.config.ShowStatusBar {
			if imgui.BeginComboV("Font size##status", fmt.Sprintf("%d", tp.config.StatusFontSize), 0) {
				for _, size := range renderer.AvailableFontSizes(renderer.RobotoMono) {
					if size < 10 || size > 48 {
						continue
					}
					label := fmt.Sprintf("%d", size)
					if imgui.SelectableBoolV(label, size == tp.config.StatusFontSize, 0, imgui.Vec2{}) {
						tp.config.StatusFontSize = size
					}
				}
				imgui.EndCombo()
			}

			metarNames := []string{"None", "Simplified", "Full"}
			current := metarNames[int(tp.config.StatusMETAR)]
			if imgui.BeginComboV("METAR", current, 0) {
				for i, name := range metarNames {
					if imgui.SelectableBoolV(name, i == int(tp.config.StatusMETAR), 0, imgui.Vec2{}) {
						tp.config.StatusMETAR = StatusMETARMode(i)
					}
				}
				imgui.EndCombo()
			}
		}
	}

	if imgui.CollapsingHeaderBoolPtr("Camera##tower", nil) {
		imgui.Checkbox("Enable mouse pan/zoom", &tp.config.MousePanZoomEnabled)
		imgui.Checkbox("Zoom toward mouse cursor", &tp.config.ZoomToCursor)
	}

	if imgui.CollapsingHeaderBoolPtr("Colors##tower", nil) {
		drawRGBSettings("Background", &tp.config.BackgroundColor)
		drawRGBSettings("Aircraft", &tp.config.AircraftColor)
		drawRGBSettings("Data block text", &tp.config.DataBlockColor)
		drawRGBSettings("Status text", &tp.config.StatusTextColor)
	}

	imgui.Separator()
	if imgui.Button("Reset Tower Cab defaults") {
		tp.config = DefaultTowerCabConfig()
	}

	tp.config.Normalize()
}

func drawRGBSettings(label string, color *renderer.RGB) {
	value := [3]float32{color.R, color.G, color.B}

	imgui.Text(label)
	imgui.SameLine()
	if imgui.ColorEdit3V(
		"##"+label,
		&value,
		imgui.ColorEditFlagsNoInputs,
	) {
		color.R = value[0]
		color.G = value[1]
		color.B = value[2]
	}
}
