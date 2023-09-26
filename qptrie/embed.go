package qptrie

import (
	"unsafe"
)

// embedKey embeds a short key into a bitpack.
func embedKey(key string) uint64 {
	size := min(len(key), embKeySizeMax)

	// NOTE: we rely on embKeyBitMask == cutBitMask here because
	//       embedKey() is used in both newLeaf() and newCutNode()

	bitpack := embKeyBitMask | uint64(size)<<embKeySizeOffset

	switch size {
	case 7:
		bitpack |= uint64(key[6]) << (byteWidth * 6)
		fallthrough
	case 6:
		ptr := unsafe.Pointer(unsafe.StringData(key))
		bitpack |= uint64(*(*uint32)(ptr))
		ptr = unsafe.Add(ptr, 4)
		bitpack |= uint64(*(*uint16)(ptr)) << (byteWidth * 4)
	case 5:
		bitpack |= uint64(key[4]) << (byteWidth * 4)
		fallthrough
	case 4:
		ptr := unsafe.Pointer(unsafe.StringData(key))
		bitpack |= uint64(*(*uint32)(ptr))
	case 3:
		bitpack |= uint64(key[2]) << (byteWidth * 2)
		fallthrough
	case 2:
		ptr := unsafe.Pointer(unsafe.StringData(key))
		bitpack |= uint64(*(*uint16)(ptr))
	case 1:
		bitpack |= uint64(key[0])
	}

	return bitpack
}

// embedKeySlow embeds a short key into a bitpack.
func embedKeySlow(key string) uint64 {
	size := byte(len(key))

	if size > embKeySizeMax {
		size = embKeySizeMax
	}

	// NOTE: we rely on embKeyBitMask == cutBitMask here because
	//       embedKey() is used in both newLeaf() and newCutNode()
	//
	var (
		bitpack = embKeyBitMask | uint64(size)<<embKeySizeOffset
		offset  = uint64(0)
	)

	for i := byte(0); i < size; i++ {
		bitpack |= uint64(key[i]) << offset
		offset += byteWidth
	}

	return bitpack
}

// extractKey extracts an embedded key from a bitpack.
func extractKey(bitpack uint64) string {
	var (
		size = bitpack & embKeySizeMask >> embKeySizeOffset
		data = bitpack & (1<<embKeySizeOffset - 1)
		ptr  = unsafe.Pointer(&data)
	)

	return unsafe.String((*byte)(ptr), size)
}

// extractKeySlow extracts an embedded key from a bitpack.
func extractKeySlow(bitpack uint64) string {
	var (
		size   = byte(bitpack & embKeySizeMask >> embKeySizeOffset)
		data   [embKeySizeMax]byte
		offset = uint64(0)
	)

	for i := byte(0); i < size; i++ {
		data[i] = byte(bitpack >> offset)
		offset += byteWidth
	}

	return unsafe.String(&data[0], size)
}
