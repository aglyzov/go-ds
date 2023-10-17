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

// addToFanNode assumes either a key doesn't match the fan's prefix or a key's
// nibble isn't set in the fan's bitmap.
func addToFanNode(node *Twig, key string, val any, replaceEmpty bool) {
	var (
		shift     = node.Shift()
		prevShift = shift
		prevKey   = key
		pfxSize   = node.FanPrefixSize()
	)

	if pfxSize > 0 {
		// the fan-node has a prefix - check if it matches the key
		var (
			pfx, _ = node.FanPrefix()

			bitsAvail  = len(key)<<byteShift - shift // number of bits available in the key
			bitsToTake = min(pfxSize, bitsAvail)
		)

		nib64, trimKey, trimShift := takeNBits(key, shift, bitsToTake)

		if pfx == nib64 && pfxSize <= bitsAvail {
			key, shift = trimKey, trimShift
		} else {
			// the prefix doesn't match the key

			if pfx == nib64 && bitsAvail <= pfxSize {
				// 1) key is smaller than the prefix and all bits of the key match:
				//
				//  Note: An old nibble can be widened {PP|NN} >>> {|PPNN} if an empty bit is unset.
				//
				//  key: []                             []         ,-[~~~~~]-> ""
				//  fan: {00000000|NN} -[~~]-> ""  >>>  {|00000} -+
				//                                                 `-[00000]-> {000|NN} -[~~]-> ""
				//
				//  key: []                             []         ,-[~~~~~]-> ""
				//  fan: {00000000_0000000|N_NN}   >>>  {|00000} -+
				//                                                 `-[00000]-> {000_00000|00N_NN}
				//
				//  key: [........]                     [........]         ,-[~~~~~]-> ""
				//  fan: {........_000000|NN_NN}   >>>  {........|00000} -+
				//                                                         `-[00000]-> {|0NN_NN}
				//
				//  key: [.._........]                  [.._........]        ,-[~~~~]-> ""
				//  fan: {.._........_0000|NNN}    >>>  {.._........|0000} -+
				//                                                           `-[0000]-> {|NNN}
				//
				//  key: [....._........]               [....._........]         ,-[~~~~~]-> ""
				//  fan: {....._........_0|NNNN}   >>>  {....._........|0NNNN} -+
				//                                                               `-[0NNNN]-> *
				//
				//  key: [..._........]                 [..._........]     ,-[~]-> ""
				//  fan: {..._........_0|NNNNN}    >>>  {..._........|0} -+
				//                                                         `-[0]-> {|NNNNN}
				//
				//  key: [....._........]               [....._........]        ,-[~~~~]-> ""
				//  fan: {....._........|NNNN}     >>>  {....._........|NNNN} -+
				//                                                              `-[NNNN]-> *
				//  --- alt ---
				//
				//  key: [..._........]                 [..._.... ....]                          ,-[~~~~~]-> ""
				//  fan: {..._........_0|NNNNN}    >>>  {..._....|...._0} -[...._0]-> {|NNNNN} -+
				//                                                                               `-[NNNNN]-> *
				//
				//  key: [....._........]               [....._....... .]       ,-[.0000]-> ""
				//  fan: {....._........|NNNN}     >>>  {....._.......|.NNNN} -+
				//                                                              `-[.NNNN]-> *
				_ = 0
			}

			if pfx != nib64 && bitsAvail <= pfxSize {
				// 2) key is smaller than the prefix and at least one bit is different:
				//
				//  key: []                            []         ,-[~~~~~]-> ""
				//  fan: {1PPPPPPP_PPPPPP|NN_NN}  >>>  {|?????} -+
				//                                                `-[1PPPP]-> {PPP_PPPPP|PNN_NN}
				//
				//  key: []                            []         ,-[~~~~~]-> ""
				//  fan: {000001PP_PPPPPP|NN_NN}  >>>  {|?????} -+
				//                                                `-[00000]-> {1PP_PPPPP|PNN_NN}
				//
				//  key: [A.......]                    [ A.......]  ,-[A....]-> "..."
				//  fan: {B......._KKKKKK|NN_NN}  >>>  {|?....}   -+
				//                                                  `-[B....]-> {..._KKKKK|KNN_NN}
				//
				//  key: [.......A]                    [..... ..A]     ,-[..A00]-> ""
				//  fan: {.......B|NN}            >>>  {.....|..?NN} -+
				//                                                     `-[..BNN]-> *
				//
				//  key: [.......A]                    [... ....A]   ,-[....A]-> ""
				//  fan: {.......B|NNNNN}         >>>  {...|....?} -+
				//                                                   `-[....B]-> {|NNNNN}
				//
				//  key: [.....A..]                    [..... A..]     ,-[A.._00]-> ""
				//  fan: {.....B.._KKKK|NNN}      >>>  {.....|?.._??} -+
				//                                                     `-[B.._KK]-> {|KKNNN}
				//
				//  key: [.......A]                    [.... ...A]     ,-[...A_0]-> ""
				//  fan: {.......B_K|NNNNN}       >>>  {....|...?_?} -+
				//                                                     `-[...B_K]-> {K|NNNNN}
				//  --- alt ---
				//
				//  key: [.....A..]                    [... ..A..]   ,-[..A..]-> ""
				//  fan: {.....B.._KKKK|NNN}      >>>  {...|..?..} -+
				//                                                   `-[..B..]-> {KK|KKNNN}
				//
				//  key: [.......A]                    [... ....A]   ,-[....A]-> ""
				//  fan: {.......B_K|NNNNN}       >>>  {...|....?} -+
				//                                                   `-[....B]-> {K|NNNNN}
				//
				_ = 0
			}

			if pfx == nib64 && bitsAvail > pfxSize {
				// 3) key is larger than the prefix and all bits of the key match:
				//
				//  Note: the following assumes a padded tail of a key (i.e. K0000) is not present
				//        in the fan's bitmap because otherwise findClosest() would find another Twig.
				//
				//  key: [....... K]               [....... K]        ,-[K_0000]-> ""
				//  fan: {.......|N_NNNN}     >>>  {.......|N_NNNN} -+
				//                                                    `-[N_NNNN]-> *
				//
				//  key: [....... K_KKKKKKKK]      [....... K_KKKKKKKK]  ,-[K_KKKK]-> "KKKK"
				//  fan: {.......|N_NNNN}     >>>  {.......|N_NNNN}    -+
				//                                                       `-[N_NNNN]-> *
				//  --- alt ---
				//
				//  key: [....... K]               [... ....K]   ,-[....K]-> ""
				//  fan: {.......|N_NNN}      >>>  {...|....?} -+
				//                                               `-[....N]-> {|NNN}
				//
				_ = 0
			}

			if pfx != nib64 && bitsAvail > pfxSize {
				// 4) key is larger than the prefix and at least one bit is different:
				//
				//  key: [A...... K]               [ A....}   ,-[A....]-> "..K"
				//  fan: {B......|N_NNNN}     >>>  {|?....} -+
				//                                            `-[B....]-> {..|N_NNNN}
				//
				//  key: [....A. KK]                [. ...A.KK]   ,-[...A.]-> "KK"
				//  fan: {....B.|NN_NNN}       >>>  {.|...?.}   -+
				//                                                `-[...B.]-> {|NN_NNN}
				//
				//  key: [.....A KK_KKKKKKKK]      [. ....A KK_KKKKKKKK]  ,-[....A]-> "KK_KKKKKKKK"
				//  fan: {.....B|NN_NNN}      >>>  {.|....?}            -+
				//                                                        `-[....B]-> {|NN_NNN}
				//
				_ = 0
			}

			var (
				diff       = (pfx ^ nib64) | (1 << bitsToTake)
				newPfxSize = bits.TrailingZeros64(diff) // number of matching bits
				newNibSize = bitsToTake - newPfxSize
			)

			// borrow some bits from the matching part to make a new fan-node wider
			for newNibSize < nibSizeMax && newPfxSize > 0 {
				newPfxSize--
				newNibSize++
			}

			oldPfxSize := pfxSize - (newPfxSize + newNibSize)

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
		}
	}

	var (
		nibSize   = node.FanNibbleSize()
		bitmap, _ = node.FanBitmap()
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
