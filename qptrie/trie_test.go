package qptrie

import (
	"fmt"
	"math/bits"
	"strconv"
	"strings"
	"testing"

	"github.com/brianvoe/gofakeit/v6"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	qp := New()

	assert.NotNil(t, qp)
}

func TestGet(t *testing.T) {
	t.Parallel()

	qp := New(KV{"abc", 123})

	for _, tcase := range []*struct {
		Key    string
		ExpVal interface{}
		ExpOK  bool
	}{
		{"", nil, false},
		{"\x00", nil, false},
		{"\x00\x00\x00", nil, false},
		{"unknown", nil, false},
		{"abc", 123, true},
		{"ABC", nil, false},
		{"ab", nil, false},
		{"abc.", nil, false},
		{"abc\x00", nil, false},
	} {
		var (
			tcase = tcase
			name  = fmt.Sprintf("%#v", tcase.Key)
		)

		t.Run(name, func(t *testing.T) {
			val, ok := qp.Get(tcase.Key)

			assert.Equal(t, tcase.ExpVal, val)
			assert.Equal(t, tcase.ExpOK, ok)
		})
	}
}

func TestSet_IsLeaf(t *testing.T) {
	t.Parallel()

	qp := New()

	assert.False(t, qp.isLeaf())

	qp.Set("abc", 123) // add a key-value pair

	assert.True(t, qp.isLeaf())

	qp.Set("abc", 345) // replace the value

	assert.True(t, qp.isLeaf())

	qp.Set("edf", 567) // add a key-value pair

	assert.False(t, qp.isLeaf())
}

func TestSet_Get(t *testing.T) {
	t.Parallel()

	var (
		qp    = New()
		state = map[string]interface{}{}
	)

	for _, tcase := range []*struct {
		Key string
		Val interface{}
	}{
		{"", 1},
		{"\x00", 2},
		{"\x00\x00\x00", 3},
		{"abcde", 4},
		{"abcdE", 5},
		{"ab", 6},
		{"abcde", 7}, // replace
		{"abcde\x00", 8},
		{"", 9}, // replace
		{"Абвгд", 10},
		{"Абвгдеё", 11},
		{"Banjo lo-fi brooklyn mlkshk cliche.", 12},
		{"Banjo lomo DIY whatever street.", 13},
	} {
		var (
			tcase = tcase
			name  = fmt.Sprintf("%#v,%#v", tcase.Key, tcase.Val)
		)

		t.Run(name, func(t *testing.T) {
			qp.Set(tcase.Key, tcase.Val)
			state[tcase.Key] = tcase.Val

			// Get all the keys we set so far
			for key, val := range state {
				actual, ok := qp.Get(key)

				assert.Equal(t, val, actual, key)
				assert.True(t, ok)
			}
		})
	}
}

func TestSet_FakeData(t *testing.T) {
	t.Parallel()

	const (
		total       = 1_000_000
		seed        = 1234567890
		wordsPerKey = 5
	)

	var (
		qp    = New()
		state = map[string]interface{}{}
		fake  = gofakeit.New(seed)
	)

	// Set fake data
	for i := 0; i < total; i++ {
		var (
			key = fake.HipsterSentence(wordsPerKey)
			val = fake.Name()
		)

		qp.Set(key, val)
		state[key] = val
	}

	// Get all the keys we set
	for key, val := range state {
		actual, ok := qp.Get(key)

		assert.Equal(t, val, actual, key)
		assert.True(t, ok)
	}
}

func TestGetNibble(t *testing.T) {
	t.Parallel()

	for _, tcase := range []*struct {
		BitKey   string
		Shift    int
		ExpNib   byte
		ExpKey   string
		ExpShift int
	}{
		{"", 0, bitmapWidth - 1, "", 0},
		{"", 7, bitmapWidth - 1, "", 0},
		{"01010101", 0, 0b01010, "01010101", 5},
		{"01010101", 1, 0b10101, "01010101", 6},
		{"01010101", 2, 0b01010, "01010101", 7},
		{"01010101", 3, 0b10101, "", 0},
		{"01010101", 4, 0b01010, "", 1},
		{"01010101", 5, 0b00101, "", 2},
		{"01010101", 6, 0b00010, "", 3},
		{"01010101", 7, 0b00001, "", 4},
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
			nib, key, shift := getNibble(key, tcase.Shift)

			bitKey := stringToBitString(key)

			assert.Equal(t, tcase.ExpNib, nib, stringToBitString(string([]byte{nib})))
			assert.Equal(t, tcase.ExpKey, bitKey)
			assert.Equal(t, tcase.ExpShift, shift)
		})
	}
}

func bitStringToString(bitStr string) (string, error) {
	bitStr = strings.Replace(bitStr, "_", "", -1)

	var buf strings.Builder

	for tail := bitStr; tail != ""; tail = tail[byteWidth:] {
		b, err := strconv.ParseInt(tail[:byteWidth], 2, 32)
		if err != nil {
			return "", err
		}

		buf.WriteByte(bits.Reverse8(byte(b)))
	}

	return buf.String(), nil
}

func stringToBitString(str string) string {
	var buf strings.Builder

	for i := 0; i < len(str); i++ {
		b := bits.Reverse8(str[i])
		buf.WriteString(fmt.Sprintf("%08b", b))
		if i != len(str)-1 {
			buf.WriteByte('_')
		}
	}

	return buf.String()
}
