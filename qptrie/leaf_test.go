package qptrie

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddToLeaf(t *testing.T) {
	t.Parallel()

	const (
		typeFanNode byte = 0b_00
		typeCutNode byte = 0b_01
	)

	for _, tcase := range []*struct {
		Name       string
		Shift      int
		Key1, Key2 string

		ExpType    byte
		ExpShift   int
		ExpCut     string
		ExpPfxSize int
		ExpNibSize int
		ExpPfx     uint64
		ExpBitmap  uint64
	}{
		{
			Name: "1-byte keys, diff-bit:0", Shift: 0,
			Key1:    "01100110",
			Key2:    "11010110",
			ExpType: typeFanNode, ExpShift: 0, ExpPfxSize: 0, ExpNibSize: 5, ExpPfx: 0b0,
			ExpBitmap: uint64(1)<<0b00110 | uint64(1)<<0b01011,
		},
		{
			Name: "1-byte keys, diff-bit:7", Shift: 0,
			Key1:    "11100011",
			Key2:    "11100010",
			ExpType: typeFanNode, ExpShift: 0, ExpPfxSize: 0, ExpNibSize: 5, ExpPfx: 0b0,
			ExpBitmap: uint64(1) << 0b00111,
		},
		{
			Name: "1-byte keys, diff-bit:5", Shift: 4,
			Key1:    "1110_0110",
			Key2:    "1110_0010",
			ExpType: typeFanNode, ExpShift: 4, ExpPfxSize: 0, ExpNibSize: 5, ExpPfx: 0b0,
			ExpBitmap: uint64(1)<<0b00110 | uint64(1)<<0b00100,
		},
		{
			Name: "empty key and 1-byte key", Shift: 0,
			Key1:    "",
			Key2:    "11010110",
			ExpType: typeFanNode, ExpShift: 0, ExpPfxSize: 0, ExpNibSize: 5, ExpPfx: 0b0,
			ExpBitmap: uint64(1)<<32 | uint64(1)<<0b01011,
		},
		{
			Name: "1-byte key and empty key", Shift: 0,
			Key1:    "11010110",
			Key2:    "",
			ExpType: typeFanNode, ExpShift: 0, ExpPfxSize: 0, ExpNibSize: 5, ExpPfx: 0b0,
			ExpBitmap: uint64(1)<<32 | uint64(1)<<0b01011,
		},
		{
			Name: "2-byte keys, diff-bit:13", Shift: 0,
			Key1:    "11010110_11100101",
			Key2:    "11010110_11101101",
			ExpType: typeCutNode, ExpShift: 0, ExpCut: "11010110",
		},
		{
			Name: "1-byte key and 2-byte key", Shift: 0,
			Key1:    "11010110",
			Key2:    "11010110_11101101",
			ExpType: typeCutNode, ExpShift: 0, ExpCut: "11010110",
		},
		{
			Name: "2-byte key and 1-byte key", Shift: 0,
			Key1:    "11010110_11101101",
			Key2:    "11010110",
			ExpType: typeCutNode, ExpShift: 0, ExpCut: "11010110",
		},
	} {
		name := fmt.Sprintf("%v, shift:%v", tcase.Name, tcase.Shift)

		t.Run(name, func(t *testing.T) {
			var (
				key1, _ = bitStringToString(tcase.Key1)
				key2, _ = bitStringToString(tcase.Key2)
			)

			twig := newLeaf(key1, tcase.Shift, "one")

			require.True(t, twig.IsLeaf(), "should be a leaf, not a node")

			addToLeaf(twig, key2, "two")

			twigType := byte((twig.bitpack >> cutBitOffset) & 0b_11)

			require.Equal(t, tcase.ExpType, twigType)

			switch twigType {
			case typeFanNode:
				actPrefix, actPfxSize := twig.FanPrefix()

				require.Equal(t, tcase.ExpShift, twig.Shift())
				require.Equal(t, tcase.ExpNibSize, twig.FanNibbleSize())
				require.Equal(t, tcase.ExpPfxSize, actPfxSize)
				require.Equal(t, tcase.ExpPfx, actPrefix)

				bitmap, _ := twig.FanBitmap()

				assert.Equal(t, tcase.ExpBitmap, bitmap)

			case typeCutNode:
				expected, _ := bitStringToString(tcase.ExpCut)
				assert.Equal(t, expected, getCutNodeKey(twig))

			default:
				t.Errorf("unexpected type: %v", twigType)
				t.Fail()
			}

			// check if both keys lead to the respective values
			twig1, _, ok := findClosest(twig, key1)
			require.True(t, twig1.IsLeaf())
			assert.True(t, ok)
			assert.Equal(t, "one", getLeafKV(twig1).Value)

			twig2, _, ok := findClosest(twig, key2)
			require.True(t, twig2.IsLeaf())
			assert.True(t, ok)
			assert.Equal(t, "two", getLeafKV(twig2).Value)
		})
	}
}
