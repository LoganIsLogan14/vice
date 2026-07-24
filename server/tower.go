// pkg/server/tower.go
// Copyright(c) 2022-2024 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package server

import (
	av "github.com/mmp/vice/aviation"
	"github.com/mmp/vice/util"
)

// validateTowerFacility validates the ATCT selected by a Tower scenario group.
func (sg *scenarioGroup) validateTowerFacility(e *util.ErrorLogger) {
	if !av.DB.IsATCT(sg.Tower) {
		e.ErrorString("Tower %q is unknown; it must be a known ATCT identifier", sg.Tower)
	}
}
