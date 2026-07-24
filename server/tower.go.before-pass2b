// pkg/server/tower.go
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
	if sg.Tower != "" {
		return sg.Tower + "_TOWER"
	}
	if sg.TRACON != "" {
		return sg.TRACON
	}
	return sg.ARTCC
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

func (sg *scenarioGroup) isTower() bool { return sg.scenarioMode() == sim.ScenarioModeTower }
func (sg *scenarioGroup) isERAM() bool  { return sg.scenarioMode() == sim.ScenarioModeERAM }
func (sg *scenarioGroup) isSTARS() bool { return sg.scenarioMode() == sim.ScenarioModeSTARS }

// validateFacilitySelector validates only the mutually exclusive facility
// selector at the top of a scenario-group file. Keeping this separate from
// the rest of PostDeserialize is the first step toward mode-specific loaders
// and validation without changing current simulation behavior.
func (sg *scenarioGroup) validateFacilitySelector(e *util.ErrorLogger) {
	switch {
	case sg.Tower != "":
		if sg.TRACON != "" || sg.ARTCC != "" {
			e.ErrorString(`"tower" cannot be combined with "tracon" or "artcc"`)
		} else if !av.DB.IsATCT(sg.Tower) {
			e.ErrorString("Tower %q is unknown; it must be a known ATCT identifier", sg.Tower)
		}

	case sg.ARTCC == "":
		if sg.TRACON == "" {
			e.ErrorString(`"tracon", "artcc", or "tower" must be specified`)
		} else if !av.DB.IsFacility(sg.TRACON) {
			e.ErrorString("TRACON %q is unknown; it must be a 3-letter identifier listed at "+
				"https://www.faa.gov/about/office_org/headquarters_offices/ato/service_units/air_traffic_services/tracon.",
				sg.TRACON)
		}

	case sg.TRACON == "":
		if _, ok := av.DB.ARTCCs[sg.ARTCC]; !ok {
			e.ErrorString("ARTCC %q is unknown; it must be a 3-letter identifier listed at "+
				"https://www.faa.gov/about/office_org/headquarters_offices/ato/service_units/air_traffic_services/artcc", sg.ARTCC)
		}
		// Preserve existing behavior: ERAM scenario groups use the ARTCC as
		// their internal TRACON/facility identifier in downstream code.
		sg.TRACON = sg.ARTCC
	}
}
