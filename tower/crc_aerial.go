// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package tower

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	_ "image/jpeg"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mmp/vice/math"
	"github.com/mmp/vice/radar"
	"github.com/mmp/vice/renderer"
)

type crcAerialBounds struct {
	south float32
	west  float32
	north float32
	east  float32
}

// lowResLatitudeCorrection shifts CRC LowRes imagery north to match the
// HighRes layer. The value was tuned against MSP imagery.
//
// 0.0002222 degrees is approximately 24.7 meters north.
const lowResLatitudeCorrection = float32(0.0002222)

// CRCAerialBackgroundSource loads CRC's airport-specific, self-georeferenced
// TowerCab JPEGs. The GPS and GPSDest EXIF tags encode the southwest and
// northeast corners respectively.
type crcAerialLayer struct {
	path    string
	texture uint32
	bounds  crcAerialBounds
	loaded  bool
}

type CRCAerialBackgroundSource struct {
	renderer renderer.Renderer
	name     string
	config   *TowerCabConfig

	low  crcAerialLayer
	high crcAerialLayer
}

func NewCRCAerialBackgroundSource(
	r renderer.Renderer,
	airportID string,
	config *TowerCabConfig,
) (*CRCAerialBackgroundSource, error) {
	root, err := crcDataDirectory()
	if err != nil {
		return nil, err
	}

	match, err := findCRCMap(root, airportID)
	if err != nil {
		return nil, err
	}

	imageID := strings.ToUpper(strings.TrimSpace(airportID))
	if len(imageID) == 4 && imageID[0] == 'K' {
		imageID = imageID[1:]
	}

	imageDirectory := filepath.Join(root, "TowerCabImages", match.ARTCC)
	lowPath := filepath.Join(imageDirectory, imageID+"-LowRes.jpg")
	highPath := filepath.Join(imageDirectory, imageID+"-HighRes.jpg")

	if _, err := os.Stat(lowPath); err != nil {
		lowPath = ""
	}
	if _, err := os.Stat(highPath); err != nil {
		highPath = ""
	}
	if lowPath == "" && highPath == "" {
		return nil, fmt.Errorf("no CRC TowerCab imagery found for %s", airportID)
	}

	return &CRCAerialBackgroundSource{
		renderer: r,
		name:     "CRC AERIAL",
		config:   config,
		low:      crcAerialLayer{path: lowPath},
		high:     crcAerialLayer{path: highPath},
	}, nil
}

func (s *CRCAerialBackgroundSource) Name() string {
	return s.name
}

func (s *CRCAerialBackgroundSource) Dispose() {
	s.disposeLayer(&s.low)
	s.disposeLayer(&s.high)
}

func (s *CRCAerialBackgroundSource) disposeLayer(layer *crcAerialLayer) {
	if layer.texture != 0 {
		s.renderer.DestroyTexture(layer.texture)
	}
	layer.texture = 0
	layer.loaded = false
}

func (s *CRCAerialBackgroundSource) Draw(
	rangeNM float32,
	transforms radar.ScopeTransformations,
	cb *renderer.CommandBuffer,
) {
	// CRC keeps its broad low-resolution image resident and draws the tighter
	// high-resolution image over it at close ranges. This avoids destroying
	// and recreating textures whenever the zoom threshold is crossed.
	if s.low.path != "" && !s.low.loaded {
		if err := s.loadLayer(&s.low); err != nil {
			return
		}
	}

	// Airports that only have a HighRes image still remain usable.
	if s.low.texture == 0 && s.high.path != "" && !s.high.loaded {
		if err := s.loadLayer(&s.high); err != nil {
			return
		}
	}

	if s.low.texture != 0 {
		s.drawLayer(&s.low, transforms, cb, 1, false)
	}

	if s.config != nil && !s.config.UseHighRes {
		return
	}

	highAlpha := s.highResAlpha(rangeNM)
	if highAlpha <= 0 || s.high.path == "" {
		return
	}

	// Lazy-load HighRes only when it is first needed, then retain it for the
	// rest of the airport session.
	if !s.high.loaded {
		if err := s.loadLayer(&s.high); err != nil {
			return
		}
	}
	if s.high.texture != 0 {
		s.drawLayer(&s.high, transforms, cb, highAlpha, true)
	}
}

func (s *CRCAerialBackgroundSource) highResAlpha(rangeNM float32) float32 {
	startRange := float32(9)
	fullRange := float32(6)
	if s.config != nil {
		startRange = s.config.HighResStartNM
		fullRange = s.config.HighResFullNM
	}

	if rangeNM >= startRange {
		return 0
	}
	if rangeNM <= fullRange {
		return 1
	}
	return (startRange - rangeNM) / (startRange - fullRange)
}

func (s *CRCAerialBackgroundSource) drawLayer(
	layer *crcAerialLayer,
	transforms radar.ScopeTransformations,
	cb *renderer.CommandBuffer,
	alpha float32,
	blend bool,
) {
	bounds := layer.bounds
	if layer == &s.low {
		bounds.south += lowResLatitudeCorrection
		bounds.north += lowResLatitudeCorrection
	}

	sw := transforms.WindowFromLatLongP(math.Point2LL{bounds.west, bounds.south})
	se := transforms.WindowFromLatLongP(math.Point2LL{bounds.east, bounds.south})
	ne := transforms.WindowFromLatLongP(math.Point2LL{bounds.east, bounds.north})
	nw := transforms.WindowFromLatLongP(math.Point2LL{bounds.west, bounds.north})

	vertices := [][2]float32{nw, ne, se, sw}
	uv := [][2]float32{
		{0, 0},
		{1, 0},
		{1, 1},
		{0, 1},
	}
	indices := []int32{0, 1, 2, 0, 2, 3}

	vertexBuffer := cb.Float2Buffer(vertices)
	uvBuffer := cb.Float2Buffer(uv)
	indexBuffer := cb.IntBuffer(indices)

	if blend {
		cb.Blend()
	}
	cb.EnableTexture(layer.texture)
	brightness := float32(0.52)
	if s.config != nil {
		brightness = s.config.AerialBrightness
	}
	cb.SetRGBA(renderer.RGBA{
		R: brightness,
		G: brightness,
		B: brightness,
		A: alpha,
	})
	cb.VertexArray(vertexBuffer, 2, 2*4)
	cb.TexCoordArray(uvBuffer, 2, 2*4)
	cb.DrawTriangles(indexBuffer, len(indices))
	cb.DisableTexCoordArray()
	cb.DisableVertexArray()
	cb.DisableTexture()
	if blend {
		cb.DisableBlend()
	}
	cb.SetRGB(renderer.RGB{R: 1, G: 1, B: 1})
}

func (s *CRCAerialBackgroundSource) loadLayer(layer *crcAerialLayer) error {
	if layer.path == "" {
		return fmt.Errorf("CRC aerial layer path is unavailable")
	}

	file, err := os.Open(layer.path)
	if err != nil {
		return fmt.Errorf("open CRC aerial image: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("read CRC aerial image: %w", err)
	}

	bounds, err := readCRCAerialBounds(data)
	if err != nil {
		return fmt.Errorf("read CRC aerial EXIF bounds: %w", err)
	}

	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("decode CRC aerial image: %w", err)
	}

	texture := s.renderer.CreateTextureFromImage(img, false)
	if texture == 0 {
		return fmt.Errorf("create CRC aerial texture")
	}

	layer.texture = texture
	layer.bounds = bounds
	layer.loaded = true
	return nil
}

func readCRCAerialBounds(jpeg []byte) (crcAerialBounds, error) {
	tiff, err := findEXIFTIFF(jpeg)
	if err != nil {
		return crcAerialBounds{}, err
	}
	if len(tiff) < 8 {
		return crcAerialBounds{}, fmt.Errorf("truncated TIFF header")
	}

	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return crcAerialBounds{}, fmt.Errorf("unknown TIFF byte order")
	}
	if order.Uint16(tiff[2:4]) != 42 {
		return crcAerialBounds{}, fmt.Errorf("invalid TIFF marker")
	}

	ifd0 := int(order.Uint32(tiff[4:8]))
	gpsOffset, err := findIFDLongTag(tiff, order, ifd0, 0x8825)
	if err != nil {
		return crcAerialBounds{}, fmt.Errorf("GPS IFD: %w", err)
	}

	gps, err := readGPSIFD(tiff, order, int(gpsOffset))
	if err != nil {
		return crcAerialBounds{}, err
	}

	south, err := gps.coordinate(2, 1)
	if err != nil {
		return crcAerialBounds{}, fmt.Errorf("southwest latitude: %w", err)
	}
	west, err := gps.coordinate(4, 3)
	if err != nil {
		return crcAerialBounds{}, fmt.Errorf("southwest longitude: %w", err)
	}
	north, err := gps.coordinate(20, 19)
	if err != nil {
		return crcAerialBounds{}, fmt.Errorf("northeast latitude: %w", err)
	}
	east, err := gps.coordinate(22, 21)
	if err != nil {
		return crcAerialBounds{}, fmt.Errorf("northeast longitude: %w", err)
	}

	if north <= south || east <= west {
		return crcAerialBounds{}, fmt.Errorf(
			"invalid geographic bounds south=%f west=%f north=%f east=%f",
			south, west, north, east,
		)
	}

	return crcAerialBounds{
		south: float32(south),
		west:  float32(west),
		north: float32(north),
		east:  float32(east),
	}, nil
}

func findEXIFTIFF(jpeg []byte) ([]byte, error) {
	if len(jpeg) < 4 || jpeg[0] != 0xff || jpeg[1] != 0xd8 {
		return nil, fmt.Errorf("not a JPEG")
	}

	for offset := 2; offset+4 <= len(jpeg); {
		if jpeg[offset] != 0xff {
			offset++
			continue
		}
		marker := jpeg[offset+1]
		offset += 2

		if marker == 0xd9 || marker == 0xda {
			break
		}
		if marker == 0x00 || marker == 0xd8 || (marker >= 0xd0 && marker <= 0xd7) {
			continue
		}
		if offset+2 > len(jpeg) {
			break
		}

		length := int(binary.BigEndian.Uint16(jpeg[offset : offset+2]))
		if length < 2 || offset+length > len(jpeg) {
			return nil, fmt.Errorf("invalid JPEG segment")
		}
		payload := jpeg[offset+2 : offset+length]
		if marker == 0xe1 && len(payload) >= 6 &&
			string(payload[:6]) == "Exif\x00\x00" {
			return payload[6:], nil
		}
		offset += length
	}

	return nil, fmt.Errorf("EXIF APP1 segment not found")
}

func findIFDLongTag(
	tiff []byte,
	order binary.ByteOrder,
	offset int,
	tag uint16,
) (uint32, error) {
	entries, err := readIFDEntries(tiff, order, offset)
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		if entry.tag == tag {
			if entry.typ != 4 || entry.count != 1 {
				return 0, fmt.Errorf("tag %#x has unexpected TIFF type/count", tag)
			}
			return order.Uint32(entry.value[:]), nil
		}
	}
	return 0, fmt.Errorf("tag %#x not found", tag)
}

type tiffIFDEntry struct {
	tag   uint16
	typ   uint16
	count uint32
	value [4]byte
}

func readIFDEntries(
	tiff []byte,
	order binary.ByteOrder,
	offset int,
) ([]tiffIFDEntry, error) {
	if offset < 0 || offset+2 > len(tiff) {
		return nil, fmt.Errorf("IFD offset out of range")
	}
	count := int(order.Uint16(tiff[offset : offset+2]))
	start := offset + 2
	end := start + count*12
	if count < 0 || end > len(tiff) {
		return nil, fmt.Errorf("truncated IFD")
	}

	entries := make([]tiffIFDEntry, 0, count)
	for i := 0; i < count; i++ {
		base := start + i*12
		var value [4]byte
		copy(value[:], tiff[base+8:base+12])
		entries = append(entries, tiffIFDEntry{
			tag:   order.Uint16(tiff[base : base+2]),
			typ:   order.Uint16(tiff[base+2 : base+4]),
			count: order.Uint32(tiff[base+4 : base+8]),
			value: value,
		})
	}
	return entries, nil
}

type gpsIFD struct {
	tiff    []byte
	order   binary.ByteOrder
	entries map[uint16]tiffIFDEntry
}

func readGPSIFD(
	tiff []byte,
	order binary.ByteOrder,
	offset int,
) (gpsIFD, error) {
	entries, err := readIFDEntries(tiff, order, offset)
	if err != nil {
		return gpsIFD{}, err
	}
	result := gpsIFD{
		tiff:    tiff,
		order:   order,
		entries: make(map[uint16]tiffIFDEntry, len(entries)),
	}
	for _, entry := range entries {
		result.entries[entry.tag] = entry
	}
	return result, nil
}

func (gps gpsIFD) coordinate(valueTag, refTag uint16) (float64, error) {
	value, ok := gps.entries[valueTag]
	if !ok {
		return 0, fmt.Errorf("GPS tag %d missing", valueTag)
	}
	ref, ok := gps.entries[refTag]
	if !ok {
		return 0, fmt.Errorf("GPS reference tag %d missing", refTag)
	}

	degrees, err := readRationals(gps.tiff, gps.order, value)
	if err != nil {
		return 0, err
	}
	if len(degrees) != 3 {
		return 0, fmt.Errorf("expected 3 GPS rationals, got %d", len(degrees))
	}

	sign := float64(1)
	reference, err := readASCII(gps.tiff, gps.order, ref)
	if err != nil {
		return 0, err
	}
	if reference == "S" || reference == "W" {
		sign = -1
	}

	return sign * (degrees[0] + degrees[1]/60 + degrees[2]/3600), nil
}

func readRationals(
	tiff []byte,
	order binary.ByteOrder,
	entry tiffIFDEntry,
) ([]float64, error) {
	if entry.typ != 5 {
		return nil, fmt.Errorf("expected TIFF RATIONAL, got type %d", entry.typ)
	}
	offset := int(order.Uint32(entry.value[:]))
	byteCount := int(entry.count) * 8
	if offset < 0 || offset+byteCount > len(tiff) {
		return nil, fmt.Errorf("rational data out of range")
	}

	values := make([]float64, entry.count)
	for i := range values {
		base := offset + i*8
		numerator := order.Uint32(tiff[base : base+4])
		denominator := order.Uint32(tiff[base+4 : base+8])
		if denominator == 0 {
			return nil, fmt.Errorf("zero rational denominator")
		}
		values[i] = float64(numerator) / float64(denominator)
	}
	return values, nil
}

func readASCII(
	tiff []byte,
	order binary.ByteOrder,
	entry tiffIFDEntry,
) (string, error) {
	if entry.typ != 2 {
		return "", fmt.Errorf("expected TIFF ASCII, got type %d", entry.typ)
	}
	if entry.count == 0 {
		return "", nil
	}

	var data []byte
	if entry.count <= 4 {
		data = entry.value[:entry.count]
	} else {
		offset := int(order.Uint32(entry.value[:]))
		if offset < 0 || offset+int(entry.count) > len(tiff) {
			return "", fmt.Errorf("ASCII data out of range")
		}
		data = tiff[offset : offset+int(entry.count)]
	}
	return strings.TrimRight(string(data), "\x00"), nil
}
