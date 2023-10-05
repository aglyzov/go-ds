package qptrie

import (
	"unsafe"
)

func newLeaf(key string, shift int, val any) *Twig {
	leaf := Twig{
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

func addToLeaf(leaf *Twig, key string, val any) {
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

	for num < minLen && key[num] == kv.Key[num] {
		num++
	}

	// replace the leaf with a node

	cur := leaf

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
		nib1, key1, shift1 = takeNBits(key[num:], shift, nibSizeMax)
		nib2, key2, shift2 = takeNBits(kv.Key[num:], shift, nibSizeMax)
	)

	for nib1 == nib2 {
		// both keys have the same nibble - append a fan-node
		node := newFanNode(shift1, nibSizeMax, 0, 0)

		cur.bitpack |= uint64(1) << nib1
		cur.pointer = unsafe.Pointer(node)
		cur = node

		nib1, key1, shift1 = takeNBits(key1, shift1, nibSizeMax)
		nib2, key2, shift2 = takeNBits(key2, shift2, nibSizeMax)
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
		Val: *(*any)(leaf.pointer),
	}
}

func setLeafValue(leaf *Twig, val any) any {
	var old any

	switch {
	case leaf.bitpack&embKeyBitMask != 0:
		// embedded leaf
		old = *(*any)(leaf.pointer)
		leaf.pointer = unsafe.Pointer(&val)

	default:
		// regular leaf
		kv := (*KV)(leaf.pointer)
		old, kv.Val = kv.Val, val
	}

	return old
}
