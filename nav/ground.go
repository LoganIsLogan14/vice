// nav/ground.go
// Copyright(c) 2022-2026 vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package nav

const (
	// touchdownAGL is how far above the arrival field an aircraft flying an
	// approach is taken to have touched down. The approach already flies the
	// aircraft down to the runway; this only decides when it stops being a
	// flight and starts being a ground vehicle.
	touchdownAGL = 50

	// rolloutSpeed is the speed a landed aircraft decelerates to. It stands
	// in for turning off the runway, which needs taxiway geometry that does
	// not exist yet.
	rolloutSpeed = 15
)

// checkTouchdown transitions an aircraft flying an approach to the landed
// state once it reaches the runway.
//
// Without this an arrival simply flies through the field at approach speed
// and is eventually removed by distance culling, which is invisible from a
// tower cab and leaves the runway looking permanently clear. Landing is
// gated on GroundOps because STARS and ERAM scenarios rely on arrivals
// continuing to fly until a controller deletes them.
func (nav *Nav) checkTouchdown() {
	if !nav.GroundOps || nav.FlightState.Landed {
		return
	}
	// Only an aircraft actually flying an approach can land; one being
	// vectored through the area at low altitude has not.
	if !nav.OnApproach(false) {
		return
	}
	if nav.FlightState.Altitude > nav.FlightState.ArrivalAirportElevation+touchdownAGL {
		return
	}

	nav.FlightState.Landed = true

	// Drop any assigned altitude or speed: the rollout is flown by
	// TargetAltitude and TargetSpeed from here on.
	nav.Altitude = NavAltitude{}
	nav.Speed = NavSpeed{}
}

// IsLanded reports whether the aircraft has touched down and is on the
// runway.
func (nav *Nav) IsLanded() bool { return nav.FlightState.Landed }
