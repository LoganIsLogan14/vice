// pkg/server/eram.go
// Copyright(c) 2022-2024 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package server

import (
	av "github.com/mmp/vice/aviation"
	"github.com/mmp/vice/util"
)

// validateERAMFacility validates the ARTCC selected by an ERAM scenario
// group. The ARTCC remains in sg.ARTCC; it must not be copied into sg.TRACON,
// because doing so causes the scenario to be classified and displayed as
// STARS later in the loading process.
func (sg *scenarioGroup) validateERAMFacility(e *util.ErrorLogger) {
	if _, ok := av.DB.ARTCCs[sg.ARTCC]; !ok {
		e.ErrorString("ARTCC %q is unknown; it must be a 3-letter identifier listed at "+
			"https://www.faa.gov/about/office_org/headquarters_offices/ato/service_units/air_traffic_services/artcc",
			sg.ARTCC)
	}
}
