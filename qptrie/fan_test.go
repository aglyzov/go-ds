package qptrie

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddToFanNode(t *testing.T) {
	t.Parallel()

	for _, tcase := range []*struct {
		Shift      int
		PfxSize    int
		NibSize    int
		Pfx        uint32
		Key1, Key2 string
		ExpShift   int
		ExpPfxSize int
		ExpNibSize int
		ExpPfx     uint32
		ExpBitmap  uint64
	}{
		{
			Shift: 0, PfxSize: 0, NibSize: 5,
			Pfx:      0b0,
			Key1:     "11100111_10001000",
			Key2:     "01100110",
			ExpShift: 0, ExpPfxSize: 0, ExpNibSize: 5, ExpPfx: 0b0,
			ExpBitmap: uint64(1)<<32 | (uint64(1) << 0b00111) | (uint64(1) << 0b00110),
		},
		{
			Shift: 5, PfxSize: 0, NibSize: 5,
			Pfx:      0b0,
			Key1:     "11100111_10001000",
			Key2:     "01100110",
			ExpShift: 5, ExpPfxSize: 0, ExpNibSize: 5, ExpPfx: 0b0,
			ExpBitmap: uint64(1)<<32 | (uint64(1) << 0b01111) | (uint64(1) << 0b00011),
		},
		{
			Shift: 0, PfxSize: 3, NibSize: 4,
			Pfx:      0b_110,
			Key1:     "011_00111_10001000",
			Key2:     "011_10110",
			ExpShift: 0, ExpPfxSize: 3, ExpNibSize: 4, ExpPfx: 0b110,
			ExpBitmap: uint64(1)<<16 | (uint64(1) << 0b1100) | (uint64(1) << 0b1101),
		},
		{
			Shift: 4, PfxSize: 4, NibSize: 3,
			Pfx:      0b_1101,
			Key1:     "0110_1011_10111000",
			Key2:     "0111_1011",
			ExpShift: 4, ExpPfxSize: 4, ExpNibSize: 3, ExpPfx: 0b1101,
			ExpBitmap: uint64(0b100000000) | (uint64(1) << 0b101),
		},
		{
			Shift: 0, PfxSize: 3, NibSize: 3,
			Pfx:      0b_110,
			Key1:     "011_0011_1_10001000",
			Key2:     "001_1011_0",
			ExpShift: 0, ExpPfxSize: 0, ExpNibSize: 3, ExpPfx: 0b0,
			ExpBitmap: (uint64(1) << 0b110) | (uint64(1) << 0b100),
		},
		{
			Shift: 0, PfxSize: 7, NibSize: 2,
			Pfx:      0b_1111101,
			Key1:     "1011111_0_11001000",
			Key2:     "1011110_1_00110111",
			ExpShift: 0, ExpPfxSize: 2, ExpNibSize: 5, ExpPfx: 0b01,
			ExpBitmap: (uint64(1) << 0b11111) | (uint64(1) << 0b01111),
		},
		{
			Shift: 6, PfxSize: 4, NibSize: 4,
			Pfx:      0b_1111,
			Key1:     "010011_11_11_001110",
			Key2:     "001011_11",
			ExpShift: 6, ExpPfxSize: 0, ExpNibSize: 4, ExpPfx: 0,
			ExpBitmap: (uint64(1) << 0b1111) | (uint64(1) << 0b0011),
		},
	} {
		name := fmt.Sprintf("shift:%v, prefix:%v, nib:%v", tcase.Shift, tcase.PfxSize, tcase.NibSize)

		t.Run(name, func(t *testing.T) {
			var (
				bitmapWidth = (uint64(1) << tcase.NibSize) + 1 // one extra bit to encode an empty key
				emptyBit    = uint64(1) << (bitmapWidth - 1)
			)

			var node = newFanNode(tcase.Shift, tcase.NibSize, tcase.PfxSize, tcase.Pfx)

			node.bitpack |= emptyBit
			node.pointer = unsafe.Pointer(newLeaf("", 0, "empty"))

			var (
				key1, _ = bitStringToString(tcase.Key1)
				key2, _ = bitStringToString(tcase.Key2)
			)

			require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

			addToFanNode(node, key1, "one")
			addToFanNode(node, key2, "two")

			require.Zero(t, node.bitpack&leafBitMask, "should be a fan-node, not leaf")

			var (
				actShift       = int(node.bitpack&nibShiftMask) >> nibShiftOffset
				actPfxSize     = int(node.bitpack&pfxSizeMask) >> pfxSizeOffset
				actNibSize     = int(node.bitpack&nibSizeMask) >> nibSizeOffset
				actPfxOffset   = pfxSizeOffset - actPfxSize
				actPfxMask     = (uint32(1) << actPfxSize) - 1
				actPfx         = uint32(node.bitpack>>actPfxOffset) & actPfxMask
				actBitmapWidth = (uint64(1) << actNibSize) + 1 // one extra bit to encode an empty key
				actBitmapMask  = (uint64(1) << actBitmapWidth) - 1
			)

			require.Equal(t, tcase.ExpShift, actShift)
			require.Equal(t, tcase.ExpPfxSize, actPfxSize)
			require.Equal(t, tcase.ExpNibSize, actNibSize)
			require.Equal(t, tcase.ExpPfx, actPfx)

			assert.Equal(t, tcase.ExpBitmap, node.bitpack&actBitmapMask)

			// check if both keys lead to the respective values
			twig1, _, ok := node.findClosest(key1)
			require.True(t, twig1.bitpack&leafBitMask != 0)
			assert.True(t, ok)
			assert.Equal(t, "one", getLeafKV(twig1).Val)

			twig2, _, ok := node.findClosest(key2)
			require.True(t, twig2.bitpack&leafBitMask != 0)
			assert.True(t, ok)
			assert.Equal(t, "two", getLeafKV(twig2).Val)
		})
	}
}
