package qptrie

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			Desc:  "empty nibble",
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
			},
		},
		{
			Desc:  "TODO",
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
			},
		},
		{
			Desc:  "TODO",
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
			},
		},
		{
			Desc:  "TODO",
			Shift: 5, PfxSize: 0, NibSize: 5,
			Prefix: 0b0,
			KeyValues: []*KV{
				{"11100_11110_001000", "A"},
				{"01100_110", "B"},
			},
			ExpShift: 5, ExpPfxSize: 0, ExpNibSize: 5, ExpPrefix: 0b0,
			ExpBitmap: (uint64(1) << 0b01111) | (uint64(1) << 0b00011),
			Queries: []*Query{
				{"", false, ""},
				{"11100_11110_001000", true, "A"},
				{"01100_110", true, "B"},
			},
		},
		{
			Desc:  "TODO",
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
			Desc:  "TODO",
			Shift: 4, PfxSize: 4, NibSize: 3,
			Prefix: 0b_1101,
			KeyValues: []*KV{
				{"0110_1011_101_11000", "A"},
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
			Desc:  "TODO",
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
			Desc:  "TODO",
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
			Desc:  "TODO",
			Shift: 6, PfxSize: 4, NibSize: 4,
			Prefix: 0b_1111,
			KeyValues: []*KV{
				{"010011_1111_0011_10", "A"},
				{"010011_11", "B"},
			},
			ExpShift: 6, ExpPfxSize: 0, ExpNibSize: 4, ExpPrefix: 0,
			ExpBitmap: (uint64(1) << 0b1111) | (uint64(1) << 0b0011),
			Queries: []*Query{
				{"", false, ""},
				{"010011_1111_0011_10", true, "A"},
				{"010011_11", true, "B"},
			},
		},
		// TODO: add a test case where the second key is smaller than a prefix
		// TODO: add support for Cut (?)
		/*
			{
				Shift: 3, PfxSize: 6, NibSize: 4,
				Prefix:      0b_001101,
				KeyValues:     []*KV{
					{"010_101100_1001_110", "A"},
				    {"010_10110", "B"},
				},
				ExpShift: 3, ExpPfxSize: 0, ExpNibSize: 5, ExpPrefix: 0,
				ExpBitmap: (uint64(1) << 0b01101) | (uint64(1) << 0b0011),
				Queries: []*Query{
				},
			},
		*/
		// TODO: add a test case where the second key is smaller than a nib and the first key
		//       continues with zeros
		/*
		 */
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
				actPrefix, actPfxSize = node.FanPrefix()
				actBitmap, _          = node.FanBitmap()
			)

			require.Equal(t, tcase.ExpShift, node.Shift())
			require.Equal(t, tcase.ExpNibSize, node.FanNibbleSize())
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
