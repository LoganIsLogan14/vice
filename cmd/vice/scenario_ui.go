// cmd/vice/scenario_ui.go
// Copyright(c) 2022-2024 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package main

import "github.com/mmp/vice/panes"

// scenarioUI is the UI boundary for a simulation family.
//
// The pane objects themselves remain in Config because they are persisted there,
// but ownership of mode-specific composition now lives in dedicated STARS,
// ERAM, and Tower implementations. Later passes can add toolbar items, windows,
// keyboard references, or other mode-specific behavior without putting mode
// checks back into ui.go.
type scenarioUI interface {
	ActivePane() panes.Pane
	SettingsPanes() []panes.UIDrawer
}

type baseScenarioUI struct {
	config     *Config
	activePane panes.Pane
}

func (ui *baseScenarioUI) ActivePane() panes.Pane {
	return ui.activePane
}

func (ui *baseScenarioUI) sharedSettingsPanes() []panes.UIDrawer {
	return collectSettingsPanes(
		ui.config.MessagesPane,
		ui.config.FlightStripPane,
	)
}

// starsScenarioUI owns the UI composition for STARS simulations.
type starsScenarioUI struct {
	baseScenarioUI
}

func (ui *starsScenarioUI) SettingsPanes() []panes.UIDrawer {
	return collectSettingsPanes(
		ui.sharedSettingsPanes(),
		ui.activePane,
		ui.config.TowerCabPane,
	)
}

// eramScenarioUI owns the UI composition for ERAM simulations.
type eramScenarioUI struct {
	baseScenarioUI
}

func (ui *eramScenarioUI) SettingsPanes() []panes.UIDrawer {
	return collectSettingsPanes(
		ui.sharedSettingsPanes(),
		ui.activePane,
		ui.config.TowerCabPane,
	)
}

// towerScenarioUI owns the UI composition for Tower simulations.
type towerScenarioUI struct {
	baseScenarioUI
}

func (ui *towerScenarioUI) SettingsPanes() []panes.UIDrawer {
	return collectSettingsPanes(
		ui.sharedSettingsPanes(),
		ui.activePane,
	)
}

// makeScenarioUI is the single factory that selects the UI implementation for
// the active simulation family.
//
// Pane identity is used here intentionally: activePaneForScenario has already
// performed the authoritative ScenarioMode dispatch, including compatibility
// fallback for older saved simulations. This keeps the compatibility policy in
// one place rather than duplicating it here.
func makeScenarioUI(config *Config, activePane panes.Pane) scenarioUI {
	base := baseScenarioUI{
		config:     config,
		activePane: activePane,
	}

	switch activePane {
	case config.TowerCabPane:
		return &towerScenarioUI{baseScenarioUI: base}
	case config.ERAMPane:
		return &eramScenarioUI{baseScenarioUI: base}
	default:
		return &starsScenarioUI{baseScenarioUI: base}
	}
}

// collectSettingsPanes converts pane values to UIDrawers, flattens any
// pre-collected []panes.UIDrawer values, and removes duplicates while
// preserving order.
func collectSettingsPanes(values ...any) []panes.UIDrawer {
	var result []panes.UIDrawer

	appendPane := func(draw panes.UIDrawer) {
		for _, existing := range result {
			if existing == draw {
				return
			}
		}
		result = append(result, draw)
	}

	for _, value := range values {
		switch value := value.(type) {
		case []panes.UIDrawer:
			for _, draw := range value {
				appendPane(draw)
			}

		case panes.UIDrawer:
			appendPane(value)
		}
	}

	return result
}
