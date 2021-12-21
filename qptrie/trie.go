package qptrie

import (
	"math/bits"
	"unsafe"
)

const (
	leafBitOffset    = 63 // most significant bit in an uint64
	nibShiftOffset   = 60
	embKeyBitOffset  = 59
	embKeySizeOffset = 56

	nibShiftWidth   = 3
	embKeySizeWidth = 3

	byteWidth   = 8
	nibbleWidth = 5

	leafBitMask    uint64 = 1 << leafBitOffset                               // 0b_100000000..0
	nibShiftMask   uint64 = ((1 << nibShiftWidth) - 1) << nibShiftOffset     // 0b_011100000..0
	embKeyBitMask  uint64 = 1 << embKeyBitOffset                             // 0b_000010000..0
	embKeySizeMask uint64 = ((1 << embKeySizeWidth) - 1) << embKeySizeOffset // 0b_000001110..0
	nibbleMask     byte   = (1 << nibbleWidth) - 1
)

var unsetPointer = unsafe.Pointer(new(struct{}))

type KV struct {
	Key string
	Val interface{}
}

type Trie struct {
	bitmap  uint64
	pointer unsafe.Pointer // should always point at something allocated!
}

func New(init ...KV) *Trie {
	var qp = &Trie{
		pointer: unsetPointer, // it is forbidden to have a nil Pointer
	}

	for _, kv := range init {
		qp.Set(kv.Key, kv.Val)
	}

	return qp
}

func (qp *Trie) Get(key string) (interface{}, bool) {
	// walk along a common prefix
	var (
		cur   = qp
		shift = int((cur.bitmap & nibShiftMask) >> nibShiftOffset)
		nib   byte
	)

	for key != "" && !cur.isLeaf() {
		var bitmap = uint32(cur.bitmap)

		if bitmap == 0 {
			// empty node
			return nil, false // not found
		}

		nib, key, shift = getNibble(key, shift)

		var (
			mask = uint32(1) << nib
			idx  = bits.OnesCount32(bitmap & (mask - 1))
		)

		if bitmap&mask == 0 {
			// the node doesn't have the nibble
			return nil, false // not found
		}

		cur = &(*[32]Trie)(cur.pointer)[idx]
	}

	if cur.isLeaf() {
		// look for the longest common key prefix
		var kv = getLeafKV(cur)

		if key == kv.Key {
			return kv.Val, true // found
		}
	}

	return nil, false
}

func (qp *Trie) Set(key string, val interface{}) (interface{}, bool) {
	// walk along a common prefix
	var (
		cur   = qp
		shift = int((cur.bitmap & nibShiftMask) >> nibShiftOffset)
		nib   byte
	)

	for key != "" && !cur.isLeaf() {
		var bitmap = uint32(cur.bitmap)

		if bitmap == 0 {
			// empty node - replace with a leaf
			var leaf = newLeaf(key, shift, val)

			cur.bitmap = leaf.bitmap
			cur.pointer = leaf.pointer

			return nil, false
		}

		nib, key, shift = getNibble(key, shift)

		var (
			mask = uint32(1) << nib
			idx  = bits.OnesCount32(bitmap & (mask - 1))
		)

		if bitmap&mask == 0 {
			// the node doesn't have the nibble yet - add a leaf
			var (
				leaf     = newLeaf(key, shift, val)
				total    = bits.OnesCount32(bitmap)
				curTwigs = (*(*[32]Trie)(cur.pointer))[:total]
				newTwigs = make([]Trie, total+1)
			)

			copy(newTwigs[:idx], curTwigs[:idx])
			newTwigs[idx] = *leaf
			copy(newTwigs[idx+1:], curTwigs[idx:])

			cur.bitmap |= uint64(mask)
			cur.pointer = unsafe.Pointer(&newTwigs[0])

			return nil, false
		}

		cur = &(*[32]Trie)(cur.pointer)[idx]
	}

	if cur.isLeaf() {
		// look for the longest common key prefix
		var (
			kv     = getLeafKV(cur)
			curLen = len(kv.Key)
			keyLen = len(key)
			minLen = keyLen
			idx    int // the nearest index having a different byte
		)

		if curLen < minLen {
			minLen = curLen
		}

		for ; idx < minLen && key[idx] == kv.Key[idx]; idx++ {
		}

		if idx == curLen && idx == keyLen {
			// the leaf has the same key - replace the value
			if cur.bitmap&embKeyBitMask != 0 {
				cur.pointer = unsafe.Pointer(&val)
			} else {
				(*KV)(cur.pointer).Val = val
			}

			return kv.Val, true
		}

		// the leaf has a different key - replace it with a node chain
		// TODO: replace with a prefix-compression node
		cur.bitmap = 0 // reset

		for i := 0; i < idx; i++ {
			nib, key, shift = getNibble(key, shift)
			node := &Trie{
				bitmap:  0,
				pointer: unsetPointer,
			}
			cur.bitmap |= uint64(1) << nib
			cur.pointer = unsafe.Pointer(node)
			cur = node
		}

		// and end the node chain with two leaves
		var (
			nib1, key1, shift1 = getNibble(kv.Key[idx:], shift)
			nib2, key2, shift2 = getNibble(key, shift)

			leaves = []Trie{
				*newLeaf(key1, shift1, kv.Val),
				*newLeaf(key2, shift2, val),
			}
		)

		cur.bitmap |= (uint64(1) << nib1) | (uint64(1) << nib2)
		cur.pointer = unsafe.Pointer(&leaves)
	}

	return nil, false
}

func (qp *Trie) isLeaf() bool {
	return qp.bitmap&leafBitMask != 0
}

func getLeafKV(leaf *Trie) KV {
	var kv KV

	if leaf.bitmap&embKeyBitMask != 0 {
		// key is embedded into the bitmap
		var (
			data [8]byte
			size = leaf.bitmap & embKeySizeMask >> embKeySizeOffset
		)

		for i := uint64(0); i < size; i++ {
			data[i] = byte(leaf.bitmap >> (8 * i))
		}

		kv.Key = string(data[:size])
		kv.Val = *(*interface{})(leaf.pointer)
	} else {
		kv = *(*KV)(leaf.pointer)
	}

	return kv
}

func getNibble(key string, shift int) (byte, string, int) {
	size := len(key)

	if size == 0 {
		return 0, key, 0
	}

	var (
		nshift = (shift + nibbleWidth) % byteWidth
		nib    = (key[0] >> shift) & nibbleMask
	)

	if nshift > shift {
		return nib, key, nshift
	}

	var next byte

	if size >= 2 {
		next = key[1]
	}

	nib |= (next & ((1 << nshift) - 1)) << (byteWidth - shift)

	return nib, key[1:], nshift
}

func newLeaf(key string, shift int, val interface{}) *Trie {
	var leaf = Trie{
		bitmap:  leafBitMask,
		pointer: unsetPointer,
	}

	if n := len(key); n <= 7 {
		// embed the key into the bitmap
		leaf.bitmap |= uint64(shift)<<nibShiftOffset |
			embKeyBitMask | uint64(n)<<embKeySizeOffset

		for i := 0; i < n; i++ {
			leaf.bitmap |= uint64(key[i]) << (8 * i)
		}

		leaf.pointer = unsafe.Pointer(&val)
	} else {
		leaf.pointer = unsafe.Pointer(&KV{key, val})
	}

	return &leaf
}
