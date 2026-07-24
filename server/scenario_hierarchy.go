// Copyright(c) 2022-2026 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX-License-Identifier: GPL-3.0-only

package server

import (
	"slices"
	"strings"

	"github.com/mmp/vice/sim"
)

// ScenarioHierarchy is the launcher-facing representation of the available
// scenarios. It deliberately sits alongside the legacy catalog maps so saved
// simulations and RPC callers remain compatible while the launcher can present
// ARTCC -> Facility -> Position -> Scenario.
type ScenarioHierarchy struct {
	ARTCCs map[string]*ScenarioARTCC
}

type ScenarioARTCC struct {
	Facilities map[string]*ScenarioFacility
}

type ScenarioFacility struct {
	Positions map[string]*ScenarioPosition
}

type ScenarioPosition struct {
	Scenarios []ScenarioSelection
}

// ScenarioSelection retains the legacy catalog keys needed by SetScenario.
type ScenarioSelection struct {
	BackendFacility string
	GroupName       string
	ScenarioName    string
	Spec            *ScenarioSpec
}

// BuildScenarioHierarchy derives the new launcher hierarchy from existing
// scenario catalogs. No JSON migration is required: legacy STARS and ERAM
// scenarios are inferred, while Tower scenarios are grouped beneath their
// primary airport (for example ZMP -> MSP -> Tower).
func BuildScenarioHierarchy(catalogs map[string]map[string]*ScenarioCatalog) ScenarioHierarchy {
	h := ScenarioHierarchy{ARTCCs: make(map[string]*ScenarioARTCC)}

	for backendFacility, groups := range catalogs {
		for groupName, catalog := range groups {
			artcc := catalog.ARTCC
			if artcc == "" {
				artcc = backendFacility
			}

			for scenarioName, spec := range catalog.Scenarios {
				facility, position := scenarioHierarchyLabels(backendFacility, catalog, spec)

				a := h.ARTCCs[artcc]
				if a == nil {
					a = &ScenarioARTCC{Facilities: make(map[string]*ScenarioFacility)}
					h.ARTCCs[artcc] = a
				}
				f := a.Facilities[facility]
				if f == nil {
					f = &ScenarioFacility{Positions: make(map[string]*ScenarioPosition)}
					a.Facilities[facility] = f
				}
				p := f.Positions[position]
				if p == nil {
					p = &ScenarioPosition{}
					f.Positions[position] = p
				}
				p.Scenarios = append(p.Scenarios, ScenarioSelection{
					BackendFacility: backendFacility,
					GroupName:       groupName,
					ScenarioName:    scenarioName,
					Spec:            spec,
				})
			}
		}
	}

	for _, artcc := range h.ARTCCs {
		for _, facility := range artcc.Facilities {
			for _, position := range facility.Positions {
				slices.SortFunc(position.Scenarios, func(a, b ScenarioSelection) int {
					if a.ScenarioName != b.ScenarioName {
						return strings.Compare(a.ScenarioName, b.ScenarioName)
					}
					if a.BackendFacility != b.BackendFacility {
						return strings.Compare(a.BackendFacility, b.BackendFacility)
					}
					return strings.Compare(a.GroupName, b.GroupName)
				})
			}
		}
	}

	return h
}

func scenarioHierarchyLabels(backendFacility string, catalog *ScenarioCatalog, spec *ScenarioSpec) (facility, position string) {
	switch spec.Mode {
	case sim.ScenarioModeTower:
		facility = airportFacilityCode(spec.PrimaryAirport)
		if facility == "" {
			facility = backendFacility
		}
		position = "Tower"
	case sim.ScenarioModeERAM:
		facility = backendFacility
		position = "Center"
	default:
		facility = backendFacility
		position = "STARS"
	}
	return facility, position
}

func airportFacilityCode(icao string) string {
	icao = strings.ToUpper(strings.TrimSpace(icao))
	if len(icao) == 4 && icao[0] == 'K' {
		return icao[1:]
	}
	return icao
}
