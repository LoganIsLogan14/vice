// sim/tower.go
// Copyright(c) 2022-2026 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package sim

// assignTowerOwnership gives a spawning flight to the cab's controlling
// position.
//
// A tower cab is a single-position environment. There are no fix pairs to
// derive an owner from, no pseudo-ERAM coordination with an adjacent
// centre, and no auto-scratchpad adaptation, so the position at the root
// of the scenario's consolidation tree owns every aircraft outright, from
// spawn until it is culled.
//
// This replaces the STARS/ERAM ownership pipeline rather than running
// after it: deriveERAMFixPair and applyFixPairAssignment exist to move
// ownership between sectors, which a cab has none of.
//
// When cab positions are modelled separately — ground, local, clearance
// delivery — this is where the handoff between them will be decided, since
// it is the single point where a new flight is given its first owner.
func (s *Sim) assignTowerOwnership(nasFp *NASFlightPlan, ac *Aircraft) {
	pos := s.scenarioRootPosition()

	nasFp.TrackingController = pos
	nasFp.OwningTCW = s.tcwForPosition(pos)
	nasFp.InboundHandoffController = pos

	// The flight talks to the cab, not to whichever centre sector the
	// inbound flow or exit route nominated.
	ac.ControllerFrequency = pos
}
