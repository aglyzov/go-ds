package qptrie

const byteWidth = 8

// takeNbits takes `num` bits [0..63] from a string skipping the first `skip`
// bits [0..7].
//
// Returns three values: <taken-bits:uint64>, "string-remainder", <new-shift:int>
//
func takeNbits(str string, skip, num int) (uint64, string, int) {
	var strLen = len(str)

	if strLen == 0 {
		return uint64(1) << num, str, 0
	}

	if num == 5 {
		// take a fast path - take5bits works almost 3 times faster
		nib, str, shift := take5bits(str, skip)

		return uint64(nib), str, shift
	}

	var (
		mask     = (uint64(1) << num) - 1
		result   = uint64(str[0] >> skip)
		strBits  = strLen*byteWidth - skip
		doneBits = byteWidth - skip
		needBits = num
	)

	if needBits > strBits {
		needBits = strBits
	}

	for i := 1; doneBits < needBits; i++ {
		result |= uint64(str[i]) << doneBits
		doneBits += byteWidth
	}

	offset := skip + needBits

	return result & mask, str[offset/byteWidth:], offset % byteWidth
}

// take5bits takes 5 bits from a string skipping the first `skip` bits [0..7].
//
// Returns three values: <taken-bits:byte>, "string-remainder", <new-shift:int>
//
func take5bits(str string, skip int) (byte, string, int) {
	const (
		bits    = 5
		mask    = 0b_011111 // 31
		noValue = 0b_100000 // 32
	)

	var strLen = len(str)

	if strLen == 0 {
		return noValue, str, 0
	}

	var (
		nshift = (skip + bits) % byteWidth
		nib    = str[0] >> skip
	)

	switch {
	case nshift > skip:
		return nib & mask, str, nshift

	case strLen == 1:
		return nib, "", 0

	default:
		nib |= str[1] << (byteWidth - skip)

		return nib & mask, str[1:], nshift
	}
}
