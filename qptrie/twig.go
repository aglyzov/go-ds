package qptrie

import (
	"fmt"
	"math/bits"
	"strconv"
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
	pfxSizeOffset    = 50 // 6-bit number: size of a stored prefix in bits [0..31]

	nibShiftWidth   = 3
	embKeySizeWidth = 3
	nibSizeWidth    = 3
	pfxSizeWidth    = 6
	bitmapWidthMax  = 33

	nibShiftMax   = (1 << nibShiftWidth) - 1   // max nibble shift 0b_111 == 7
	embKeySizeMax = (1 << embKeySizeWidth) - 1 // max amount of bytes to be embedded 0b_111 == 7
	nibSizeMax    = 5                          // largest nibble size in bits
	pfxSizeMax    = 47                         // largest prefix size in bits (when nib is 1 bit)

	leafBitMask    uint64 = 1 << leafBitOffset                     // 0b_100000000000000..0
	embKeyBitMask  uint64 = 1 << embKeyBitOffset                   // 0b_010000000000000..0
	cutBitMask     uint64 = 1 << cutBitOffset                      // 0b_010000000000000..0
	nibShiftMask   uint64 = nibShiftMax << nibShiftOffset          // 0b_001110000000000..0
	embKeySizeMask uint64 = embKeySizeMax << embKeySizeOffset      // 0b_000001110000000..0
	nibSizeMask    uint64 = (1<<nibSizeWidth - 1) << nibSizeOffset // 0b_000001110000000..0
	pfxSizeMask    uint64 = (1<<pfxSizeWidth - 1) << pfxSizeOffset // 0b_000000001111110..0
)

var unsetPtr = unsafe.Pointer(new(struct{}))

// KV represents a key-value pair
type KV struct {
	Key string
	Val any
}

// Twig is a uniform element of a QP-Trie (meaning either a node or a leaf).
type Twig struct {
	bitpack uint64
	pointer unsafe.Pointer // should always point at something allocated!
}

// New returns a new Trie optionally initialized with the given key-value pairs.
func New(init ...KV) *Twig {
	qp := newFanNode(0, nibSizeMax, 0, 0)

	for _, kv := range init {
		qp.Set(kv.Key, kv.Val)
	}

	return qp
}

func (twig *Twig) IsLeaf() bool {
	return twig.bitpack&leafBitMask != 0
}

func (twig *Twig) IsNode() bool {
	return !twig.IsLeaf()
}

func (twig *Twig) IsCutNode() bool {
	return twig.IsNode() && twig.bitpack&cutBitMask != 0
}

func (twig *Twig) IsFanNode() bool {
	return twig.IsNode() && twig.bitpack&cutBitMask == 0
}

func (twig *Twig) Shift() int {
	return int(twig.bitpack & nibShiftMask >> nibShiftOffset)
}

func (twig *Twig) Bitmap() (uint64, int) {
	nibSize := twig.bitpack & nibSizeMask >> nibSizeOffset
	bmpSize := 1<<nibSize + 1
	bmpMask := uint64(1<<bmpSize) - 1

	return twig.bitpack & bmpMask, bmpSize
}

func (twig *Twig) String() string {
	var (
		b     strings.Builder
		shift = strconv.Itoa(twig.Shift())
	)

	b.WriteString("<qptrie|")

	if twig.IsLeaf() {
		b.WriteString("Leaf")
		isEmbed := twig.bitpack&embKeyBitMask != 0
		if isEmbed {
			b.WriteString("|emb")
		}
		b.WriteString("|sh:" + shift)
		b.WriteString(fmt.Sprintf("|%#v", getLeafKey(twig)))
	} else if twig.IsCutNode() {
		b.WriteString("Cut")
		isEmbed := twig.bitpack&embKeySizeMask != 0
		if isEmbed {
			b.WriteString("|emb")
		}
		b.WriteString("|sh:" + shift)
		b.WriteString(fmt.Sprintf("|%#v", getCutNodeKey(twig)))
	} else {
		b.WriteString("Fan")
		b.WriteString("|sh:" + shift)
		pfxSize := twig.bitpack & pfxSizeMask >> pfxSizeOffset
		if pfxSize != 0 {
			var (
				offset = pfxSizeOffset - pfxSize
				mask   = uint64(1)<<pfxSize - 1
				prefix = (twig.bitpack >> offset) & mask
				format = "|pfx:%0" + strconv.FormatUint(pfxSize, 10) + "b"
			)

			b.WriteString(fmt.Sprintf(format, prefix))
		}
		nibLen := twig.bitpack & nibSizeMask >> nibSizeOffset
		b.WriteString("|" + strconv.FormatUint(nibLen, 10) + "bit")

		var (
			bitmap, size = twig.Bitmap()
			format       = "|bmp:%0" + strconv.Itoa(size) + "b"
		)

		b.WriteString(fmt.Sprintf(format, bitmap))
	}

	b.WriteByte('>')

	return b.String()
}

// Get returns a value associated with the given key.
func (twig *Twig) Get(key string) (any, bool) {
	if closest, _, ok := findClosest(twig, key); ok {
		return getLeafKV(closest).Val, true
	}

	return nil, false
}

// Set assigns a value to a key in the given Twig.
func (twig *Twig) Set(key string, val any) (any, bool) {
	closest, key, ok := findClosest(twig, key)
	if ok {
		// matched exactly - replace the value
		return setLeafValue(closest, val), true
	}

	if closest.bitpack&leafBitMask == 0 {
		// it's a node
		switch closest.bitpack&cutBitMask == 0 {
		case true:
			// it's a fan-node
			addToFanNode(closest, key, val, true)
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

// TODO/IDEA:
//
//	also return a matched part or a number of matched bits of a prefix,
//	this is to skip unnecessary comparisons in addTo* routines.
func findClosest(qp *Twig, key string) (*Twig, string, bool) {
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
			// -- cut-node --
			//
			chunk := getCutNodeKey(cur)

			if !strings.HasPrefix(key, chunk) {
				// the cut doesn't match
				return cur, key, false
			}

			// jump to the next twig
			cur = getCutNodeTwig(cur)
			key = key[len(chunk):]
			shift = 0 // reset

			continue
		}

		// -- fan-node --
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
			if len(key)<<byteShift-shift < pfxSize {
				// the key is smaller than the prefix
				return cur, prevKey, false
			}

			var (
				pfxOffset = pfxSizeOffset - pfxSize
				pfxMask   = uint64(1)<<pfxSize - 1
				pfx       = (bitpack >> pfxOffset) & pfxMask
			)

			nib64, key, shift = takeNBits(key, shift, pfxSize)

			if pfx != nib64 {
				// the prefix doesn't match the key
				return cur, prevKey, false
			}
		}

		// TODO: switch on nibSize and call the fast-path routines (take5Bits, ...)
		nib64, key, shift = takeNBits(key, shift, nibSize)
		nib = byte(nib64)

		var (
			mask = uint64(1) << nib
			idx  = bits.OnesCount64(bitmap & (mask - 1))
		)

		if bitmap&mask == 0 {
			// the node doesn't have a nibble
			return cur, prevKey, false
		}

		cur = &(*[bitmapWidthMax]Twig)(cur.pointer)[idx]
	}

	// -- cur is a leaf --

	return cur, key, key == getLeafKey(cur)
}
