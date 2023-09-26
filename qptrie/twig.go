package qptrie

import (
	"math/bits"
	"strings"
	"unsafe"
)

const (
	// bit fields
	leafBitOffset    = 63 // 1-bit flag:   1 - leaf, 0 - node
	embKeyBitOffset  = 62 // 1-bit flag:   key tail  1 - embedded, 0 - stored in pointer (*KV)
	cutBitOffset     = 62 // 1-bit flag:   1 - cut-node, 0 - fan-node
	nibShiftOffset   = 59 // 3-bit number: current nibble's bit offset in to a byte
	embKeySizeOffset = 56 // 3-bit number: number of embedded key bytes [1..7]
	nibSizeOffset    = 56 // 3-bit number: size of a nibble in bits [1..5]
	pfxSizeOffset    = 51 // 5-bit number: size of a stored prefix in bits [0..31]

	nibShiftWidth   = 3
	embKeySizeWidth = 3
	nibSizeWidth    = 3
	pfxSizeWidth    = 5
	bitmapWidthMax  = 33

	nibShiftMax   = (1 << nibShiftWidth) - 1   // max nibble shift 0b_111 == 7
	embKeySizeMax = (1 << embKeySizeWidth) - 1 // max amount of bytes to be embedded 0b_111 == 7
	nibSizeMax    = 5                          // largest nibble size in bits
	pfxSizeMax    = (1 << pfxSizeWidth) - 1    // largest prefix size in bits 0b_11111 == 31

	leafBitMask    uint64 = 1 << leafBitOffset                         // 0b_10000000000000..0
	embKeyBitMask  uint64 = 1 << embKeyBitOffset                       // 0b_01000000000000..0
	cutBitMask     uint64 = 1 << cutBitOffset                          // 0b_01000000000000..0
	nibShiftMask   uint64 = nibShiftMax << nibShiftOffset              // 0b_00111000000000..0
	embKeySizeMask uint64 = embKeySizeMax << embKeySizeOffset          // 0b_00000111000000..0
	nibSizeMask    uint64 = ((1 << nibSizeWidth) - 1) << nibSizeOffset // 0b_00000111000000..0
	pfxSizeMask    uint64 = pfxSizeMax << pfxSizeOffset                // 0b_00000000111110..0
)

var unsetPtr = unsafe.Pointer(new(struct{}))

type KV struct {
	Key string
	Val interface{}
}

// Twig is a uniform element of a QP-Trie (meaning either a node or a leaf).
type Twig struct {
	bitpack uint64
	pointer unsafe.Pointer // should always point at something allocated!
}

func New(init ...KV) *Twig {
	qp := newFanNode(0, nibSizeMax, 0, 0)

	for _, kv := range init {
		qp.Set(kv.Key, kv.Val)
	}

	return qp
}

func (qp *Twig) Get(key string) (interface{}, bool) {
	if closest, _, ok := qp.findClosest(key); ok {
		return getLeafKV(closest).Val, true
	}

	return nil, false
}

func (qp *Twig) Set(key string, val interface{}) (interface{}, bool) {
	closest, key, ok := qp.findClosest(key)
	if ok {
		// matched exactly - replace the value
		return setLeafValue(closest, val), true
	}

	if closest.bitpack&leafBitMask == 0 {
		// it's a node
		switch {
		case closest.bitpack&cutBitMask == 0:
			// it's a fan-node
			addToFanNode(closest, key, val)
		default:
			// it's a cut-node
			addToCutNode(closest, key, val)
		}
	} else {
		// it's a leaf
		addToLeaf(closest, key, val)
	}

	return nil, false
}

func (qp *Twig) findClosest(key string) (*Twig, string, bool) {
	// walk along a common prefix
	var (
		nib   byte // current nibble
		cur   = qp // current twig
		shift = int((cur.bitpack & nibShiftMask) >> nibShiftOffset)
	)

	// while it's a node
	for cur.bitpack&leafBitMask == 0 {
		if cur.bitpack&cutBitMask != 0 {
			//
			// -- cur is a cut-node --
			//
			cut := getCutNodeKey(cur)

			if !strings.HasPrefix(key, cut) {
				// the cut doesn't match
				return cur, key, false
			}

			// jump to the next twig
			cur = getCutNodeTwig(cur)
			key = key[len(cut):]
			shift = 0 // reset

			continue
		}

		// -- cur is a fan-node --
		var (
			bitpack     = cur.bitpack
			nibSize     = int(bitpack & nibSizeMask >> nibSizeOffset)
			bitmapWidth = (uint64(1) << nibSize) + 1 // one extra bit to encode an empty key
			bitmapMask  = (uint64(1) << bitmapWidth) - 1
			bitmap      = bitpack & bitmapMask
		)

		if bitmap == 0 {
			// empty node
			return cur, key, false
		}

		var (
			nib64   uint64
			prevKey = key
			pfxSize = int(bitpack & pfxSizeMask >> pfxSizeOffset)
		)

		if pfxSize > 0 {
			// the fan-node has a prefix - need to check if it matches the key
			var (
				pfxOffset = pfxSizeOffset - pfxSize
				pfxMask   = (uint64(1) << pfxSize) - 1
				pfx       = (bitpack >> pfxOffset) & pfxMask
			)

			nib64, key, shift = takeNbits(key, shift, pfxSize)

			if pfx != nib64 {
				// the prefix doesn't match the key
				return cur, prevKey, false
			}
		}

		nib64, key, shift = takeNbits(key, shift, nibSize)
		nib = byte(nib64 & 0xFF)

		var (
			mask = uint64(1) << nib
			idx  = bits.OnesCount64(bitmap & (mask - 1))
		)

		if bitmap&mask == 0 {
			// the node doesn't have the nibble
			return cur, prevKey, false
		}

		cur = &(*[bitmapWidthMax]Twig)(cur.pointer)[idx]
	}

	// -- cur is a leaf --

	return cur, key, key == getLeafKey(cur)
}
