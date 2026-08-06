// Copyright(c) vice contributors, licensed under the GNU Public License, Version 3.
// SPDX: GPL-3.0-only

package renderer

import (
	gomath "math"
	"sync"

	"github.com/mmp/vice/util"
)

// TexturedQuadBuffers stores geometry batched for one texture.
type TexturedQuadBuffers struct {
	p       [][2]float32
	uv      [][2]float32
	rgb     []RGB
	indices []int32
}

func (b *TexturedQuadBuffers) Reset() {
	b.p = b.p[:0]
	b.uv = b.uv[:0]
	b.rgb = b.rgb[:0]
	b.indices = b.indices[:0]
}

func (b *TexturedQuadBuffers) AddQuad(
	positions [4][2]float32,
	texCoords [4][2]float32,
	color RGB,
) {
	start := int32(len(b.p))
	b.p = append(b.p, positions[:]...)
	b.uv = append(b.uv, texCoords[:]...)
	b.rgb = append(b.rgb, color, color, color, color)
	b.indices = append(b.indices, start, start+1, start+2, start+3)
}

func (b *TexturedQuadBuffers) GenerateCommands(cb *CommandBuffer) {
	if len(b.indices) == 0 {
		return
	}

	p := cb.Float2Buffer(b.p)
	cb.VertexArray(p, 2, 2*4)

	rgb := cb.RGBBuffer(b.rgb)
	cb.RGB32Array(rgb, 3, 3*4)

	uv := cb.Float2Buffer(b.uv)
	cb.TexCoordArray(uv, 2, 2*4)

	indices := cb.IntBuffer(b.indices)
	cb.DrawQuads(indices, len(b.indices))
}

// TexturedQuadsDrawBuilder batches textured quads by texture ID.
type TexturedQuadsDrawBuilder struct {
	quads map[uint32]*TexturedQuadBuffers
}

func (b *TexturedQuadsDrawBuilder) AddQuad(
	textureID uint32,
	positions [4][2]float32,
	texCoords [4][2]float32,
	color RGB,
) {
	if textureID == 0 {
		return
	}

	if b.quads == nil {
		b.quads = make(map[uint32]*TexturedQuadBuffers)
	}

	buffers, ok := b.quads[textureID]
	if !ok {
		buffers = &TexturedQuadBuffers{}
		b.quads[textureID] = buffers
	}

	buffers.AddQuad(positions, texCoords, color)
}

// AddSprite adds a centered rectangular sprite. rotationRadians is clockwise
// from the positive vertical axis, matching Tower Cab heading handling.
func (b *TexturedQuadsDrawBuilder) AddSprite(
	textureID uint32,
	center [2]float32,
	width float32,
	height float32,
	rotationRadians float32,
	color RGB,
) {
	if textureID == 0 || width <= 0 || height <= 0 {
		return
	}

	halfWidth := width / 2
	halfHeight := height / 2

	local := [4][2]float32{
		{-halfWidth, halfHeight},
		{halfWidth, halfHeight},
		{halfWidth, -halfHeight},
		{-halfWidth, -halfHeight},
	}

	sinRotation := float32(gomath.Sin(float64(rotationRadians)))
	cosRotation := float32(gomath.Cos(float64(rotationRadians)))

	var positions [4][2]float32
	for i, point := range local {
		positions[i] = [2]float32{
			center[0] + point[0]*cosRotation + point[1]*sinRotation,
			center[1] - point[0]*sinRotation + point[1]*cosRotation,
		}
	}

	texCoords := [4][2]float32{
		{0, 0},
		{1, 0},
		{1, 1},
		{0, 1},
	}

	b.AddQuad(textureID, positions, texCoords, color)
}

func (b *TexturedQuadsDrawBuilder) Reset() {
	for _, buffers := range b.quads {
		buffers.Reset()
	}
}

func (b *TexturedQuadsDrawBuilder) GenerateCommands(cb *CommandBuffer) {
	if len(b.quads) == 0 {
		return
	}

	cb.Blend()

	for textureID, buffers := range util.SortedMap(b.quads) {
		if len(buffers.indices) == 0 {
			continue
		}

		cb.EnableTexture(textureID)
		buffers.GenerateCommands(cb)
	}

	cb.DisableVertexArray()
	cb.DisableColorArray()
	cb.DisableTexCoordArray()
	cb.DisableTexture()
	cb.DisableBlend()
}

var texturedQuadsDrawBuilderPool = sync.Pool{
	New: func() any {
		return &TexturedQuadsDrawBuilder{}
	},
}

func GetTexturedQuadsDrawBuilder() *TexturedQuadsDrawBuilder {
	return texturedQuadsDrawBuilderPool.Get().(*TexturedQuadsDrawBuilder)
}

func ReturnTexturedQuadsDrawBuilder(builder *TexturedQuadsDrawBuilder) {
	builder.Reset()
	texturedQuadsDrawBuilderPool.Put(builder)
}
