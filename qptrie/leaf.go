// Package qptrie defines an implementation of a QP-Trie data structure with opinionated
// extensions.
//
// A QP-Trie consists of a number of connected Twigs (nodes and leaves). All branches
// end with a leaf Twig.
//
// Each Twig has two fields:
//
//  - bitpack - 64-bit packed settings of the twig (structure depends on a twig type);
//  - pointer - an unsafe.Pointer to either a leaf value or an array of node children.
//
// Bitpack structure variants:
//
//  - Regular Leaf:
//
//    [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//    <1:leaf> <0:reg> <NNN:shift> ---------------------------------  TODO: embed the first part of the key
//
//  - Embedding Leaf:
//
//    [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//    <1:leaf> <1:emb> <NNN:shift> <NNN:emb-len> <KKK...KKK:emb-key>
//
//  - Fan-node:
//
//    [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [   5:55-51   ] [   50-..   ] [ 32|16|08|04|02|01-00 ]
//    <0:node> <0:fan> <NNN:shift> <NNN:nib-len> <NNNNN:pfx-len> <KK...KK:pfx> <BBBBB...BBBBB:twig-map>
//
//  - Regular Cut-Node:
//
//    [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//    <0:node> <1:cut> <NNN:shift> <000:not-emb> -------------------  TODO: embed the first part of the key (?)
//
//  - Embedding Cut-Node:
//
//    [ 1:63 ] [ 1:62] [ 3:61-59 ] [  3:58-56  ] [    56:55-00     ]
//    <0:node> <1:cut> <NNN:shift> <NNN:emb-len> [KKK...KKK:emb-key]
//
// Pointer variants:
//
//  - Regular Leaf:        unsafe.Pointer( &KV{Key:"tail", Val:<value:interface{}>} )
//  - Embedding Leaf:      unsafe.Pointer( &<value:interface{}> )
//  - Fan-Node:            unsafe.Pointer( <twigs:*[N]Twig> )
//  - Regular Cut-Node:    unsafe.Pointer( &KV{Key:"tail", Val:(interface{}).(<twig:*Twig>)} )
//  - Embedding Cut-Node:  unsafe.Pointer( <twig:*Twig>} )
//
package qptrie

import (
	"unsafe"
)

func newLeaf(key string, shift int, val interface{}) *Twig {
	var leaf = Twig{
		bitpack: leafBitMask | uint64(shift)<<nibShiftOffset,
		pointer: unsetPtr, // it is forbidden to have a nil Pointer
	}

	if len(key) <= embKeySizeMax {
		leaf.bitpack |= embedKey(key)
		leaf.pointer = unsafe.Pointer(&val)
	} else {
		leaf.pointer = unsafe.Pointer(&KV{key, val})
	}

	return &leaf
}

func addToLeaf(leaf *Twig, key string, val interface{}) {
	// find the longest common key prefix
	var (
		shift  = int(leaf.bitpack & nibShiftMask >> nibShiftOffset)
		kv     = getLeafKV(leaf)
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

	// replace the leaf with a node

	var cur = leaf

	if num > 0 {
		// keys have a common prefix - replace cur with a cut-node -> fan-node sequence

		// TODO: add a single fan-node with a prefix if possible

		var (
			prefix  = key[:num]
			fanNode = newFanNode(0, nibSizeMax, 0, 0)
			cutNode = newCutNode(prefix, shift, fanNode)
		)

		shift = 0 // reset

		*cur = *cutNode
		cur = fanNode
	} else {
		*cur = *newFanNode(shift, nibSizeMax, 0, 0)
	}

	// append fan-nodes if necessary

	// TODO: add a single fan-node with a prefix if possible

	var (
		nib1, key1, shift1 = takeNbits(key[num:], shift, nibSizeMax)
		nib2, key2, shift2 = takeNbits(kv.Key[num:], shift, nibSizeMax)
	)

	for nib1 == nib2 {
		// both keys have the same nibble - append a fan-node
		node := newFanNode(shift1, nibSizeMax, 0, 0)

		cur.bitpack |= uint64(1) << nib1
		cur.pointer = unsafe.Pointer(node)
		cur = node

		nib1, key1, shift1 = takeNbits(key1, shift1, nibSizeMax)
		nib2, key2, shift2 = takeNbits(key2, shift2, nibSizeMax)
	}

	// end with two leaves
	var (
		leaf1  = newLeaf(key1, shift1, val)
		leaf2  = newLeaf(key2, shift2, kv.Val)
		leaves [2]Twig
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
}

func getLeafKey(leaf *Twig) string {
	if leaf.bitpack&embKeyBitMask == 0 {
		return (*KV)(leaf.pointer).Key // regular leaf
	}

	return extractKey(leaf.bitpack) // embedded leaf
}

func getLeafKV(leaf *Twig) KV {
	if leaf.bitpack&embKeyBitMask == 0 {
		return *(*KV)(leaf.pointer) // regular leaf
	}

	return KV{
		Key: extractKey(leaf.bitpack),
		Val: *(*interface{})(leaf.pointer),
	}
}

func setLeafValue(leaf *Twig, val interface{}) interface{} {
	var old interface{}

	switch {
	case leaf.bitpack&embKeyBitMask != 0:
		// embedded leaf
		old = *(*interface{})(leaf.pointer)
		leaf.pointer = unsafe.Pointer(&val)

	default:
		// regular leaf
		kv := (*KV)(leaf.pointer)
		old, kv.Val = kv.Val, val
	}

	return old
}
