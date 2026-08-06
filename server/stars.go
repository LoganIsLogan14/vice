// pkg/server/stars.go
// Copyright(c) 2022-2024 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package server

import (
	av "github.com/mmp/vice/aviation"
	"github.com/mmp/vice/util"
)

// validateSTARSFacility validates the TRACON selected by a STARS scenario
// group. No facility fields are synthesized or rewritten here.
func (sg *scenarioGroup) validateSTARSFacility(e *util.ErrorLogger) {
	if !av.DB.IsFacility(sg.TRACON) {
		e.ErrorString("TRACON %q is unknown; it must be a 3-letter identifier listed at "+
			"https://www.faa.gov/about/office_org/headquarters_offices/ato/service_units/air_traffic_services/tracon.",
			sg.TRACON)
	}
}
