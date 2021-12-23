package qptrie

import (
	"math/bits"
	"unsafe"
)

const (
	// common bit fields
	leafBitOffset    = 63 // 1-bit flag:   1 - leaf, 0 - node
	nibShiftOffset   = 59 // 3-bit number: current nibble's shift in a byte
	embKeySizeOffset = 56 // 3-bit number: number of embedded key bytes [1..7]

	nibShiftWidth   = 3
	embKeySizeWidth = 3

	nibShiftMax   = (1 << nibShiftWidth) - 1   // 0b_111 == 7
	embKeySizeMax = (1 << embKeySizeWidth) - 1 // 0b_111 == 7

	leafBitMask    uint64 = 1 << leafBitOffset                // 0b_100000000..0
	nibShiftMask   uint64 = nibShiftMax << nibShiftOffset     // 0b_001110000..0
	embKeyBitMask  uint64 = 1 << embKeyBitOffset              // 0b_010000000..0
	embKeySizeMask uint64 = embKeySizeMax << embKeySizeOffset // 0b_000001110..0

	// leaf specific bit fields
	embKeyBitOffset = 62 // 1-bit flag:   key tail  1 - embedded, 0 - stored in pointer (*KV)

	// node specific bit fields
	cutBitOffset = 62 // 1-bit flag:  1 - cut-node, 0 - fan-node
	bitmapOffset = 0  // 33-bit map:  1 - corresponding child twig is available

	bitmapWidth = 33

	cutBitMask uint64 = 1 << cutBitOffset                        // 0b_010000000..0
	bitmapMask uint64 = ((1 << bitmapWidth) - 1) << bitmapOffset // 0b_0..000111..1

	// other
	byteWidth   = 8
	nibbleWidth = 5

	nibbleMask byte = (1 << nibbleWidth) - 1
)

var unsetPointer = unsafe.Pointer(new(struct{}))

type KV struct {
	Key string
	Val interface{}
}

// Trie is a twig of a QP-Trie (meaning either a node or a leaf).
//
// Each twig has two fields:
//
//  - bitpack - 64-bit packed settings of the twig (structure depends on a twig type);
//  - pointer - an unsafe.Pointer to either a leaf value or an array of node children.
//
// Bitpack structure variants:
//
//                    [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//  - regular-leaf:   <1:leaf> <0:reg> <NNN:shift> ---------------------------------
//  - emb-tail-leaf:  <1:leaf> <1:emb> <NNN:shift> <NNN:emb-len> <KKK...KKK:emb-key>
//
//                    [ 1:63 ] [ 1:62] [ 3:61-59 ] [ 26:58-33  ] [    33:32-00     ]
//  - fan-node:       <0:node> <0:fan> <NNN:shift> ------------- <BBB...BBB:nib-map>
//
//                    [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//  - reg-cut-node:   <0:node> <1:cut> <NNN:shift> <000:not-emb> -------------------
//  - emb-cut-node:   <0:node> <1:cut> <NNN:shift> <NNN:emb-len> [KKK...KKK:emb-key]
//
// Pointer variants:
//
//  - regular-leaf:   unsafe.Pointer( &KV{Key:"tail", Val:<value:interface{}>} )
//  - emb-tail-leaf:  unsafe.Pointer( &<value:interface{}> )
//  - fan-node:       unsafe.Pointer( <twigs:*[N]Trie> )
//  - reg-cut-node:   unsafe.Pointer( &KV{Key:"tail", Val:(interface{})(<twig:*Trie>)} )
//  - emb-cut-node:   unsafe.Pointer( <twig:*Trie>} )
//
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

	for cur.bitpack&leafBitMask == 0 {
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

	for cur.bitpack&leafBitMask == 0 {
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

	for nib1 == nib2 {
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
	if leaf.bitpack&embKeyBitMask == 0 {
		return *(*KV)(leaf.pointer) // regular leaf
	}

	return KV{
		Key: extractKey(leaf.bitpack),
		Val: *(*interface{})(leaf.pointer),
	}
}

func getNodeCut(node *Trie) string {
	if node.bitpack&cutBitMask == 0 {
		return "" // fan-node doesn't store a key cut
	}

	if node.bitpack&embKeySizeMask == 0 {
		return (*KV)(node.pointer).Key // regular cut-node
	}

	return extractKey(node.bitpack)
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
		bitpack: leafBitMask | uint64(shift)<<nibShiftOffset,
		pointer: unsetPointer,
	}

	if len(key) <= embKeySizeMax {
		leaf.bitpack |= embedKey(key)
		leaf.pointer = unsafe.Pointer(&val)
	} else {
		leaf.pointer = unsafe.Pointer(&KV{key, val})
	}

	return &leaf
}

func newCutNode(cut string, shift int, twig *Trie) *Trie {
	var node = Trie{
		bitpack: uint64(shift) << nibShiftOffset,
		pointer: unsetPointer,
	}

	if len(cut) <= embKeySizeMax {
		// embed the key into the bitmap
		node.bitpack |= embedKey(cut)
		node.pointer = unsafe.Pointer(twig)
	} else {
		node.pointer = unsafe.Pointer(&KV{cut, twig})
	}

	return &node
}

// embedKey embeds a short key into a bitpack.
//
func embedKey(key string) uint64 {
	var size = byte(len(key))

	if size > embKeySizeMax {
		size = embKeySizeMax
	}

	// NOTE: we rely on embKeyBitMask == cutBitMask here because
	//       embedKey() is used in both newLeaf() and newCutNode()
	//
	var bitpack = embKeyBitMask | uint64(size) << embKeySizeOffset

	for i := byte(0); i < size; i++ {
		bitpack |= uint64(key[i]) << (byteWidth * i)
	}

	return bitpack
}

// extractKey extracts an embedded key from a bitpack.
//
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
