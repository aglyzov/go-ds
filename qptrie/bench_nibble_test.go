package qptrie

import (
	"testing"
)

func getKeysToTakeBits() []string {
	return []string{"", "a", "12", "ABC", "12345", "ABC1234", "ABCDEFGHI"}
}

func BenchmarkTakeNBits(b *testing.B) {
	var (
		keys    = getKeysToTakeBits()
		numKeys = len(keys)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := keys[i%numKeys]
		_, _, _ = takeNBits(key, i&0b111, i&0b111111) // &0b111 == %8, &0b111111 == %64
	}
}

func BenchmarkTakeNBitsSwitch(b *testing.B) {
	var (
		keys    = getKeysToTakeBits()
		numKeys = len(keys)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := keys[i%numKeys]
		switch num := i & 0b111111; num { // i&0b111111 == i%64
		case 4:
			_, _, _ = take4Bits(key, i&0b111)
		case 5:
			_, _, _ = take5Bits(key, i&0b111)
		default:
			_, _, _ = takeNBits(key, i&0b111, num) // i&0b111 == i%8
		}
	}
}

func BenchmarkTakeNBits32(b *testing.B) {
	var (
		keys    = getKeysToTakeBits()
		numKeys = len(keys)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := keys[i%numKeys]
		_, _, _ = takeNBits(key, i&0b111, 32) // &0b111 == %8
	}
}

func BenchmarkTakeNBits56(b *testing.B) {
	var (
		keys    = getKeysToTakeBits()
		numKeys = len(keys)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := keys[i%numKeys]
		_, _, _ = takeNBits(key, i&0b111, 56) // &0b111 == %8
	}
}

func BenchmarkTakeNbits5(b *testing.B) {
	var (
		keys    = getKeysToTakeBits()
		numKeys = len(keys)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := keys[i%numKeys]
		_, _, _ = takeNBits(key, i&0b111, 5) // i&0b111 == i%8
	}
}

func BenchmarkTake4Bits(b *testing.B) {
	var (
		keys    = getKeysToTakeBits()
		numKeys = len(keys)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := keys[i%numKeys]
		_, _, _ = take4Bits(key, i&0b111) // i&0b111 == i%8
	}
}

func BenchmarkTake5Bits(b *testing.B) {
	var (
		keys    = getKeysToTakeBits()
		numKeys = len(keys)
	)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		key := keys[i%numKeys]
		_, _, _ = take5Bits(key, i&0b111) // i&0b111 == i%8
	}
}
