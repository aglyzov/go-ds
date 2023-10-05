package qptrie

import (
	"fmt"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/stretchr/testify/assert"
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
		ExpVal any
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
