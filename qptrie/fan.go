package qptrie

import (
	"math/bits"
	"unsafe"
)

func newFanNode(nibShift, nibSize, pfxSize int, pfx uint32) *Twig {
	bitpack := (uint64(nibShift)<<nibShiftOffset)&nibShiftMask |
		(uint64(nibSize)<<nibSizeOffset)&nibSizeMask |
		(uint64(pfxSize)<<pfxSizeOffset)&pfxSizeMask

	if pfxSize > 0 {
		// embed the prefix
		var (
			pfxMask   = (uint64(1) << pfxSize) - 1
			pfxOffset = pfxSizeOffset - pfxSize
		)

		bitpack |= uint64(pfx) & pfxMask << pfxOffset
	}

	return &Twig{
		bitpack: bitpack,
		pointer: unsetPtr, // it is forbidden to have a nil Pointer
	}
}

func addToFanNode(node *Twig, key string, val any) {
	var (
		bitpack = node.bitpack
		shift   = int(bitpack & nibShiftMask >> nibShiftOffset)
		pfxSize = int(bitpack & pfxSizeMask >> pfxSizeOffset)
	)

	if pfxSize > 0 {
		// the fan-node has a prefix - check if it matches the key
		var (
			pfxOffset = pfxSizeOffset - pfxSize
			pfxMask   = (uint64(1) << pfxSize) - 1
			pfx       = (bitpack >> pfxOffset) & pfxMask
			prevShift = shift
			prevKey   = key
			nib64     uint64
		)

		nib64, key, shift = takeNBits(key, shift, pfxSize)

		if pfx != nib64 {
			//if pfx != nib64 {
			// the prefix doesn't match the key - insert a fan-node before the old one:
			//
			//   old-prefix == matching-part + unmatched-part
			//
			//   matching-part::new-fan-node -> unmatched-part::old-fan-node
			//
			var (
				diff       = uint32(pfx^nib64) | (1 << pfxSize)
				newPfxSize = bits.TrailingZeros32(diff) // number of matching bits
				newNibSize = pfxSize - newPfxSize
			)

			for newNibSize < nibSizeMax && newPfxSize > 0 {
				// borrow some bits from a matching part to make a new fan-node more deep
				newPfxSize--
				newNibSize++
			}

			// adjust the old node's prefix and shift
			oldPfxSize := pfxSize - pfxSize

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
				newPfx = uint32(pfx) & ((1 << newPfxSize) - 1)
				newFan = newFanNode(prevShift, newNibSize, newPfxSize, newPfx)
				newNib = (pfx >> newPfxSize) & ((1 << newNibSize) - 1)
			)

			newFan.bitpack |= uint64(1) << newNib
			newFan.pointer = unsafe.Pointer(&oldFan) // newFan[newNib] -> oldFan

			*node = *newFan // replace the old fan-node with the new one

			// trim the key and adjust current shift
			_, key, shift = takeNBits(prevKey, prevShift, newPfxSize)

			bitpack = node.bitpack
		}
	}

	var (
		nibSize     = int(bitpack & nibSizeMask >> nibSizeOffset)
		bitmapWidth = (uint64(1) << nibSize) + 1 // one extra bit to encode an empty key
		bitmapMask  = (uint64(1) << bitmapWidth) - 1
		bitmap      = bitpack & bitmapMask
	)

	if bitmap == 0 {
		// node is empty - replace with a leaf
		*node = *newLeaf(key, shift, val)

		return
	}

	nib64, key, shift := takeNBits(key, shift, nibSize)
	nib := byte(nib64 & 0xFF)

	var (
		mask     = uint64(1) << nib
		total    = bits.OnesCount64(bitmap)
		idx      = bits.OnesCount64(bitmap & (mask - 1))
		leaf     = newLeaf(key, shift, val)
		curTwigs = (*(*[bitmapWidthMax]Twig)(node.pointer))[:total]
		newTwigs = make([]Twig, total+1)
	)

	copy(newTwigs[:idx], curTwigs[:idx])
	newTwigs[idx] = *leaf
	copy(newTwigs[idx+1:], curTwigs[idx:])

	node.bitpack |= uint64(mask)
	node.pointer = unsafe.Pointer(&newTwigs[0])
}
