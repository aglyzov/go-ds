package qptrie

import (
	"testing"

	"github.com/brianvoe/gofakeit/v6"
)

func BenchmarkGoMap_Set(b *testing.B) {
	var (
		keys = getKeys(b.N)
		m    = make(map[string]interface{})
	)

	b.ResetTimer()

	for i, key := range keys {
		m[key] = i
	}
}

func BenchmarkGoMap_Get(b *testing.B) {
	var (
		keys = getKeys(b.N)
		m    = make(map[string]interface{})
	)

	for i, key := range keys {
		m[key] = i
	}

	b.ResetTimer()

	for _, key := range keys {
		_ = m[key]
	}
}

func BenchmarkQPTrie_Set(b *testing.B) {
	var (
		keys = getKeys(b.N)
		qp   = New()
	)

	b.ResetTimer()

	for i, key := range keys {
		qp.Set(key, i)
	}
}

func BenchmarkQPTrie_Get(b *testing.B) {
	var (
		keys = getKeys(b.N)
		qp   = New()
	)

	for i, key := range keys {
		qp.Set(key, i)
	}

	b.ResetTimer()

	for _, key := range keys {
		_, _ = qp.Get(key)
	}
}

func getKeys(total int) []string {
	const seed = 1234567890

	var (
		faker = gofakeit.New(seed)
		keys  = make([]string, total)
	)

	for i := range keys {
		keys[i] = faker.Sentence(4)
	}

	return keys
}
