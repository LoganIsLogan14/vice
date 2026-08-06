// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package tower

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CRCLayoutStats summarizes a CRC visual map without exposing any renderer-
// specific representation. Phase 1 intentionally only discovers and parses
// CRC data; it does not change Tower Cab rendering.
type CRCLayoutStats struct {
	Name          string
	ARTCC         string
	GeoJSONPath   string
	FeatureCount  int
	LineCount     int
	PolygonCount  int
	LineVertices  int
	PolygonPoints int
	Colors        map[string]int
	ZIndexes      map[int]int
}

type crcARTCCIndex struct {
	VideoMaps []crcVideoMapMetadata `json:"videoMaps"`
}

type crcVideoMapMetadata struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Tags           []string `json:"tags"`
	SourceFileName string   `json:"sourceFileName"`
}

type crcFeatureCollection struct {
	Type     string       `json:"type"`
	Features []crcFeature `json:"features"`
}

type crcFeature struct {
	Type       string        `json:"type"`
	Geometry   crcGeometry   `json:"geometry"`
	Properties crcProperties `json:"properties"`
}

type crcGeometry struct {
	Type        string          `json:"type"`
	Coordinates json.RawMessage `json:"coordinates"`
}

type crcProperties struct {
	Color     string   `json:"color"`
	Thickness *float64 `json:"thickness"`
	ZIndex    int      `json:"zIndex"`
}

type crcMapMatch struct {
	ARTCC string
	Map   crcVideoMapMetadata
	Rank  int
}

// InspectCRCLayoutForAirport locates a CRC Cab or ASDEX map for an airport and
// returns parse statistics. It prefers "<IATA> Cab" over "<IATA> ASDEX".
//
// CRC data is normally read from ~/.vatsim-crc. Set VICE_CRC_DATA_DIR to use a
// different CRC data directory.
func InspectCRCLayoutForAirport(airportID string) (CRCLayoutStats, error) {
	root, err := crcDataDirectory()
	if err != nil {
		return CRCLayoutStats{}, err
	}

	match, err := findCRCMap(root, airportID)
	if err != nil {
		return CRCLayoutStats{}, err
	}

	path := filepath.Join(root, "VideoMaps", match.ARTCC, match.Map.ID+".geojson")
	stats, err := inspectCRCGeoJSON(path)
	if err != nil {
		return CRCLayoutStats{}, fmt.Errorf("inspect %q: %w", match.Map.Name, err)
	}

	stats.Name = match.Map.Name
	stats.ARTCC = match.ARTCC
	stats.GeoJSONPath = path
	return stats, nil
}

func crcDataDirectory() (string, error) {
	if configured := strings.TrimSpace(os.Getenv("VICE_CRC_DATA_DIR")); configured != "" {
		return configured, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".vatsim-crc"), nil
}

func findCRCMap(root, airportID string) (crcMapMatch, error) {
	code := normalizeCRCAirportID(airportID)
	if code == "" {
		return crcMapMatch{}, errors.New("airport ID is empty")
	}

	wanted := []string{
		code + " Cab",
		code + " ASDEX",
	}

	artccDir := filepath.Join(root, "ARTCCs")
	entries, err := os.ReadDir(artccDir)
	if err != nil {
		return crcMapMatch{}, fmt.Errorf("read %s: %w", artccDir, err)
	}

	var matches []crcMapMatch
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".json") {
			continue
		}

		path := filepath.Join(artccDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		var index crcARTCCIndex
		if err := json.Unmarshal(data, &index); err != nil {
			continue
		}

		artcc := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		for _, videoMap := range index.VideoMaps {
			for rank, desiredName := range wanted {
				if strings.EqualFold(strings.TrimSpace(videoMap.Name), desiredName) {
					matches = append(matches, crcMapMatch{
						ARTCC: artcc,
						Map:   videoMap,
						Rank:  rank,
					})
				}
			}
		}
	}

	if len(matches) == 0 {
		return crcMapMatch{}, fmt.Errorf(
			"no CRC map named %q or %q was found under %s",
			wanted[0], wanted[1], artccDir,
		)
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].Rank != matches[j].Rank {
			return matches[i].Rank < matches[j].Rank
		}
		if matches[i].ARTCC != matches[j].ARTCC {
			return matches[i].ARTCC < matches[j].ARTCC
		}
		return matches[i].Map.Name < matches[j].Map.Name
	})

	return matches[0], nil
}

func normalizeCRCAirportID(airportID string) string {
	code := strings.ToUpper(strings.TrimSpace(airportID))

	// CRC's U.S. map names generally use the three-letter FAA/IATA identifier,
	// while VICE scenarios normally use the four-letter ICAO identifier.
	if len(code) == 4 && code[0] == 'K' {
		code = code[1:]
	}
	return code
}

func inspectCRCGeoJSON(path string) (CRCLayoutStats, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return CRCLayoutStats{}, fmt.Errorf("read GeoJSON: %w", err)
	}

	var collection crcFeatureCollection
	if err := json.Unmarshal(data, &collection); err != nil {
		return CRCLayoutStats{}, fmt.Errorf("decode GeoJSON: %w", err)
	}
	if collection.Type != "" && collection.Type != "FeatureCollection" {
		return CRCLayoutStats{}, fmt.Errorf(
			"unsupported GeoJSON root type %q", collection.Type,
		)
	}

	stats := CRCLayoutStats{
		FeatureCount: len(collection.Features),
		Colors:       make(map[string]int),
		ZIndexes:     make(map[int]int),
	}

	for index, feature := range collection.Features {
		if feature.Properties.Color != "" {
			stats.Colors[feature.Properties.Color]++
		}
		stats.ZIndexes[feature.Properties.ZIndex]++

		switch feature.Geometry.Type {
		case "LineString":
			var coordinates [][]json.Number
			if err := json.Unmarshal(feature.Geometry.Coordinates, &coordinates); err != nil {
				return CRCLayoutStats{}, fmt.Errorf(
					"feature %d LineString coordinates: %w", index, err,
				)
			}
			stats.LineCount++
			stats.LineVertices += len(coordinates)

		case "Polygon":
			var coordinates [][][]json.Number
			if err := json.Unmarshal(feature.Geometry.Coordinates, &coordinates); err != nil {
				return CRCLayoutStats{}, fmt.Errorf(
					"feature %d Polygon coordinates: %w", index, err,
				)
			}
			stats.PolygonCount++
			for _, ring := range coordinates {
				stats.PolygonPoints += len(ring)
			}

		default:
			// Unknown geometry is tolerated in phase 1 so one new CRC feature
			// type does not prevent us from inspecting the rest of the map.
		}
	}

	return stats, nil
}
