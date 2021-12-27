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

func newCutNode(cut string, shift int, twig *Twig) *Twig {
	var node = Twig{
		bitpack: cutBitMask | uint64(shift)<<nibShiftOffset,
		pointer: unsetPtr, // it is forbidden to have a nil Pointer
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

func addToCutNode(node *Twig, key string, val interface{}) {
	// find the longest common key prefix
	var (
		shift  = int(node.bitpack & nibShiftMask >> nibShiftOffset)
		cut    = getCutNodeKey(node)
		twig   = getCutNodeTwig(node)
		cutLen = len(cut)
		keyLen = len(key)
		minLen = keyLen
		num    int // number of common prefix bytes
	)

	if cutLen < minLen {
		minLen = cutLen
	}

	for ; num < minLen && key[num] == cut[num]; num++ {
	}

	if num > 0 {
		// keys have a common prefix - cut the old-cut-node -> old-twig pair like this:
		//
		//                                 ,-> new-cut-node -> old-twig
		//   old-cut-node -> new-fan-node -+
		//                                 `-> new-leaf
		var (
			prefix = key[:num]
			newFan = newFanNode(0, nibSizeMax, 0, 0)
			oldCut = newCutNode(prefix, shift, newFan)
		)

		*node = *oldCut // replace
		shift = 0       // reset

		var (
			// TODO: takeNbits
			nib1, key1, shift1 = take5bits(key[num:], shift)
			nib2, key2, shift2 = take5bits(cut[num:], shift)
		)

		for nib1 == nib2 {
			// both keys have the same nibble - append a fan-node
			node := newFanNode(shift1, nibSizeMax, 0, 0)

			newFan.bitpack |= uint64(1) << nib1
			newFan.pointer = unsafe.Pointer(node)
			newFan = node

			// TODO: takeNbits
			nib1, key1, shift1 = take5bits(key1, shift1)
			nib2, key2, shift2 = take5bits(key2, shift2)
		}

		var (
			newLeaf = newLeaf(key1, shift1, val)
			newCut  = newCutNode(key2, shift2, twig)
			twigs   [2]Twig
		)

		if nib1 < nib2 {
			twigs[0] = *newLeaf
			twigs[1] = *newCut
		} else {
			twigs[1] = *newLeaf
			twigs[0] = *newCut
		}

		newFan.bitpack |= (uint64(1) << nib1) | (uint64(1) << nib2)
		newFan.pointer = unsafe.Pointer(&twigs)

		return
	}

	// keys don't have a common prefix - insert a fan-node like this:
	//
	//                 ,-> new-cut-node -> old-twig
	//   new-fan-node -+
	//                 `-> new-leaf
	//
	var newFan = newFanNode(shift, nibSizeMax, 0, 0)

	*node = *newFan // replace
	newFan = node

	var (
		// TODO: takeNbits
		nib1, key1, shift1 = take5bits(key, shift)
		nib2, key2, shift2 = take5bits(cut, shift)
	)

	for nib1 == nib2 {
		// both keys have the same nibble - append a fan-node
		node := newFanNode(shift1, nibSizeMax, 0, 0)

		newFan.bitpack |= uint64(1) << nib1
		newFan.pointer = unsafe.Pointer(node)
		newFan = node

		// TODO: takeNbits
		nib1, key1, shift1 = take5bits(key1, shift1)
		nib2, key2, shift2 = take5bits(key2, shift2)
	}

	var (
		newLeaf = newLeaf(key1, shift1, val)
		newCut  = newCutNode(key2, shift2, twig)
		twigs   [2]Twig
	)

	if nib1 < nib2 {
		twigs[0] = *newLeaf
		twigs[1] = *newCut
	} else {
		twigs[1] = *newLeaf
		twigs[0] = *newCut
	}

	newFan.bitpack |= (uint64(1) << nib1) | (uint64(1) << nib2)
	newFan.pointer = unsafe.Pointer(&twigs)

	return
}

func getCutNodeKey(node *Twig) string {
	if node.bitpack&embKeySizeMask == 0 {
		return (*KV)(node.pointer).Key // regular cut-node
	}

	return extractKey(node.bitpack)
}

func getCutNodeTwig(node *Twig) *Twig {
	if node.bitpack&embKeySizeMask == 0 {
		return (*KV)(node.pointer).Val.(*Twig) // regular cut-node
	}

	return (*Twig)(node.pointer)
}
