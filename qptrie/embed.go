package qptrie

// embedKey embeds a short key into a bitpack.
func embedKey(key string) uint64 {
	size := byte(len(key))

	if size > embKeySizeMax {
		size = embKeySizeMax
	}

	// NOTE: we rely on embKeyBitMask == cutBitMask here because
	//       embedKey() is used in both newLeaf() and newCutNode()
	//
	bitpack := embKeyBitMask | uint64(size)<<embKeySizeOffset

	for i := byte(0); i < size; i++ {
		bitpack |= uint64(key[i]) << (byteWidth * i)
	}

	return bitpack
}

// extractKey extracts an embedded key from a bitpack.
func extractKey(bitpack uint64) string {
	var (
		size = byte(bitpack & embKeySizeMask >> embKeySizeOffset)
		data [embKeySizeMax]byte
	)

	for i := byte(0); i < size; i++ {
		data[i] = byte(bitpack >> (byteWidth * i))
	}

	return string(data[:size])
}
