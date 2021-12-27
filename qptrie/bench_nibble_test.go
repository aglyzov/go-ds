package qptrie

import (
	"testing"
)

func BenchmarkTakeNbits(b *testing.B) {
	var (
		keys    = []string{"", "a", "ab", "abc", "abcde"}
		numKeys = len(keys)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := keys[i%numKeys]
		_, _, _ = takeNbits(key, i%8, 5)
	}
}

func BenchmarkTake5bits(b *testing.B) {
	const noValue = ^byte(0)

	var (
		keys    = []string{"", "a", "ab", "abc", "abcde"}
		numKeys = len(keys)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := keys[i%numKeys]
		_, _, _ = take5bits(key, i%8)
	}
}
