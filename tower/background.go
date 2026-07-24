// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package tower

import (
	"github.com/mmp/vice/radar"
	"github.com/mmp/vice/renderer"
)

// AirportBackgroundSource draws an optional georeferenced layer beneath the
// airport surface geometry.
type AirportBackgroundSource interface {
	Name() string
	Draw(rangeNM float32, transforms radar.ScopeTransformations, cb *renderer.CommandBuffer)
	Dispose()
}

func backgroundSourceName(source AirportBackgroundSource) string {
	if source == nil {
		return "NONE"
	}
	return source.Name()
}
