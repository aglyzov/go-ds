package qptrie

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFanPrefixSize(t *testing.T) {
	t.Parallel()

	for _, tcase := range []*struct {
		Twig     *Twig
		Expected int
	}{
		{newFanNode(7, 1, 0, 0), 0},
		{newFanNode(1, 2, 1, 0), 1},
		{newFanNode(0, 3, 13, 0), 13},
		{newFanNode(5, 4, 28, 0), 28},
		{newFanNode(2, 5, 15, 0), 15},
	} {
		tcase := tcase

		t.Run(strconv.Itoa(tcase.Expected)+"bit prefix", func(t *testing.T) {
			actual := fanPrefixSize(tcase.Twig)

			assert.Equal(t, tcase.Expected, actual)
		})
	}
}

func TestFanPrefixMax(t *testing.T) {
	t.Parallel()

	for _, tcase := range []*struct {
		Twig     *Twig
		Expected int
	}{
		{newFanNode(7, 1, 0, 0), 47},
		{newFanNode(1, 2, 1, 0), 45},
		{newFanNode(0, 3, 12, 0), 41},
		{newFanNode(5, 4, 0, 0), 33},
		{newFanNode(2, 5, 16, 0), 17},
	} {
		tcase := tcase

		t.Run(strconv.Itoa(fanNibbleSize(tcase.Twig))+"bit nibble", func(t *testing.T) {
			actual := fanPrefixMax(tcase.Twig)

			assert.Equal(t, tcase.Expected, actual)
		})
	}
}

func TestFanBitmapSize(t *testing.T) {
	t.Parallel()

	for _, tcase := range []*struct {
		Twig     *Twig
		Expected int
	}{
		{newFanNode(7, 1, 0, 0), 3},
		{newFanNode(1, 2, 1, 0), 5},
		{newFanNode(0, 3, 12, 0), 9},
		{newFanNode(5, 4, 0, 0), 17},
		{newFanNode(2, 5, 16, 0), 33},
	} {
		tcase := tcase

		t.Run(strconv.Itoa(fanNibbleSize(tcase.Twig))+"bit nibble", func(t *testing.T) {
			actual := fanBitmapSize(tcase.Twig)

			assert.Equal(t, tcase.Expected, actual)
		})
	}
}

func TestAddToFanNode(t *testing.T) {
	t.Parallel()

	type KV struct {
		Key   string // bit string, e.g. "1100_1100_00011110" (left-to-right)
		Value any
	}
	type Query struct {
		Key   string // bit string, e.g. "1100_1100_00011110" (left-to-right)
		OK    bool
		Value any
	}

	for _, tcase := range []*struct {
		Desc      string
		Shift     int
		PfxSize   int
		NibSize   int
		Prefix    uint64
		KeyValues []*KV

		ExpShift   int
		ExpPfxSize int
		ExpNibSize int
		ExpPrefix  uint64
		ExpBitmap  uint64
		Queries    []*Query
	}{
		{
			Desc:  "single empty key",
			Shift: 0, PfxSize: 0, NibSize: 2,
			Prefix: 0b0,
			KeyValues: []*KV{
				{"", "A"},
			},
			ExpShift: 0, ExpPfxSize: 0, ExpNibSize: 2, ExpPrefix: 0b0,
			ExpBitmap: 0b10000,
			Queries: []*Query{
				{"", true, "A"},
				{"01_011010", false, ""},
			},
		},
		{
			Desc:  "two keys, one is empty",
			Shift: 0, PfxSize: 0, NibSize: 2,
			Prefix: 0b0,
			KeyValues: []*KV{
				{"", "A"},
				{"01_011010", "B"},
			},
			ExpShift: 0, ExpPfxSize: 0, ExpNibSize: 2, ExpPrefix: 0b0,
			ExpBitmap: 0b10000 | (uint64(1) << 0b10),
			Queries: []*Query{
				{"", true, "A"},
				{"01_011010", true, "B"},
				{"01_011011", false, ""},
			},
		},
		{
			Desc:  "two 1B keys, different nibbles",
			Shift: 0, PfxSize: 0, NibSize: 3,
			Prefix: 0b0,
			KeyValues: []*KV{
				{"110_11101", "A"},
				{"011_11010", "B"},
			},
			ExpShift: 0, ExpPfxSize: 0, ExpNibSize: 3, ExpPrefix: 0b0,
			ExpBitmap: (uint64(1) << 0b011) | (uint64(1) << 0b110),
			Queries: []*Query{
				{"", false, ""},
				{"110_11101", true, "A"},
				{"011_11010", true, "B"},
				{"011_11011", false, ""},
			},
		},
		{
			Desc:  "2B and 1B keys, different nibbles",
			Shift: 0, PfxSize: 0, NibSize: 5,
			Prefix: 0b0,
			KeyValues: []*KV{
				{"11100_11110001000", "A"},
				{"01100_110", "B"},
			},
			ExpShift: 0, ExpPfxSize: 0, ExpNibSize: 5, ExpPrefix: 0b0,
			ExpBitmap: (uint64(1) << 0b00111) | (uint64(1) << 0b00110),
			Queries: []*Query{
				{"", false, ""},
				{"11100_11110001000", true, "A"},
				{"01100_110", true, "B"},
				{"01100_111", false, ""},
			},
		},
		{
			Desc:  "shifted 2B and 1B keys, one nibble is padded",
			Shift: 5, PfxSize: 0, NibSize: 5,
			Prefix: 0b0,
			KeyValues: []*KV{
				{"11100_11110_001000", "A"},
				{"11100_110", "B"},
			},
			ExpShift: 5, ExpPfxSize: 0, ExpNibSize: 5, ExpPrefix: 0b0,
			ExpBitmap: (uint64(1) << 0b01111) | (uint64(1) << 0b00011),
			Queries: []*Query{
				{"", false, ""},
				{"11100_11110_001000", true, "A"},
				{"11100_110", true, "B"},
				{"11100_111", false, ""},
			},
		},
		{
			Desc:  "same prefix, different nibbles",
			Shift: 0, PfxSize: 3, NibSize: 4,
			Prefix: 0b_110,
			KeyValues: []*KV{
				{"011_0011_110001000", "A"},
				{"011_1011_0", "B"},
			},
			ExpShift: 0, ExpPfxSize: 3, ExpNibSize: 4, ExpPrefix: 0b110,
			ExpBitmap: (uint64(1) << 0b1100) | (uint64(1) << 0b1101),
			Queries: []*Query{
				{"", false, ""},
				{"011_0011_110001000", true, "A"},
				{"011_1011_0", true, "B"},
			},
		},
		{
			Desc:  "shifted 2B and 1B keys, same prefix, one nibble is empty",
			Shift: 4, PfxSize: 4, NibSize: 3,
			Prefix: 0b_1101,
			KeyValues: []*KV{
				{"0111_1011_101_11000", "A"},
				{"0111_1011", "B"},
			},
			ExpShift: 4, ExpPfxSize: 4, ExpNibSize: 3, ExpPrefix: 0b1101,
			ExpBitmap: uint64(0b100000000) | (uint64(1) << 0b101),
			Queries: []*Query{
				{"", false, ""},
				{"0110_1011_101_11000", true, "A"},
				{"0111_1011", true, "B"},
			},
		},
		{
			Desc:  "2B and 1B keys, different prefixes",
			Shift: 0, PfxSize: 3, NibSize: 3,
			Prefix: 0b_110,
			KeyValues: []*KV{
				{"011_001_1110001000", "A"},
				{"001_101_10", "B"},
			},
			ExpShift: 0, ExpPfxSize: 0, ExpNibSize: 3, ExpPrefix: 0b0,
			ExpBitmap: (uint64(1) << 0b110) | (uint64(1) << 0b100),
			Queries: []*Query{
				{"", false, ""},
				{"011_001_1110001000", true, "A"},
				{"001_101_10", true, "B"},
			},
		},
		{
			Desc:  "two 2B keys, different prefixes",
			Shift: 0, PfxSize: 7, NibSize: 2,
			Prefix: 0b_1111101,
			KeyValues: []*KV{
				{"1011111_01_1001000", "A"},
				{"1011110_10_0110111", "B"},
			},
			ExpShift: 0, ExpPfxSize: 2, ExpNibSize: 5, ExpPrefix: 0b01,
			ExpBitmap: (uint64(1) << 0b11111) | (uint64(1) << 0b01111),
			Queries: []*Query{
				{"", false, ""},
				{"1011111_01_1001000", true, "A"},
				{"1011110_10_0110111", true, "B"},
			},
		},
		{
			Desc:  "shifted 2B and 1B keys, different prefixes, one prefix is padded",
			Shift: 6, PfxSize: 4, NibSize: 4,
			Prefix: 0b_1111,
			KeyValues: []*KV{
				{"010011_1111_0011_10", "A"},
				{"010011_11", "B"},
			},
			ExpShift: 6, ExpPfxSize: 0, ExpNibSize: 2, ExpPrefix: 0,
			ExpBitmap: (uint64(1) << 0b11),
			Queries: []*Query{
				{"", false, ""},
				{"010011_1111_0011_10", true, "A"},
				{"010011_11", true, "B"},
			},
		},
		// TODO: add a test case where the second key is smaller than a prefix
		// TODO: add support for Cut (?)
		{
			Desc:  "shifted 2B and 1B keys, same prefixes, one prefix is padded",
			Shift: 3, PfxSize: 6, NibSize: 4,
			Prefix: 0b_001101,
			KeyValues: []*KV{
				{"010_101100_1001_110", "A"},
				{"010_10110", "B"},
			},
			ExpShift: 3, ExpPfxSize: 0, ExpNibSize: 5, ExpPrefix: 0,
			ExpBitmap: (uint64(1) << 0b01101),
			Queries: []*Query{
				{"", false, ""},
				{"010_101100_1001_110", true, "A"},
				{"010_10110", true, "B"},
			},
		},
		// TODO: add a test case where the second key is smaller than a nib and the first key
		//       continues with zeros
	} {
		var (
			tcase = tcase
			keys  []string
		)

		for _, kv := range tcase.KeyValues {
			keys = append(keys, kv.Key)
		}

		name := fmt.Sprintf(
			"[%v] shift:%v, prefix:%v, nib:%v, keys:[%v]",
			tcase.Desc, tcase.Shift, tcase.PfxSize, tcase.NibSize, strings.Join(keys, ","),
		)

		t.Run(name, func(t *testing.T) {
			node := newFanNode(tcase.Shift, tcase.NibSize, tcase.PfxSize, tcase.Prefix)

			for _, kv := range tcase.KeyValues {
				key, err := bitStringToString(kv.Key)
				require.NoError(t, err)

				addToFanNode(node, key, kv.Value, false)

				t.Logf("[+] set %v = %v: %v", kv.Key, kv.Value, node)

				require.True(t, node.IsFanNode())
			}

			var (
				actPrefix, actPfxSize = fanPrefix(node)
				actBitmap, _          = fanBitmap(node)
			)

			require.Equal(t, tcase.ExpShift, node.Shift())
			require.Equal(t, tcase.ExpNibSize, fanNibbleSize(node))
			require.Equal(t, tcase.ExpPfxSize, actPfxSize)
			require.Equal(t, tcase.ExpPrefix, actPrefix)

			assert.Equal(t, tcase.ExpBitmap, actBitmap)

			// check if all keys lead to the expected values
			for _, query := range tcase.Queries {
				key, err := bitStringToString(query.Key)

				assert.NoError(t, err)

				twig, _, ok := findClosest(node, key)

				t.Logf("[=] queried %v: %v, %v", query.Key, ok, twig)

				assert.Equal(t, query.OK, ok)

				if ok {
					if assert.True(t, twig.IsLeaf()) {
						assert.Equal(t, query.Value, getLeafKV(twig).Value)
					}
				}
			}
		})
	}
}
