// pkg/server/scenario_mode.go
// Copyright(c) 2022-2024 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package server

import (
	av "github.com/mmp/vice/aviation"
	"github.com/mmp/vice/sim"
	"github.com/mmp/vice/util"
)

// facility returns the internal facility identifier used by the scenario
// catalogs and facility-configuration lookup. Tower scenarios deliberately
// use an _TOWER suffix so an ATCT and a same-named TRACON can coexist.
func (sg *scenarioGroup) facility() string {
	switch sg.scenarioMode() {
	case sim.ScenarioModeTower:
		return sg.Tower + "_TOWER"
	case sim.ScenarioModeERAM:
		return sg.ARTCC
	default:
		return sg.TRACON
	}
}

// parentARTCC returns the ARTCC that owns this scenario group.
func (sg *scenarioGroup) parentARTCC() string {
	if sg.ARTCC != "" {
		return sg.ARTCC
	}
	if sg.Tower != "" {
		return av.DB.ARTCCForFacility(sg.Tower)
	}
	return av.DB.ARTCCForFacility(sg.TRACON)
}

// scenarioMode returns the controller environment implied by the scenario
// group's facility selector. Individual scenarios may still explicitly set
// their mode, but this is the canonical default for the group.
func (sg *scenarioGroup) scenarioMode() sim.ScenarioMode {
	switch {
	case sg.Tower != "":
		return sim.ScenarioModeTower
	case sg.ARTCC != "" && sg.TRACON == "":
		return sim.ScenarioModeERAM
	default:
		return sim.ScenarioModeSTARS
	}
}

func (sg *scenarioGroup) isTower() bool {
	return sg.scenarioMode() == sim.ScenarioModeTower
}

func (sg *scenarioGroup) isERAM() bool {
	return sg.scenarioMode() == sim.ScenarioModeERAM
}

func (sg *scenarioGroup) isSTARS() bool {
	return sg.scenarioMode() == sim.ScenarioModeSTARS
}

// validateFacilitySelector verifies that exactly one facility selector is
// present, then delegates mode-specific validation. It must never rewrite
// ARTCC, TRACON, or Tower; those fields describe the scenario group's source
// identity and are used later to select the correct controller environment.
func (sg *scenarioGroup) validateFacilitySelector(e *util.ErrorLogger) {
	selectors := 0
	if sg.Tower != "" {
		selectors++
	}
	if sg.TRACON != "" {
		selectors++
	}
	if sg.ARTCC != "" {
		selectors++
	}

	if selectors == 0 {
		e.ErrorString(`"tracon", "artcc", or "tower" must be specified`)
		return
	}
	if selectors > 1 {
		e.ErrorString(`only one of "tracon", "artcc", or "tower" may be specified`)
		return
	}

	switch sg.scenarioMode() {
	case sim.ScenarioModeTower:
		sg.validateTowerFacility(e)
	case sim.ScenarioModeERAM:
		sg.validateERAMFacility(e)
	case sim.ScenarioModeSTARS:
		sg.validateSTARSFacility(e)
	}
}
