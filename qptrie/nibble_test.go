package qptrie

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTakeNbits(t *testing.T) {
	t.Parallel()

	const noValue = ^uint64(0)

	for _, tcase := range []*struct {
		BitKey   string
		Shift    int
		Size     int
		ExpNib   uint64
		ExpKey   string
		ExpShift int
	}{
		{"", 0, 4, 0b10000, "", 0},
		{"", 0, 8, 0b100000000, "", 0},
		{"", 0, 13, 0b10000000000000, "", 0},
		{"", 7, 4, 0b10000, "", 0},
		{"", 7, 8, 0b100000000, "", 0},
		{"", 7, 13, 0b10000000000000, "", 0},
		{"01010101", 0, 5, 0b01010, "01010101", 5},
		{"01010101", 1, 5, 0b10101, "01010101", 6},
		{"01010101", 2, 5, 0b01010, "01010101", 7},
		{"01010101", 3, 5, 0b10101, "", 0},
		{"01010101", 4, 5, 0b01010, "", 0},
		{"01010101", 5, 5, 0b00101, "", 0},
		{"01010101", 6, 5, 0b00010, "", 0},
		{"01010101", 7, 5, 0b00001, "", 0},
		{"01010101", 3, 12, 0b000000010101, "", 0},
		{"01010101", 3, 0, 0b00000, "01010101", 3},
		{"01010101_11001100", 3, 5, 0b10101, "11001100", 0},
		{"01010101_11001100", 4, 5, 0b11010, "11001100", 1},
		{"01010101_11001100", 5, 5, 0b11101, "11001100", 2},
		{"01010101_11001100", 6, 5, 0b01110, "11001100", 3},
		{"01010101_11001100", 7, 5, 0b00111, "11001100", 4},
		{"01010101_11001100", 7, 8, 0b01100111, "11001100", 7},
		{"01010101_11001100", 7, 10, 0b0001100111, "", 0},
		{"01010101_11001100_10101010", 7, 1, 0b1, "11001100_10101010", 0},
		{"01010101_11001100_10101010", 7, 5, 0b00111, "11001100_10101010", 4},
		{"01010101_11001100_10101010", 7, 10, 0b1001100111, "10101010", 1},
	} {
		var (
			tcase = tcase
			name  = fmt.Sprintf("%#v,%#v,%#v", tcase.BitKey, tcase.Shift, tcase.Size)
		)
		key, err := bitStringToString(tcase.BitKey)
		require.NoError(t, err)

		t.Run(name, func(t *testing.T) {
			nib, key, shift := takeNbits(key, tcase.Shift, tcase.Size)

			bitKey := stringToBitString(key)

			assert.Equal(t, tcase.ExpNib, nib, uint64ToBitString(nib))
			assert.Equal(t, tcase.ExpKey, bitKey)
			assert.Equal(t, tcase.ExpShift, shift)
		})
	}
}

func TestTake5bits(t *testing.T) {
	t.Parallel()

	const noValue = ^byte(0)

	for _, tcase := range []*struct {
		BitKey   string
		Shift    int
		ExpNib   byte
		ExpKey   string
		ExpShift int
	}{
		{"", 0, 0b100000, "", 0},
		{"", 7, 0b100000, "", 0},
		{"01010101", 0, 0b01010, "01010101", 5},
		{"01010101", 1, 0b10101, "01010101", 6},
		{"01010101", 2, 0b01010, "01010101", 7},
		{"01010101", 3, 0b10101, "", 0},
		{"01010101", 4, 0b01010, "", 0},
		{"01010101", 5, 0b00101, "", 0},
		{"01010101", 6, 0b00010, "", 0},
		{"01010101", 7, 0b00001, "", 0},
		{"01010101_11001100", 3, 0b10101, "11001100", 0},
		{"01010101_11001100", 4, 0b11010, "11001100", 1},
		{"01010101_11001100", 5, 0b11101, "11001100", 2},
		{"01010101_11001100", 6, 0b01110, "11001100", 3},
		{"01010101_11001100", 7, 0b00111, "11001100", 4},
		{"01010101_11001100_10101010", 7, 0b00111, "11001100_10101010", 4},
	} {
		var (
			tcase = tcase
			name  = fmt.Sprintf("%#v,%#v", tcase.BitKey, tcase.Shift)
		)
		key, err := bitStringToString(tcase.BitKey)
		require.NoError(t, err)

		t.Run(name, func(t *testing.T) {
			nib, key, shift := take5bits(key, tcase.Shift)

			bitKey := stringToBitString(key)

			assert.Equal(t, tcase.ExpNib, nib, stringToBitString(string([]byte{nib})))
			assert.Equal(t, tcase.ExpKey, bitKey)
			assert.Equal(t, tcase.ExpShift, shift)
		})
	}
}
