package qptrie

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	qp := New()

	require.NotNil(t, qp)
	assert.Equal(t,
		strconv.FormatUint(0b0_0_000_101_00000_000000000000000000_000000000000000000000000000000000, 2),
		strconv.FormatUint(qp.bitpack, 2),
	)
	assert.Equal(t, unsetPtr, qp.pointer)
}

func TestGet(t *testing.T) {
	t.Parallel()

	const (
		emptyKey        = ""
		embeddedKey     = "key-emb" // max embedded key size is 7 bytes
		upperCaseEmbKey = "KEY-EMB"
		longerEmbKey    = "key-emb\x00"
		regularKey      = "key-regular"
		longerKey       = "key-regular\x00"
		upperCaseKey    = "KEY-REGULAR"
		unknownKey      = "key-unknown"
		zeroKey         = "\x00"
		value           = 123
	)

	var (
		emptyFan     = newFanNode(0, 5, 0, 0)
		regularFan   = newFanNode(0, 4, 0, 0)
		prefixFan    = newFanNode(0, 3, 4*byteWidth, stringToUint64("key-"))
		emptyLeaf    = newLeaf(emptyKey, 0, value)
		embeddedLeaf = newLeaf(embeddedKey, 0, value)
		regularLeaf  = newLeaf(regularKey, 0, value)
	)

	addToFanNode(regularFan, emptyKey, value, false)
	addToFanNode(regularFan, regularKey, value, false)

	addToFanNode(prefixFan, regularKey, value, false)

	for _, tcase := range []*struct {
		Name   string
		Root   *Twig
		Key    string
		ExpVal any
		ExpOK  bool
	}{
		{"empty fan, empty key", emptyFan, emptyKey, nil, false},
		{"empty fan, zero key", emptyFan, zeroKey, nil, false},
		{"empty fan, unknown key", emptyFan, unknownKey, nil, false},
		{"regular fan, empty key", regularFan, emptyKey, value, true},
		{"regular fan, zero key", regularFan, zeroKey, nil, false},
		{"regular fan, unknown key", regularFan, unknownKey, nil, false},
		{"regular fan, regular key", regularFan, regularKey, value, true},
		{"regular fan, upper-case key", regularFan, upperCaseKey, nil, false},
		{"regular fan, longer key", regularFan, longerKey, nil, false},
		{"prefix fan, empty key", prefixFan, emptyKey, nil, false},
		{"prefix fan, zero key", prefixFan, zeroKey, nil, false},
		{"prefix fan, unknown key", prefixFan, unknownKey, nil, false},
		{"prefix fan, regular key", prefixFan, regularKey, value, true},
		{"prefix fan, upper-case key", prefixFan, upperCaseKey, nil, false},
		{"prefix fan, longer key", prefixFan, longerKey, nil, false},
		{"empty leaf, empty key", emptyLeaf, emptyKey, value, true},
		{"empty leaf, zero key", emptyLeaf, zeroKey, nil, false},
		{"empty leaf, unknown key", emptyLeaf, unknownKey, nil, false},
		{"embedded leaf, empty key", embeddedLeaf, emptyKey, nil, false},
		{"embedded leaf, zero key", embeddedLeaf, zeroKey, nil, false},
		{"embedded leaf, embedded key", embeddedLeaf, embeddedKey, value, true},
		{"embedded leaf, upper-case embedded key", embeddedLeaf, upperCaseEmbKey, nil, false},
		{"embedded leaf, longer embedded key", embeddedLeaf, longerEmbKey, nil, false},
		{"embedded leaf, unknown key", embeddedLeaf, unknownKey, nil, false},
		{"regular leaf, empty key", regularLeaf, emptyKey, nil, false},
		{"regular leaf, zero key", regularLeaf, zeroKey, nil, false},
		{"regular leaf, regular key", regularLeaf, regularKey, value, true},
		{"regular leaf, upper-case regular key", regularLeaf, upperCaseKey, nil, false},
		{"regular leaf, longer key", regularLeaf, longerKey, nil, false},
		{"regular leaf, unknown key", regularLeaf, unknownKey, nil, false},
	} {
		tcase := tcase

		t.Run(tcase.Name, func(t *testing.T) {
			val, ok := tcase.Root.Get(tcase.Key)

			assert.Equal(t, tcase.ExpVal, val)
			assert.Equal(t, tcase.ExpOK, ok)
		})
	}
}

func TestSet_Get(t *testing.T) {
	t.Parallel()

	var (
		qp    = New()
		state = map[string]any{}
	)

	for _, tcase := range []*struct {
		Key string
		Val any
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
		{"Kickstarter distillery lomo mlkshk echo.", 14},
		{"Kogi biodiesel dreamcatcher mumblecore irony.", 15},
		{"+1 selvage selfies whatever Godard.", 16},
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
		total       = 1_00
		seed        = 1234567890
		wordsPerKey = 5
	)

	var (
		qp    = New()
		state = map[string]any{}
		fake  = gofakeit.New(seed)
		ready bool
	)

	// Set fake data
	for i := 0; i < total; i++ {
		var (
			key = fake.HipsterSentence(wordsPerKey)
			val = fake.Name()
		)

		qp.Set(key, val)
		state[key] = val

		if key == "Kogi biodiesel dreamcatcher mumblecore irony." {
			ready = true
		}
		if ready {
			_, ok := qp.Get("Kogi biodiesel dreamcatcher mumblecore irony.")

			if !ok {
				t.Logf(">>> %v", key)
				t.FailNow()
			}
		}
	}

	// Get all the keys we set
	for key, val := range state {
		actual, ok := qp.Get(key)

		assert.Equal(t, val, actual, key)
		assert.True(t, ok)
	}
}
