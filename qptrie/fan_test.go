package qptrie

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddToFanNode_NoPrefix_Nibble5_Shift0(t *testing.T) {
	t.Parallel()

	const (
		nibSize     = 5
		bitmapWidth = (uint64(1) << nibSize) + 1 // one extra bit to encode an empty key
		bitmapMask  = (uint64(1) << bitmapWidth) - 1
		emptyBit    = uint64(1) << (bitmapWidth - 1)
	)

	var node = newFanNode(0, nibSize, 0, 0)

	node.bitpack |= emptyBit
	node.pointer = unsafe.Pointer(newLeaf("", 0, "empty"))

	var (
		key1, _ = bitStringToString("11100111_10001000")
		key2, _ = bitStringToString("01100110")
	)

	require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

	addToFanNode(node, key1, "one")
	addToFanNode(node, key2, "two")

	require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

	var (
		actPfxSize = int(node.bitpack&pfxSizeMask) >> pfxSizeOffset
		actNibSize = int(node.bitpack&nibSizeMask) >> nibSizeOffset
		actBitmapWidth = (uint64(1) << actNibSize) + 1 // one extra bit to encode an empty key
		actBitmapMask  = (uint64(1) << actBitmapWidth) - 1
	)

	require.Equal(t, 0, actPfxSize)
	require.Equal(t, 5, actNibSize)


	var (
		expected = emptyBit | (uint64(1) << 0b00111) | (uint64(1) << 0b00110)
		actual   = node.bitpack & actBitmapMask
	)

	assert.Equal(t, expected, actual)

	// check if both keys lead to the respective values
	twig1, _, ok := node.findClosest(key1)
	assert.True(t, twig1.bitpack & leafBitMask != 0)
	assert.True(t, ok)

	twig2, _, ok := node.findClosest(key2)
	assert.True(t, twig2.bitpack & leafBitMask != 0)
	assert.True(t, ok)
}

func TestAddToFanNode_NoPrefix_Nibble5_Shift5(t *testing.T) {
	t.Parallel()

	const (
		nibSize     = 5
		shift       = 5
		bitmapWidth = (uint64(1) << nibSize) + 1 // one extra bit to encode an empty key
		bitmapMask  = (uint64(1) << bitmapWidth) - 1
		emptyBit    = uint64(1) << (bitmapWidth - 1)
	)

	var node = newFanNode(shift, nibSize, 0, 0)

	node.bitpack |= emptyBit
	node.pointer = unsafe.Pointer(newLeaf("", 0, "empty"))

	var (
		key1, _ = bitStringToString("11100111_10001000")
		key2, _ = bitStringToString("01100110")
	)

	require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

	addToFanNode(node, key1, "one")
	addToFanNode(node, key2, "two")

	require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

	var (
		actPfxSize = int(node.bitpack&pfxSizeMask) >> pfxSizeOffset
		actNibSize = int(node.bitpack&nibSizeMask) >> nibSizeOffset
		actBitmapWidth = (uint64(1) << actNibSize) + 1 // one extra bit to encode an empty key
		actBitmapMask  = (uint64(1) << actBitmapWidth) - 1
	)

	require.Equal(t, 0, actPfxSize)
	require.Equal(t, 5, actNibSize)


	var (
		expected = emptyBit | (uint64(1) << 0b01111) | (uint64(1) << 0b00011)
		actual   = node.bitpack & actBitmapMask
	)

	assert.Equal(t, expected, actual)

	// check if both keys lead to the respective values
	twig1, _, ok := node.findClosest(key1)
	assert.True(t, twig1.bitpack & leafBitMask != 0)
	assert.True(t, ok)

	twig2, _, ok := node.findClosest(key2)
	assert.True(t, twig2.bitpack & leafBitMask != 0)
	assert.True(t, ok)
}

func TestAddToFanNode_MatchedPrefix3_Nibble4_Shift0(t *testing.T) {
	t.Parallel()

	const (
		pfxSize     = 3
		nibSize     = 4
		prefix      = 0b_110
		bitmapWidth = (uint64(1) << nibSize) + 1 // one extra bit to encode an empty key
		bitmapMask  = (uint64(1) << bitmapWidth) - 1
		emptyBit    = uint64(1) << (bitmapWidth - 1)
	)

	var node = newFanNode(0, nibSize, pfxSize, prefix)

	node.bitpack |= emptyBit
	node.pointer = unsafe.Pointer(newLeaf("", 0, "empty"))

	var (
		key1, _ = bitStringToString("011_00111_10001000")
		key2, _ = bitStringToString("011_10110")
	)

	require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

	addToFanNode(node, key1, "one")
	addToFanNode(node, key2, "two")

	require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

	var (
		actPfxSize = int(node.bitpack&pfxSizeMask) >> pfxSizeOffset
		actNibSize = int(node.bitpack&nibSizeMask) >> nibSizeOffset
		actBitmapWidth = (uint64(1) << actNibSize) + 1 // one extra bit to encode an empty key
		actBitmapMask  = (uint64(1) << actBitmapWidth) - 1
	)

	require.Equal(t, 3, actPfxSize)
	require.Equal(t, 4, actNibSize)


	var (
		expected = emptyBit | (uint64(1) << 0b1100) | (uint64(1) << 0b1101)
		actual   = node.bitpack & actBitmapMask
	)

	assert.Equal(t, expected, actual)

	// check if both keys lead to the respective values
	twig1, _, ok := node.findClosest(key1)
	assert.True(t, twig1.bitpack & leafBitMask != 0)
	assert.True(t, ok)

	twig2, _, ok := node.findClosest(key2)
	assert.True(t, twig2.bitpack & leafBitMask != 0)
	assert.True(t, ok)
}

func TestAddToFanNode_MatchedPrefix4_Nibble3_Shift4(t *testing.T) {
	t.Parallel()

	const (
		shift       = 4
		pfxSize     = 4
		nibSize     = 3
		prefix      = 0b_1101
		bitmapWidth = (uint64(1) << nibSize) + 1 // one extra bit to encode an empty key
		bitmapMask  = (uint64(1) << bitmapWidth) - 1
		emptyBit    = uint64(1) << (bitmapWidth - 1)
	)

	var node = newFanNode(shift, nibSize, pfxSize, prefix)

	node.bitpack |= emptyBit
	node.pointer = unsafe.Pointer(newLeaf("", 0, "empty"))

	var (
		key1, _ = bitStringToString("0110_1011_10111000")
		key2, _ = bitStringToString("0111_1011")
	)

	require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

	addToFanNode(node, key1, "one")
	addToFanNode(node, key2, "two")

	require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

	var (
		actPfxSize = int(node.bitpack&pfxSizeMask) >> pfxSizeOffset
		actNibSize = int(node.bitpack&nibSizeMask) >> nibSizeOffset
		actBitmapWidth = (uint64(1) << actNibSize) + 1 // one extra bit to encode an empty key
		actBitmapMask  = (uint64(1) << actBitmapWidth) - 1
	)

	require.Equal(t, 4, actPfxSize)
	require.Equal(t, 3, actNibSize)


	var (
		expected = emptyBit | (uint64(1) << 0b101)
		actual   = node.bitpack & actBitmapMask
	)

	assert.Equal(t, expected, actual)

	// check if both keys lead to the respective values
	twig1, _, ok := node.findClosest(key1)
	assert.True(t, twig1.bitpack & leafBitMask != 0)
	assert.True(t, ok)

	twig2, _, ok := node.findClosest(key2)
	assert.True(t, twig2.bitpack & leafBitMask != 0)
	assert.True(t, ok)
}

func TestAddToFanNode_UnmatchedPrefix3_Nibble4_Shift0(t *testing.T) {
	t.Parallel()

	const (
		shift       = 0
		pfxSize     = 3
		nibSize     = 4
		prefix      = 0b_110
		bitmapWidth = (uint64(1) << nibSize) + 1 // one extra bit to encode an empty key
		bitmapMask  = (uint64(1) << bitmapWidth) - 1
		emptyBit    = uint64(1) << (bitmapWidth - 1)
	)

	var node = newFanNode(shift, nibSize, pfxSize, prefix)

	node.bitpack |= emptyBit
	node.pointer = unsafe.Pointer(newLeaf("", 0, "empty"))

	var (
		key1, _ = bitStringToString("011_0011_1_10001000")
		key2, _ = bitStringToString("001_1011_0")
	)

	require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

	addToFanNode(node, key1, "one")
	addToFanNode(node, key2, "two")

	require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

	var (
		actPfxSize = int(node.bitpack&pfxSizeMask) >> pfxSizeOffset
		actNibSize = int(node.bitpack&nibSizeMask) >> nibSizeOffset
		actBitmapWidth = (uint64(1) << actNibSize) + 1 // one extra bit to encode an empty key
		actBitmapMask  = (uint64(1) << actBitmapWidth) - 1
	)

	require.Equal(t, 0, actPfxSize)
	require.Equal(t, 3, actNibSize)

	var (
		expected = (uint64(1) << 0b110) | (uint64(1) << 0b100)
		actual   = node.bitpack & actBitmapMask
	)

	assert.Equal(t, expected, actual)

	// check if both keys lead to the respective values
	twig1, _, ok := node.findClosest(key1)
	assert.True(t, twig1.bitpack & leafBitMask != 0)
	assert.True(t, ok)

	twig2, _, ok := node.findClosest(key2)
	assert.True(t, twig2.bitpack & leafBitMask != 0)
	assert.True(t, ok)
}

func TestAddToFanNode_UnmatchedPrefix7_Nibble2_Shift0(t *testing.T) {
	t.Parallel()

	const (
		shift       = 0
		pfxSize     = 7
		nibSize     = 2
		prefix      = 0b_1111101
		bitmapWidth = (uint64(1) << nibSize) + 1 // one extra bit to encode an empty key
		bitmapMask  = (uint64(1) << bitmapWidth) - 1
		emptyBit    = uint64(1) << (bitmapWidth - 1)
	)

	var node = newFanNode(shift, nibSize, pfxSize, prefix)

	node.bitpack |= emptyBit
	node.pointer = unsafe.Pointer(newLeaf("", 0, "empty"))

	var (
		key1, _ = bitStringToString("1011111_0_11001000")
		key2, _ = bitStringToString("1011110_1_00110111")
	)

	require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

	addToFanNode(node, key1, "one")
	addToFanNode(node, key2, "two")

	require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

	var (
		actPfxSize     = int(node.bitpack&pfxSizeMask) >> pfxSizeOffset
		actPfxOffset   = pfxSizeOffset - actPfxSize
		actPfxMask     = (uint64(1) << actPfxSize) - 1
		actPfx         = (node.bitpack >> actPfxOffset) & actPfxMask
		actNibSize     = int(node.bitpack&nibSizeMask) >> nibSizeOffset
		actBitmapWidth = (uint64(1) << actNibSize) + 1 // one extra bit to encode an empty key
		actBitmapMask  = (uint64(1) << actBitmapWidth) - 1
	)

	require.Equal(t, 2, actPfxSize)
	require.Equal(t, uint64(0b01), actPfx)
	require.Equal(t, 5, actNibSize)

	var (
		expected = (uint64(1) << 0b11111) | (uint64(1) << 0b01111)
		actual   = node.bitpack & actBitmapMask
	)

	assert.Equal(t, expected, actual)

	// check if both keys lead to the respective values
	twig1, _, ok := node.findClosest(key1)
	assert.True(t, twig1.bitpack & leafBitMask != 0)
	assert.True(t, ok)

	twig2, _, ok := node.findClosest(key2)
	assert.True(t, twig2.bitpack & leafBitMask != 0)
	assert.True(t, ok)
}

func TestAddToFanNode_UnmatchedPrefix4_Nibble4_Shift6(t *testing.T) {
	t.Parallel()

	const (
		shift       = 6
		pfxSize     = 4
		nibSize     = 4
		prefix      = 0b_1111
		bitmapWidth = (uint64(1) << nibSize) + 1 // one extra bit to encode an empty key
		bitmapMask  = (uint64(1) << bitmapWidth) - 1
		emptyBit    = uint64(1) << (bitmapWidth - 1)
	)

	var node = newFanNode(shift, nibSize, pfxSize, prefix)

	node.bitpack |= emptyBit
	node.pointer = unsafe.Pointer(newLeaf("", 0, "empty"))

	var (
		key1, _ = bitStringToString("010011_11_11_001110")
		key2, _ = bitStringToString("001011_11")
	)

	require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

	addToFanNode(node, key1, "one")
	addToFanNode(node, key2, "two")

	require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

	var (
		actShift   = int(node.bitpack&nibShiftMask) >> nibShiftOffset
		actPfxSize = int(node.bitpack&pfxSizeMask) >> pfxSizeOffset
		actNibSize = int(node.bitpack&nibSizeMask) >> nibSizeOffset
		actBitmapWidth = (uint64(1) << actNibSize) + 1 // one extra bit to encode an empty key
		actBitmapMask  = (uint64(1) << actBitmapWidth) - 1
	)

	require.Equal(t, 6, actShift)
	require.Equal(t, 0, actPfxSize)
	require.Equal(t, 4, actNibSize)

	var (
		expected = (uint64(1) << 0b1111) | (uint64(1) << 0b0011)
		actual   = node.bitpack & actBitmapMask
	)

	assert.Equal(t, expected, actual)

	// check if both keys lead to the respective values
	twig1, _, ok := node.findClosest(key1)
	assert.True(t, twig1.bitpack & leafBitMask != 0)
	assert.True(t, ok)

	twig2, _, ok := node.findClosest(key2)
	assert.True(t, twig2.bitpack & leafBitMask != 0)
	assert.True(t, ok)
}
