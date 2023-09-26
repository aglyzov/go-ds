package qptrie

import (
	"math/bits"
	"unsafe"
)

func newCutNode(cut string, shift int, twig *Twig) *Twig {
	node := Twig{
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

// addToCutNode assumes a key doesn't match the cut and replaces the node with a
// combination of fan-nodes and cut-nodes in order to add the given key-value pair.
//
// Possible scenarios:
// ------------------
//
// 1) key is smaller and all bits of the key match:
//
//	|...........cut.................|....next....|
//	|...........key...........|
//
// 2) there is at least one bit of difference (no matter the key size):
//
//	|........cut....!.......|....next....|
//	                * diff bit
//	|........key....!..|
//
//	or
//
//	|........cut....!.......|....next....|
//	                * diff bit
//	|........key....!..............|
func addToCutNode(node *Twig, key string, val interface{}) {
	// find the longest common key prefix
	var (
		shift     = int(node.bitpack & nibShiftMask >> nibShiftOffset)
		cut       = getCutNodeKey(node)
		cutBytes  = len(cut)
		keyBytes  = len(key)
		minBytes  = keyBytes // min(cutBytes, keyBytes)
		headBytes int        // amount of full bytes in a common prefix (head)
	)

	if minBytes > cutBytes {
		minBytes = cutBytes
	}

	// count full bytes in a common prefix
	for ; headBytes < minBytes && key[headBytes] == cut[headBytes]; headBytes++ {
	}

	var (
		keyBits = keyBytes*byteWidth - shift // total bits in the key
		cutBits = cutBytes*byteWidth - shift // total bits in the cut
	)

	// determine total number of bits in a head
	headBits := headBytes*byteWidth - shift // always <= cutBits (preliminary)

	if headBytes < minBytes {
		// there is at least one bit of difference
		headBits += bits.LeadingZeros8(key[headBytes] ^ cut[headBytes])
	}

	var (
		tailBits = cutBits - headBits // cut's tail

		// next twig
		next       = getCutNodeTwig(node)
		nextIsNode = next.bitpack&leafBitMask == 0
		nextIsFan  = next.bitpack&cutBitMask == 0
		// nextPfxSize = 0 // preliminary

		// preliminary parameters of a new fan-node
		pfxSize = headBits
		nibSize = nibSizeMax
		remBits = cutBits - pfxSize - nibSize
		// brwBits = 0 // borrowed bits
	)

	if nextIsNode && nextIsFan {
		var (
			size   = (next.bitpack & pfxSizeMask) >> pfxSizeOffset
			offset = pfxSizeOffset - size
			mask   = (uint64(1) << size) - 1
		)

		nextPfxSize = int((next.bitpack >> offset) & mask)
	}

	/* TODO
	// extend the tail at the expense of the next fan's prefix if necessary
	for remBits < 0 && nextPfxSize > 0 {
		// TODO: do it in one go without a loop
		//
		//                   |-->
		//  |..head...|.tail.|...nPfx...|       |..head...|...tail...|.nPfx.|
		//  |...pfx...|..nibble..|         >>>  |...pfx...|..nibble..|
		//
		remBits++
		brwBits++
		nextPfxSize--

		// TODO: tweak next's shift & prefix, append the borrowed bits to the tail
	}
	*/

	// shrink the new fan's prefix if necessary
	for remBits < 0 && pfxSize > 0 && pfxSize+nibSize-1 > headBits {
		// TODO: do it in one go without a loop
		//
		//  |...head..|.tail.|           |...head..|.tail.|
		//  |...pfx...|..nibble..|  >>>  |.pfx.|..nibble..|
		//         <--|       <--|
		//
		remBits++
		pfxSize--
	}

	// shrink the new fan's nibble if necessary
	for remBits < 0 && nibSize > 1 && pfxSize+nibSize-1 > headBits {
		// TODO: do it in one go without a loop
		//
		//  |.tail.|           |.tail.|
		//  |..nibble..|  >>>  |nibble|
		//          <--|
		//
		remBits++
		nibSize--
	}

	// shrink the prefix that is too large
	var (
		pfxSizeMax = 18 // TODO: make it dynamic depending on a nibble size
		pfxOffset  = 0
	)

	if delta := pfxSize - pfxSizeMax; delta > 0 {
		//
		//  |.....head.....|...tail...|       |.....head.....|...tail...|
		//  |......pfx.....|.nibble.|    >>>       |...pfx...|.nibble.|
		//  |-->
		//
		pfxSize -= delta
		pfxOffset = delta
	}

	// TODO: create a fan-node to insert in the middle of the cut

	if pfxOffset > 0 {
		// TODO:

		if pfxOffset <= pfxSizeMax+nibSizeMax {
			// delta is small enough to be covered by a fan-node with a prefix.
			// replace the cut-node with a new fan-node.
			var (
				nibSize = nibSizeMax
				pfxSize = pfxOffset - nibSizeMax
			)

			if nibSize < pfxOffset {
				//
				//  |.delta.|          |.delta.|
				//  |..nibble..|  >>>  |.nibble|
				//          <--|
				//
				nibSize = pfxOffset
				pfxSize = 0
			}

			if pfxSize > pfxOffset-nibSize {
				//
				//  |...delta...|          |...delta...|
				//  |..pfx..|nibble|  >>>  |.pfx|nibble|
				//       <--|
				//
				pfxSize = pfxOffset - nibSize
			}

			var (
				pfx      uint64
				nib      uint64
				nibShift = shift
			)

			if pfxSize > 0 {
				pfx, cut, nibShift = takeNbits(cut, shift, pfxSize)
			}

			nib, cut, shift = takeNbits(cut, nibShift, nibSize)

			*node = *newFanNode(shift, nibSize, pfxSize, uint32(pfx))

			node.bitpack |= uint64(1) << nib
			// node.pointer = // TODO: add nib -> pointer to the new fan-node
		} else {
			// trim the cut-node key to cover the fist delta bits
		}
	}

	if headBytes > 0 {
		// keys have a common prefix - cut the old-cut-node -> old-twig pair like this:
		//
		//                                 ,-> new-cut-node -> old-twig
		//   old-cut-node -> new-fan-node -+
		//                                 `-> new-leaf
		var (
			prefix = key[:num] // TODO: fix - prefix has to be less than one of the keys
			newFan = newFanNode(0, nibSizeMax, 0, 0)
			oldCut = newCutNode(prefix, shift, newFan)
		)

		*node = *oldCut // replace
		shift = 0       // reset

		var (
			nib1, key1, shift1 = takeNbits(key[num:], shift, nibSizeMax)
			nib2, key2, shift2 = takeNbits(cut[num:], shift, nibSizeMax)
		)

		for nib1 == nib2 {
			// both keys have the same nibble - append a fan-node
			node := newFanNode(shift1, nibSizeMax, 0, 0)

			newFan.bitpack |= uint64(1) << nib1
			newFan.pointer = unsafe.Pointer(node)
			newFan = node

			nib1, key1, shift1 = takeNbits(key1, shift1, nibSizeMax)
			nib2, key2, shift2 = takeNbits(key2, shift2, nibSizeMax)
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
	newFan := newFanNode(shift, nibSizeMax, 0, 0)

	*node = *newFan // replace
	newFan = node

	var (
		nib1, key1, shift1 = takeNbits(key, shift, nibSizeMax)
		nib2, key2, shift2 = takeNbits(cut, shift, nibSizeMax)
	)

	for nib1 == nib2 {
		// both keys have the same nibble - append a fan-node
		node := newFanNode(shift1, nibSizeMax, 0, 0)

		newFan.bitpack |= uint64(1) << nib1
		newFan.pointer = unsafe.Pointer(node)
		newFan = node

		nib1, key1, shift1 = takeNbits(key1, shift1, nibSizeMax)
		nib2, key2, shift2 = takeNbits(key2, shift2, nibSizeMax)
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
