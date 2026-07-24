// Copyright(c) 2022-2026 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX-License-Identifier: GPL-3.0-only

package server

import (
	"testing"

	"github.com/mmp/vice/sim"
)

func TestBuildScenarioHierarchy(t *testing.T) {
	catalogs := map[string]map[string]*ScenarioCatalog{
		"M98": {
			"MSP": {
				ARTCC: "ZMP",
				Scenarios: map[string]*ScenarioSpec{
					"South Flow": {Mode: sim.ScenarioModeSTARS, PrimaryAirport: "KMSP"},
					"Day Ops":    {Mode: sim.ScenarioModeTower, PrimaryAirport: "KMSP"},
				},
			},
		},
	}

	h := BuildScenarioHierarchy(catalogs)
	zmp := h.ARTCCs["ZMP"]
	if zmp == nil {
		t.Fatal("missing ZMP")
	}
	if zmp.Facilities["M98"].Positions["STARS"] == nil {
		t.Fatal("missing ZMP -> M98 -> STARS")
	}
	if zmp.Facilities["MSP"].Positions["Tower"] == nil {
		t.Fatal("missing ZMP -> MSP -> Tower")
	}
}
