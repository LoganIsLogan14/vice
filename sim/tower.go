// sim/tower.go
// Copyright(c) 2022-2026 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package sim

import (
	av "github.com/mmp/vice/aviation"
	"github.com/mmp/vice/nav"
)

const (
	// towerFinalNM is how far out a tower cab's arrivals appear, already
	// established on the final approach course.
	towerFinalNM = 15

	// towerGlidepathFtPerNM is a nominal 3 degree glidepath, used to place
	// an arrival at a plausible altitude for its distance from the runway.
	towerGlidepathFtPerNM = 318

	towerFinalIAS     = 180
	towerThresholdIAS = 140

	// towerRolloutNM is how far down the runway an arrival rolls before it
	// is considered clear of it.
	towerRolloutNM = 1.5
)

// towerArrivalRunway returns the runway arrivals are landing on at the
// given airport under the current configuration.
func (s *Sim) towerArrivalRunway(airport string) (av.Runway, bool) {
	// Spread arrivals over every configured runway rather than sending them
	// all to whichever happens to be listed first; a parallel operation
	// otherwise uses half the airport.
	var ids []av.RunwayID
	for _, ar := range s.State.ArrivalRunways {
		if ar.Airport == airport {
			ids = append(ids, ar.Runway)
		}
	}
	if len(ids) == 0 {
		return av.Runway{}, false
	}
	id := ids[s.Rand.Intn(len(ids))]

	faaAP, ok := av.DB.Airports[airport]
	if !ok {
		return av.Runway{}, false
	}
	for _, rwy := range faaAP.Runways {
		if rwy.Id == string(id) {
			return rwy, true
		}
	}
	return av.Runway{}, false
}

// placeTowerArrivalOnFinal discards an arrival's enroute and terminal
// routing and puts it on final for the landing runway instead.
//
// A tower cab has no interest in the STAR: the whole segment from the
// arrival fix to the final approach course belongs to positions the cab
// does not work, and at facilities where the TRACON vectors to final there
// is no adapted approach for the aircraft to fly itself down anyway, so
// arrivals would circle the STAR forever and never land.
//
// Returns false when the scenario gives no usable landing runway, in which
// case the aircraft keeps its original routing.
func (s *Sim) placeTowerArrivalOnFinal(ac *Aircraft) bool {
	airport := ac.FlightPlan.ArrivalAirport
	rwy, ok := s.towerArrivalRunway(airport)
	if !ok {
		s.lg.Warnf("tower: no usable arrival runway for %s (configured: %v); "+
			"%s keeps its original routing", airport, s.State.ArrivalRunways, ac.ADSBCallsign)
		return false
	}
	faaAP, ok := av.DB.Airports[airport]
	if !ok {
		s.lg.Warnf("tower: %s is not in the airport database; %s keeps its original routing",
			airport, ac.ADSBCallsign)
		return false
	}

	b := newPatternBuilder(rwy, faaAP.Elevation, s.State.NmPerLongitude, s.State.MagneticVariation)

	// The route is just the runway now: cross the threshold, then roll out
	// along it. Reaching the end of the rollout stands in for clearing the
	// runway onto a taxiway, which needs surface geometry that does not
	// exist yet; without it landed aircraft accumulate on the runway
	// forever, since nothing else removes them.
	rollout := b.waypoint("_twr_rollout", towerRolloutNM, 0, 0, 0, 0)
	rollout.SetDelete(true)

	ac.Nav.Waypoints = []av.Waypoint{
		b.waypoint("_twr_threshold", 0, 0, 0, towerThresholdIAS, 0),
		rollout,
	}

	// Drop the STAR's altitude and speed restrictions along with any
	// approach the inbound flow told the aircraft to expect; none of it
	// applies to an aircraft that is already on final.
	ac.Nav.Altitude = nav.NavAltitude{}
	ac.Nav.Speed = nav.NavSpeed{}
	ac.Nav.Approach = nav.NavApproach{}

	start := b.waypoint("_twr_final", -towerFinalNM, 0, 0, 0, 0)
	ac.Nav.FlightState.Position = start.Location
	ac.Nav.FlightState.Altitude = float32(faaAP.Elevation) + towerFinalNM*towerGlidepathFtPerNM
	ac.Nav.FlightState.Heading = rwy.Heading
	ac.Nav.FlightState.IAS = towerFinalIAS

	s.lg.Debugf("tower: placed %s on a %.0fnm final for %s runway %s at %.0f ft",
		ac.ADSBCallsign, float32(towerFinalNM), airport, rwy.Id,
		ac.Nav.FlightState.Altitude)

	return true
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
