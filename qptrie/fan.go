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

func addToFanNode(node *Twig, key string, val interface{}) {
	var (
		bitpack = node.bitpack
		shift   = int(bitpack & nibShiftMask >> nibShiftOffset)
		pfxSize = int(bitpack & pfxSizeMask >> pfxSizeOffset)
	)

	if pfxSize > 0 {
		// the fan-node has a prefix - check if it matches the key
		var (
			keySize   = len(key)*byteWidth - shift
			cmpSize   = pfxSize
			notEnough = keySize < cmpSize
		)

		if notEnough {
			// The key doesn't have enough bits - we can only compare this much.
			// In any case we have to split the prefix inserting a new fan-node.
			cmpSize = keySize
		}

		var (
			pfxOffset = pfxSizeOffset - pfxSize
			pfxMask   = (uint64(1) << cmpSize) - 1
			pfx       = (bitpack >> pfxOffset) & pfxMask
			prevShift = shift
			prevKey   = key
			nib64     uint64
		)

		nib64, key, shift = takeNbits(key, shift, cmpSize)

		if notEnough || pfx != nib64 {
			// the prefix doesn't match the key - insert a fan-node before the old one:
			//
			//   old-prefix == matching-part + unmatched-part
			//
			//   matching-part::new-fan-node -> unmatched-part::old-fan-node
			//
			var (
				diff       = uint32(pfx^nib64) | (1 << cmpSize)
				newPfxSize = bits.TrailingZeros32(diff) // number of matching bits
				newNibSize = cmpSize - newPfxSize
			)

			for newNibSize < nibSizeMax && newPfxSize > 0 {
				// borrow some bits from a matching part to make a new fan-node more deep
				newPfxSize--
				newNibSize++
			}

			// adjust the old node's prefix and shift
			var oldPfxSize = 0

			if newNibSize > nibSizeMax {
				oldPfxSize = newNibSize - nibSizeMax
				newNibSize = nibSizeMax
			}

			var (
				oldShift = (prevShift + newPfxSize + newNibSize) % byteWidth
				oldFan   = *node // copy
			)

			oldFan.bitpack &= ^(nibShiftMask | pfxSizeMask) // clear the fields
			oldFan.bitpack |= uint64(oldShift)<<nibSizeOffset | uint64(oldPfxSize)<<pfxSizeOffset

			// insert a fan-node before the old one
			var (
				newPfx = uint32(pfx) & ((1 << newPfxSize) - 1)
				newFan = newFanNode(prevShift, newNibSize, newPfxSize, newPfx)
				newNib = (nib64 >> newPfxSize) & ((1 << newNibSize) - 1)
			)

			newFan.bitpack |= uint64(1) << newNib
			newFan.pointer = unsafe.Pointer(&oldFan) // newFan[newNib] -> oldFan

			*node = *newFan // replace the old fan-node with the new one

			// trim the key and adjust current shift
			_, key, shift = takeNbits(prevKey, prevShift, newPfxSize)

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

	nib64, key, shift := takeNbits(key, shift, nibSize)
	nib := byte(nib64 & 0xFF)

	var (
		mask     = uint64(1) << nib
		idx      = bits.OnesCount64(bitmap & (mask - 1))
		leaf     = newLeaf(key, shift, val)
		total    = bits.OnesCount64(bitmap)
		curTwigs = (*(*[bitmapWidthMax]Twig)(node.pointer))[:total]
		newTwigs = make([]Twig, total+1)
	)

	copy(newTwigs[:idx], curTwigs[:idx])
	newTwigs[idx] = *leaf
	copy(newTwigs[idx+1:], curTwigs[idx:])

	node.bitpack |= uint64(mask)
	node.pointer = unsafe.Pointer(&newTwigs[0])
}
