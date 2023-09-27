package qptrie

const (
	byteShift   = 3
	byteWidth   = 1 << byteShift // 0b1000
	byteModMask = byteWidth - 1  // 0b0111
)

// takeNBits takes `num` bits [0..63] from a string skipping the first `skip`
// bits [0..7].
//
// Returns three values: <taken-bits:uint64>, "string-remainder", <new-shift:int>
func takeNBits(str string, skip, num int) (uint64, string, int) {
	// fast path
	switch num {
	case 4:
		nib, str, shift := take4Bits(str, skip)
		return uint64(nib), str, shift
	case 5:
		nib, str, shift := take5Bits(str, skip)
		return uint64(nib), str, shift
	}

	strLen := len(str)

	if strLen == 0 {
		return uint64(1) << num, str, 0
	}

	var (
		mask     = (uint64(1) << num) - 1
		result   = uint64(str[0] >> skip)
		strBits  = strLen<<byteShift - skip
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

	return result & mask, str[offset>>byteShift:], offset & byteModMask
}

// take5Bits takes 5 bits from a string skipping the first `skip` bits [0..7].
//
// Returns three values: <taken-bits:byte>, "string-remainder", <new-shift:int>
func take5Bits(str string, skip int) (byte, string, int) {
	const (
		bits    = 5
		mask    = 0b_011111 // 31
		noValue = 0b_100000 // 32
	)

	strLen := len(str)

	if strLen == 0 {
		return noValue, str, 0
	}

	var (
		nshift = (skip + bits) & byteModMask // % byteWidth
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

// take4Bits takes 4 bits from a string skipping the first `skip` bits [0..7].
//
// Returns three values: <taken-bits:byte>, "string-remainder", <new-shift:int>
func take4Bits(str string, skip int) (byte, string, int) {
	const (
		bits    = 4
		mask    = 0b_01111 // 15
		noValue = 0b_10000 // 16
	)

	strLen := len(str)

	if strLen == 0 {
		return noValue, str, 0
	}

	var (
		nshift = (skip + bits) & byteModMask // % byteWidth
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
