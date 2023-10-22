package qptrie

import (
	"math"
	"testing"
)

func BenchmarkReplaceBits_SHL_SUB_XOR_AND_XOR(b *testing.B) {
	var (
		bmpSize        = 17 // bits
		bitpack uint64 = 0b0_0_101_100_010000_1100101001110001_00000000000000000_00000000000000000
		bitmap1 uint64 = 0b10000101000101101
		bitmap2 uint64 = 0b01100011101100010
		result  uint64
	)
	bitpack |= bitmap1

	b.ResetTimer()

	for i := 0; i < b.N*100; i++ {
		mask := uint64(1)<<bmpSize - 1
		result = bitpack ^ ((bitpack ^ bitmap2) & mask)
	}

	b.StopTimer()

	if result&(1<<bmpSize-1) != bitmap2 {
		panic("doesn't match bitmap2")
	}
}

func BenchmarkReplaceBits_SHL_AND_OR(b *testing.B) {
	var (
		bmpSize        = 17 // bits
		bitpack uint64 = 0b0_0_101_100_010000_1100101001110001_00000000000000000_00000000000000000
		bitmap1 uint64 = 0b10000101000101101
		bitmap2 uint64 = 0b01100011101100010
		result  uint64
	)
	bitpack |= bitmap1

	b.ResetTimer()

	for i := 0; i < b.N*100; i++ {
		mask := uint64(math.MaxUint64) << bmpSize
		result = (bitpack & mask) | bitmap2
	}

	b.StopTimer()

	if result&(1<<bmpSize-1) != bitmap2 {
		panic("doesn't match bitmap2")
	}
}

func BenchmarkReplaceBits_SHL_SUB_AND_XOR_OR(b *testing.B) {
	var (
		bmpSize        = 17 // bits
		bitpack uint64 = 0b0_0_101_100_010000_1100101001110001_00000000000000000_00000000000000000
		bitmap1 uint64 = 0b10000101000101101
		bitmap2 uint64 = 0b01100011101100010
		result  uint64
	)
	bitpack |= bitmap1

	b.ResetTimer()

	for i := 0; i < b.N*100; i++ {
		mask := uint64(1)<<bmpSize - 1
		result = (bitpack ^ (bitpack & mask)) | bitmap2
	}

	b.StopTimer()

	if result&(1<<bmpSize-1) != bitmap2 {
		panic("doesn't match bitmap2")
	}
}
