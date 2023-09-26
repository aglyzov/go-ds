package qptrie

import (
	"testing"
)

func getKeysToEmbed() []string {
	return []string{"", "A", "ab", "ABC", "1234", "abcde", "AbCdEf", "ABC1234"}
}

func getKeysToExtract() []uint64 {
	var (
		keys     = getKeysToEmbed()
		embedded = make([]uint64, len(keys))
	)
	for i, key := range keys {
		embedded[i] = embedKey(key)
	}
	return embedded
}

func BenchmarkEmbedKey(b *testing.B) {
	var (
		keys = getKeysToEmbed()
		num  = len(keys)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := keys[i%num]
		_ = embedKey(key)
	}
}

func BenchmarkEmbedKeySlow(b *testing.B) {
	var (
		keys = getKeysToEmbed()
		num  = len(keys)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := keys[i%num]
		_ = embedKeySlow(key)
	}
}

func BenchmarkExtractKey(b *testing.B) {
	var (
		keys = getKeysToExtract()
		num  = len(keys)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := keys[i%num]
		_ = extractKey(key)
	}
}

func BenchmarkExtractKeySlow(b *testing.B) {
	var (
		keys = getKeysToExtract()
		num  = len(keys)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := keys[i%num]
		_ = extractKeySlow(key)
	}
}
