package qptrie

import (
	"math/bits"
	"unsafe"
)

// newFanNode returns an empty Fan node.
func newFanNode(nibShift, nibSize, pfxSize int, pfx uint64) *Twig {
	bitpack := (uint64(nibShift)<<nibShiftOffset)&nibShiftMask |
		(uint64(nibSize)<<nibSizeOffset)&nibSizeMask

	if pfxSize > 0 {
		// embed the prefix
		var (
			pfxMask   = (uint64(1) << pfxSize) - 1
			pfxOffset = pfxSizeOffset - pfxSize
		)

		bitpack |= (uint64(pfxSize)<<pfxSizeOffset)&pfxSizeMask |
			pfx&pfxMask<<pfxOffset
	}

	return &Twig{
		bitpack: bitpack,
		pointer: unsetPtr, // it is forbidden to have a nil Pointer
	}
}

func addToFanNode(node *Twig, key string, val any, replaceEmpty bool) {
	var (
		bitpack   = node.bitpack
		shift     = int(bitpack & nibShiftMask >> nibShiftOffset)
		pfxSize   = int(bitpack & pfxSizeMask >> pfxSizeOffset)
		prevShift = shift
		prevKey   = key
	)

	if pfxSize > 0 {
		// the fan-node has a prefix - check if it matches the key
		var (
			pfxOffset = pfxSizeOffset - pfxSize
			pfxMask   = (uint64(1) << pfxSize) - 1
			pfx       = (bitpack >> pfxOffset) & pfxMask
		)

		// TODO: check if the key is smaller than the prefix
		// 1) key is smaller and all bits of the key match:
		//
		//	|...........prefix.............|....nib....|
		//	|...........key...........|
		//

		nib64, trimKey, trimShift := takeNBits(key, shift, pfxSize)

		if pfx == nib64 {
			key, shift = trimKey, trimShift
		} else {
			// The prefix doesn't match the key - insert a fan-node before the old one:
			//
			//   old-prefix == "matched" + "unmatched"
			//
			//   [new-fan:"matched"] --> [old-fan:"unmatched"]
			//
			//
			//	{...old-pfx...!.....|..old-nib..}
			//	              * diff bit
			//	[.....key.....!.........................]
			//
			//  {...new-pfx...|!....} + {.|..old-nib..}
			//
			var (
				diff       = (pfx ^ nib64) | (1 << pfxSize)
				newPfxSize = bits.TrailingZeros64(diff) // number of matching bits
				newNibSize = pfxSize - newPfxSize
			)

			for newNibSize < nibSizeMax && newPfxSize > 0 {
				// borrow some bits from a matching part to make a new fan-node wider
				newPfxSize--
				newNibSize++
			}

			oldPfxSize := pfxSize - (newPfxSize + newNibSize) // TODO: check

			// adjust the old node's prefix and shift
			if newNibSize > nibSizeMax {
				oldPfxSize += newNibSize - nibSizeMax
				newNibSize = nibSizeMax
			}

			var (
				oldShift = (prevShift + newPfxSize + newNibSize) % byteWidth
				oldFan   = *node // copy
			)

			oldFan.bitpack &= ^(nibShiftMask | pfxSizeMask) // clear the fields
			oldFan.bitpack |= uint64(oldShift)<<nibShiftOffset | uint64(oldPfxSize)<<pfxSizeOffset

			// insert a fan-node before the old one
			var (
				newPfx = pfx & ((1 << newPfxSize) - 1)
				newFan = newFanNode(prevShift, newNibSize, newPfxSize, newPfx)
				newNib = (pfx >> newPfxSize) & ((1 << newNibSize) - 1)
			)

			newFan.bitpack |= uint64(1) << newNib
			newFan.pointer = unsafe.Pointer(&oldFan) // newFan[newNib] -> oldFan

			*node = *newFan // replace the old fan-node with the new one

			// trim the key and adjust current shift
			_, key, shift = takeNBits(prevKey, prevShift, newPfxSize)

			prevKey = key
			prevShift = shift

			bitpack = node.bitpack
		}
	}

	var (
		nibSize     = int(bitpack & nibSizeMask >> nibSizeOffset)
		bitmapWidth = (uint64(1) << nibSize) + 1 // one extra bit to encode an empty key
		bitmapMask  = (uint64(1) << bitmapWidth) - 1
		bitmap      = bitpack & bitmapMask
	)

	if bitmap == 0 && replaceEmpty {
		// node is empty - replace with a leaf
		*node = *newLeaf(prevKey, prevShift, val)

		return
	}

	nib64, key, shift := takeNBits(key, shift, nibSize)
	nib := byte(nib64 & 0xFF)

	var (
		bit      = uint64(1) << nib
		total    = bits.OnesCount64(bitmap)
		idx      = bits.OnesCount64(bitmap & (bit - 1))
		leaf     = newLeaf(key, shift, val)
		curTwigs = (*(*[bitmapWidthMax]Twig)(node.pointer))[:total]
		newTwigs = make([]Twig, total+1)
		//
		// TODO/IDEA: allocate more than one at a time and remember the number of empty slots
		//            in the bitpack
	)

	// e.g. bitmap: 01011
	//                ^
	//      bit      : 00100
	//      total    : 3
	//      idx      : 2     (OnesCount(00011))
	//      curTwigs : [3]Twig{<0>, <1>, <3>}

	if idx > 0 {
		copy(newTwigs[:idx], curTwigs[:idx])
	}

	newTwigs[idx] = *leaf

	if idx < total {
		copy(newTwigs[idx+1:], curTwigs[idx:])
	}

	node.bitpack |= bit
	node.pointer = unsafe.Pointer(&newTwigs[0])
}
