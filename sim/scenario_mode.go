// sim/scenario_mode.go
// Copyright(c) 2022-2026 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package sim

import (
	"fmt"
	"strings"

	av "github.com/mmp/vice/aviation"
)

// ScenarioMode identifies the controller environment used by a scenario.
// An empty JSON value is resolved by the server for backward compatibility.
type ScenarioMode string

const (
	ScenarioModeSTARS ScenarioMode = "stars"
	ScenarioModeERAM  ScenarioMode = "eram"
	ScenarioModeTower ScenarioMode = "tower"
)

func (m ScenarioMode) Valid() bool {
	switch m {
	case ScenarioModeSTARS, ScenarioModeERAM, ScenarioModeTower:
		return true
	default:
		return false
	}
}

func (m ScenarioMode) String() string { return string(m) }

// IsTerminal reports whether this sim runs a terminal flight plan
// environment (STARS) rather than an enroute one (ERAM). Tower cabs are
// terminal: an ATCT sits underneath a TRACON, so its flight plans follow
// STARS conventions.
//
// Tower facilities are catalogued with an "_TOWER" suffix, which av.DB
// does not recognize, so dispatching on the scenario's mode is both more
// direct and more accurate than inspecting the facility identifier. When
// ScenarioMode is empty — a sim saved before tower scenarios existed —
// this falls back to exactly the test that preceded it, so restored sims
// keep their original behavior.
func (s *CommonState) IsTerminal() bool {
	switch s.ScenarioMode {
	case ScenarioModeSTARS, ScenarioModeTower:
		return true
	case ScenarioModeERAM:
		return false
	default:
		return av.DB.IsTRACON(s.Facility)
	}
}

func ParseScenarioMode(s string) (ScenarioMode, error) {
	mode := ScenarioMode(strings.ToLower(strings.TrimSpace(s)))
	if !mode.Valid() {
		return "", fmt.Errorf("unknown scenario mode %q (expected stars, eram, or tower)", s)
	}
	return mode, nil
}
