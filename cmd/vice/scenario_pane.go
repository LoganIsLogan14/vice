// cmd/vice/scenario_pane.go
// Copyright(c) 2022-2024 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package main

import (
	av "github.com/mmp/vice/aviation"
	"github.com/mmp/vice/client"
	"github.com/mmp/vice/log"
	"github.com/mmp/vice/panes"
	"github.com/mmp/vice/platform"
)

// activePaneForScenario is the single dispatch point from simulation mode to
// presentation pane. Facility classification remains only as a compatibility
// fallback for saved simulations created before ScenarioMode was persisted.
func activePaneForScenario(config *Config, c *client.ControlClient) panes.Pane {
	isSTARSSim := av.DB.IsTRACON(c.State.Facility) || av.DB.IsATCT(c.State.Facility)
	return config.ActivePane(c.State.ScenarioMode, isSTARSSim)
}

// loadScenarioPanes initializes panes when restoring a saved simulation.
func loadScenarioPanes(
	config *Config,
	c *client.ControlClient,
	plat platform.Platform,
	lg *log.Logger,
) panes.Pane {
	activePane := activePaneForScenario(config, c)
	activePane.LoadedSim(c, plat, lg)

	// Tower Cab remains available as a companion pane in STARS and ERAM.
	// In Tower mode it is already active, so it must not be initialized twice.
	if activePane != config.TowerCabPane {
		config.TowerCabPane.LoadedSim(c, plat, lg)
	}

	return activePane
}

// resetScenarioPanes initializes panes for a newly connected simulation.
func resetScenarioPanes(
	config *Config,
	c *client.ControlClient,
	plat platform.Platform,
	lg *log.Logger,
) panes.Pane {
	activePane := activePaneForScenario(config, c)
	activePane.ResetSim(c, plat, lg)

	// Tower Cab remains available as a companion pane in STARS and ERAM.
	// In Tower mode it is already active, so it must not be initialized twice.
	if activePane != config.TowerCabPane {
		config.TowerCabPane.ResetSim(c, plat, lg)
	}

	return activePane
}
