// cmd/vice/scenario_ui.go
// Copyright(c) 2022-2024 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package main

import "github.com/mmp/vice/panes"

// scenarioUI describes the pane-level UI owned by the active simulation mode.
// Keeping this interface small makes it possible to move mode-specific toolbar,
// window, and drawing behavior behind the same boundary in later passes.
type scenarioUI interface {
	ActivePane() panes.Pane
	SettingsPanes() []panes.UIDrawer
}

type configuredScenarioUI struct {
	activePane    panes.Pane
	settingsPanes []panes.UIDrawer
}

func (ui *configuredScenarioUI) ActivePane() panes.Pane {
	return ui.activePane
}

func (ui *configuredScenarioUI) SettingsPanes() []panes.UIDrawer {
	return ui.settingsPanes
}

// makeScenarioUI constructs the UI composition for the selected simulation
// mode. Shared panes are included first, followed by the active mode pane.
// Tower Cab remains available as a companion in STARS and ERAM, while Tower
// mode naturally avoids adding it twice.
func makeScenarioUI(config *Config, activePane panes.Pane) scenarioUI {
	ui := &configuredScenarioUI{activePane: activePane}

	appendSettingsPane := func(pane any) {
		draw, ok := pane.(panes.UIDrawer)
		if !ok {
			return
		}
		for _, existing := range ui.settingsPanes {
			if existing == draw {
				return
			}
		}
		ui.settingsPanes = append(ui.settingsPanes, draw)
	}

	appendSettingsPane(config.MessagesPane)
	appendSettingsPane(config.FlightStripPane)
	appendSettingsPane(activePane)
	appendSettingsPane(config.TowerCabPane)

	return ui
}
