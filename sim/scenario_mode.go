// sim/scenario_mode.go
// Copyright(c) 2022-2026 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package sim

import (
	"fmt"
	"strings"
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

func ParseScenarioMode(s string) (ScenarioMode, error) {
	mode := ScenarioMode(strings.ToLower(strings.TrimSpace(s)))
	if !mode.Valid() {
		return "", fmt.Errorf("unknown scenario mode %q (expected stars, eram, or tower)", s)
	}
	return mode, nil
}
