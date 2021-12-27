package qptrie

import (
	"fmt"
	"math/bits"
	"strconv"
	"strings"
)

func bitStringToString(bitStr string) (string, error) {
	bitStr = strings.Replace(bitStr, "_", "", -1)

	var buf strings.Builder

	for tail := bitStr; tail != ""; tail = tail[byteWidth:] {
		b, err := strconv.ParseInt(tail[:byteWidth], 2, 32)
		if err != nil {
			return "", err
		}

		buf.WriteByte(bits.Reverse8(byte(b)))
	}

	return buf.String(), nil
}

func stringToBitString(str string) string {
	var buf strings.Builder

	for i := 0; i < len(str); i++ {
		b := bits.Reverse8(str[i])
		buf.WriteString(fmt.Sprintf("%08b", b))
		if i != len(str)-1 {
			buf.WriteByte('_')
		}
	}

	return buf.String()
}

func uint64ToBitString(val uint64) string {
	var buf strings.Builder

	for i := 0; i < 64; i += byteWidth {
		b := byte(val >> i)
		b = bits.Reverse8(b)
		buf.WriteString(fmt.Sprintf("%08b", b))
		buf.WriteByte('_')
	}

	return strings.TrimRight(buf.String(), "0_")
}
