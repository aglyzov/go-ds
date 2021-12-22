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
	bitmapOffset     = 0

	nibShiftWidth   = 3
	embKeySizeWidth = 3
	bitmapWidth     = 33

	byteWidth   = 8
	nibbleWidth = 5

	leafBitMask    uint64 = 1 << leafBitOffset                               // 0b_100000000..0
	nibShiftMask   uint64 = ((1 << nibShiftWidth) - 1) << nibShiftOffset     // 0b_011100000..0
	embKeyBitMask  uint64 = 1 << embKeyBitOffset                             // 0b_000010000..0
	embKeySizeMask uint64 = ((1 << embKeySizeWidth) - 1) << embKeySizeOffset // 0b_000001110..0
	bitmapMask     uint64 = ((1 << bitmapWidth) - 1) << bitmapOffset         // 0b_0..0111..1

	nibbleMask byte = (1 << nibbleWidth) - 1
)

var unsetPointer = unsafe.Pointer(new(struct{}))

type KV struct {
	Key string
	Val interface{}
}

type Trie struct {
	bitpack uint64
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
		shift = int((cur.bitpack & nibShiftMask) >> nibShiftOffset)
		nib   byte
	)

	for !cur.isLeaf() {
		var bitmap = cur.bitpack & bitmapMask >> bitmapOffset

		if bitmap == 0 {
			// empty node
			return nil, false // not found
		}

		nib, key, shift = getNibble(key, shift)

		var (
			mask = uint64(1) << nib
			idx  = bits.OnesCount64(bitmap & (mask - 1))
		)

		if bitmap&mask == 0 {
			// the node doesn't have the nibble
			return nil, false // not found
		}

		cur = &(*[bitmapWidth]Trie)(cur.pointer)[idx]
	}

	// look for the longest common key prefix
	var kv = getLeafKV(cur)

	if key == kv.Key {
		return kv.Val, true // found
	}

	return nil, false
}

func (qp *Trie) Set(key string, val interface{}) (interface{}, bool) {
	// walk along a common prefix
	var (
		cur   = qp
		shift = int((cur.bitpack & nibShiftMask) >> nibShiftOffset)
		nib   byte
	)

	for !cur.isLeaf() {
		var bitmap = cur.bitpack & bitmapMask >> bitmapOffset

		if bitmap == 0 {
			// empty node - replace with a leaf
			var leaf = newLeaf(key, shift, val)

			cur.bitpack = leaf.bitpack
			cur.pointer = leaf.pointer

			return nil, false
		}

		nib, key, shift = getNibble(key, shift)

		var (
			mask = uint64(1) << nib
			idx  = bits.OnesCount64(bitmap & (mask - 1))
		)

		if bitmap&mask == 0 {
			// the node doesn't have the nibble yet - add a leaf
			var (
				leaf     = newLeaf(key, shift, val)
				total    = bits.OnesCount64(bitmap)
				curTwigs = (*(*[bitmapWidth]Trie)(cur.pointer))[:total]
				newTwigs = make([]Trie, total+1)
			)

			copy(newTwigs[:idx], curTwigs[:idx])
			newTwigs[idx] = *leaf
			copy(newTwigs[idx+1:], curTwigs[idx:])

			cur.bitpack |= uint64(mask)
			cur.pointer = unsafe.Pointer(&newTwigs[0])

			return nil, false
		}

		cur = &(*[32]Trie)(cur.pointer)[idx]
	}

	// look for the longest common key prefix in the leaf
	var (
		kv     = getLeafKV(cur)
		curLen = len(kv.Key)
		keyLen = len(key)
		minLen = keyLen
		num    int // number of common prefix bytes
	)

	if curLen < minLen {
		minLen = curLen
	}

	for ; num < minLen && key[num] == kv.Key[num]; num++ {
	}

	if num == curLen && num == keyLen {
		// the leaf has the same key - replace the value
		if cur.bitpack&embKeyBitMask != 0 {
			cur.pointer = unsafe.Pointer(&val)
		} else {
			(*KV)(cur.pointer).Val = val
		}

		return kv.Val, true
	}

	// the leaf has a different key - replace it with a node chain
	// TODO: replace with a prefix-compression node
	cur.bitpack = 0 // reset

	var (
		chainBits = num*byteWidth - shift
		chainLen  = chainBits / nibbleWidth
	)

	for i := 0; i < chainLen; i++ {
		nib, key, shift = getNibble(key, shift)
		node := &Trie{
			bitpack: uint64(shift) << nibShiftOffset,
			pointer: unsetPointer,
		}
		cur.bitpack |= uint64(1) << nib
		cur.pointer = unsafe.Pointer(node)
		cur = node
	}

	// and end the node chain with two leaves
	var (
		keyLenDiff         = keyLen - len(key)
		nib1, key1, shift1 = getNibble(key, shift)
		nib2, key2, shift2 = getNibble(kv.Key[keyLenDiff:], shift)
	)

	if nib1 == nib2 {
		// the last nibble of the shortest key is the same in both keys - add another node
		node := &Trie{
			bitpack: uint64(shift1) << nibShiftOffset,
			pointer: unsetPointer,
		}
		cur.bitpack |= uint64(1) << nib1
		cur.pointer = unsafe.Pointer(node)
		cur = node

		nib1, key1, shift1 = getNibble(key1, shift1)
		nib2, key2, shift2 = getNibble(key2, shift2)
	}

	var (
		leaf1  = newLeaf(key1, shift1, val)
		leaf2  = newLeaf(key2, shift2, kv.Val)
		leaves [2]Trie
	)

	if nib1 < nib2 {
		leaves[0] = *leaf1
		leaves[1] = *leaf2
	} else {
		leaves[1] = *leaf1
		leaves[0] = *leaf2
	}

	cur.bitpack |= (uint64(1) << nib1) | (uint64(1) << nib2)
	cur.pointer = unsafe.Pointer(&leaves)

	return nil, false
}

func (qp *Trie) isLeaf() bool {
	return qp.bitpack&leafBitMask != 0
}

func getLeafKV(leaf *Trie) KV {
	var kv KV

	if leaf.bitpack&embKeyBitMask != 0 {
		// key is embedded into the bitmap
		var (
			data [8]byte
			size = leaf.bitpack & embKeySizeMask >> embKeySizeOffset
		)

		for i := uint64(0); i < size; i++ {
			data[i] = byte(leaf.bitpack >> (8 * i))
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
		return bitmapWidth - 1, key, 0
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
		bitpack: leafBitMask,
		pointer: unsetPointer,
	}

	if n := len(key); n <= 7 {
		// embed the key into the bitmap
		leaf.bitpack |= uint64(shift)<<nibShiftOffset |
			embKeyBitMask | uint64(n)<<embKeySizeOffset

		for i := 0; i < n; i++ {
			leaf.bitpack |= uint64(key[i]) << (8 * i)
		}

		leaf.pointer = unsafe.Pointer(&val)
	} else {
		leaf.pointer = unsafe.Pointer(&KV{key, val})
	}

	return &leaf
}
