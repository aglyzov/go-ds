package qptrie

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testEmbedKey(t *testing.T, fn func(string) uint64) {
	for _, tcase := range []*struct {
		Key     string
		Bitpack uint64
	}{
		//                                                                   E nsz ksz ---------------------- embedded key --------------------------
		{"                                                      00100001", 0b1_000_001_00000000_00000000_00000000_00000000_00000000_00000000_00100001},
		{"                                             11010000_10100011", 0b1_000_010_00000000_00000000_00000000_00000000_00000000_11010000_10100011},
		{"                                    00010100_11010000_10100011", 0b1_000_011_00000000_00000000_00000000_00000000_00010100_11010000_10100011},
		{"                           01000110_00010100_11010000_10100011", 0b1_000_100_00000000_00000000_00000000_01000110_00010100_11010000_10100011},
		{"                  10101010_01000110_00010100_11010000_10100011", 0b1_000_101_00000000_00000000_10101010_01000110_00010100_11010000_10100011},
		{"         11100110_10101010_01000110_00010100_11010000_10100011", 0b1_000_110_00000000_11100110_10101010_01000110_00010100_11010000_10100011},
		{"00011000_11100110_10101010_01000110_00010100_11010000_10100011", 0b1_000_111_00011000_11100110_10101010_01000110_00010100_11010000_10100011},
	} {
		var (
			tcase  = tcase
			bitKey = strings.TrimSpace(tcase.Key)
			name   = bitKey
		)

		t.Run(name, func(t *testing.T) {
			key, err := bitStringToString(reverseString(bitKey))
			require.NoError(t, err)

			bitpack := fn(key)

			assert.Equal(t, tcase.Bitpack, bitpack, uint64ToBitString(bitpack))
		})
	}
}

func testExtractKey(t *testing.T, fn func(uint64) string) {
	for _, tcase := range []*struct {
		Bitpack uint64
		ExpKey  string
	}{
		// L E nsz ksz ---------------------- embedded key --------------------------
		{0b1_1_000_000_00000000_00000000_00000000_00000000_00000000_00000000_00000000, "                                                              "},
		{0b1_1_000_001_00000000_00000000_00000000_00000000_00000000_00000000_00100001, "                                                      00100001"},
		{0b1_1_000_010_00000000_00000000_00000000_00000000_00000000_11010000_10100011, "                                             11010000_10100011"},
		{0b1_1_000_011_00000000_00000000_00000000_00000000_00010100_11010000_10100011, "                                    00010100_11010000_10100011"},
		{0b1_1_000_100_00000000_00000000_00000000_01000110_00010100_11010000_10100011, "                           01000110_00010100_11010000_10100011"},
		{0b1_1_000_101_00000000_00000000_10101010_01000110_00010100_11010000_10100011, "                  10101010_01000110_00010100_11010000_10100011"},
		{0b1_1_000_110_00000000_11100110_10101010_01000110_00010100_11010000_10100011, "         11100110_10101010_01000110_00010100_11010000_10100011"},
		{0b1_1_000_111_00011000_11100110_10101010_01000110_00010100_11010000_10100011, "00011000_11100110_10101010_01000110_00010100_11010000_10100011"},
	} {
		var (
			tcase = tcase
			name  = fmt.Sprintf("%0b", tcase.Bitpack)
		)

		t.Run(name, func(t *testing.T) {
			key := fn(tcase.Bitpack)

			assert.Equal(t, strings.TrimSpace(tcase.ExpKey), reverseString(stringToBitString(key)))
		})
	}
}

func TestEmbedKey(t *testing.T) {
	t.Parallel()

	testEmbedKey(t, embedKey)
}

func TestEmbedKeySlow(t *testing.T) {
	t.Parallel()

	testEmbedKey(t, embedKeySlow)
}

func TestExtractKey(t *testing.T) {
	t.Parallel()

	testExtractKey(t, extractKey)
}

func TestExtractKeySlow(t *testing.T) {
	t.Parallel()

	testExtractKey(t, extractKeySlow)
}
