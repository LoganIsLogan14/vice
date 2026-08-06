// sim/tower.go
// Copyright(c) 2022-2026 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package sim

import (
	"github.com/mmp/vice/math"
)

// towerApproachClearanceNM is how far from the field a tower cab's
// arrivals are cleared for the approach they were told to expect.
//
// In a full facility the TRACON issues this clearance. A tower cab has no
// TRACON position working the arrivals, so without it they would fly their
// STAR indefinitely, never become established, and never touch down.
// Clearing on the way in rather than at spawn means they still fly the
// arrival procedure first.
const towerApproachClearanceNM = 15

// maybeClearTowerArrival stands in for the TRACON, clearing an inbound for
// its expected approach once it is close enough to the field.
func (s *Sim) maybeClearTowerArrival(ac *Aircraft) {
	if !s.State.IsTower() || ac.Nav.IsLanded() {
		return
	}
	// Nothing to clear them for unless the inbound flow adapted an
	// "expect_approach", and never re-clear one already established.
	if ac.Nav.Approach.Cleared || ac.Nav.Approach.Assigned == nil {
		return
	}
	if math.NMDistance2LL(ac.Position(), ac.Nav.FlightState.ArrivalAirportLocation) >
		towerApproachClearanceNM {
		return
	}

	ac.Nav.ClearedApproach("", nil, s.State.SimTime.NavTime(), false)
}

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

	// A cab is looking at the runway, so arrivals have to actually land on
	// it rather than flying through the field and being culled 200 miles
	// later.
	ac.Nav.GroundOps = true
}
