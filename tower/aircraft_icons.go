// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package tower

import (
	"bytes"
	"embed"
	"image"
	"io/fs"
	"path"
	"strings"

	"github.com/mmp/vice/renderer"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

const aircraftIconTextureSize = 128

// aircraftIconAliases maps ICAO variants used by VICE or flight-plan feeds to
// the closest SVG filename. Exact SVG matches always take priority, so these
// aliases are only used when the icon set has no file for the reported type.
var aircraftIconAliases = map[string][]string{
	"E75L": {"E175"},
	"E75S": {"E175"},

	// New-engine variants can use the corresponding base-airframe silhouette
	// when the icon set does not provide a dedicated SVG.
	"A20N": {"A320"},
	"A21N": {"A321"},
	"B38M": {"B738"},
	"B39M": {"B739"},
}

//go:embed icons/*.svg
var aircraftIconFS embed.FS

func loadAircraftTextures(r renderer.Renderer) map[string]uint32 {
	textures := make(map[string]uint32)

	entries, err := fs.ReadDir(aircraftIconFS, "icons")
	if err != nil {
		return textures
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(path.Ext(entry.Name()), ".svg") {
			continue
		}

		data, err := aircraftIconFS.ReadFile(path.Join("icons", entry.Name()))
		if err != nil {
			continue
		}

		texture := rasterizeAircraftSVG(r, data)
		if texture == 0 {
			continue
		}

		icao := strings.ToUpper(strings.TrimSuffix(entry.Name(), path.Ext(entry.Name())))
		textures[icao] = texture
	}

	return textures
}

func rasterizeAircraftSVG(r renderer.Renderer, data []byte) uint32 {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(data), oksvg.WarnErrorMode)
	if err != nil {
		return 0
	}

	size := aircraftIconTextureSize
	width := float64(size)
	height := float64(size)
	x := float64(0)
	y := float64(0)

	if icon.ViewBox.W > 0 && icon.ViewBox.H > 0 {
		aspect := icon.ViewBox.W / icon.ViewBox.H
		if aspect > 1 {
			height = width / aspect
			y = (float64(size) - height) / 2
		} else {
			width = height * aspect
			x = (float64(size) - width) / 2
		}
	}

	icon.SetTarget(x, y, width, height)

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	raster := rasterx.NewDasher(size, size, scanner)
	icon.Draw(raster, 1)

	// Turn the SVG into a white alpha mask. The textured-quad builder then
	// applies Tower Cab's configured aircraft, outline, and shadow colors.
	for i := 0; i+3 < len(img.Pix); i += 4 {
		alpha := img.Pix[i+3]
		img.Pix[i] = alpha
		img.Pix[i+1] = alpha
		img.Pix[i+2] = alpha
	}

	return r.CreateTextureFromImage(img, false)
}

func destroyAircraftTextures(r renderer.Renderer, textures map[string]uint32) {
	if r == nil {
		return
	}
	for _, texture := range textures {
		if texture != 0 {
			r.DestroyTexture(texture)
		}
	}
}

func normalizeAircraftIconType(aircraftType string) string {
	aircraftType = strings.ToUpper(strings.TrimSpace(aircraftType))
	if aircraftType == "" {
		return ""
	}

	// Equipment and wake-turbulence suffixes may follow the ICAO designator,
	// for example B738/L or A320-S. Keep only the designator.
	if index := strings.IndexAny(aircraftType, "/- \t"); index >= 0 {
		aircraftType = aircraftType[:index]
	}

	return strings.Trim(aircraftType, "[](){}")
}

func aircraftTextureForTrack(textures map[string]uint32, aircraftType string) uint32 {
	icao := normalizeAircraftIconType(aircraftType)

	// Prefer a dedicated SVG whenever one exists.
	if texture := textures[icao]; texture != 0 {
		return texture
	}

	// Otherwise try known equivalent or closely related silhouettes. Each
	// alias is checked against the textures actually loaded from icons/*.svg.
	for _, alias := range aircraftIconAliases[icao] {
		if texture := textures[alias]; texture != 0 {
			return texture
		}
	}

	// Final fallback: use the B738 silhouette for unknown or missing types.
	return textures["B738"]
}
