package set

import (
	//"fmt"
	"github.com/hideo55/go-popcount"
)

type Set struct {
	root		*Node
	size		uint64
}

type Node struct {
	bitmap		[4]uint64  // 256 bits representing 2**8 entries
	children	[]*Node
}

func NewSet() *Set {
	return &Set{
		root	: &Node{},
		size	: 0,
	}
}

func (t *Set) Len() uint64 {
	if t == nil {
		return 0
	}
	return t.size
}
func (t *Set) Has(val uint64) bool {
	if t == nil {
		return false
	}

	shift := byte(64-8)
	node  := t.root

	//fmt.Println("------------------")

	for i := 0; ; i++ {
		idx := byte((val >> shift) & 0xFF)
		ofs := idx >> 6
		bmp := node.bitmap[ofs]
		idx  = idx & 0x3F  // the lowest 6 bits (2**6 == 64)
		//fmt.Printf("ofs:%v, bmp:%0x, idx:%v\n", ofs, bmp, idx)
		if (bmp >> idx) & 0x01 == 0 {
			return false  // underlying nodes don't have it
		}
		if i == 7 {
			break  // this is a leaf
		}
		cnt := popcount.Count(bmp & ((1 << idx) - 1))
		for j := byte(0); j < ofs; j++ {
			cnt += popcount.Count(node.bitmap[j])
		}
		node = node.children[cnt]
		shift -= 8
	}

	return true
}
func (t *Set) Add(val uint64) (add bool) {
	shift := byte(64-8)
	node  := t.root

	//fmt.Println("------------------")

	for i := 0; ; i++ {
		idx := byte((val >> shift) & 0xFF)
		ofs := idx >> 6
		bmp := node.bitmap[ofs]
		idx  = idx & 0x3F  // the lowest 6 bits (2**6 == 64)
		//fmt.Printf("ofs:%v, bmp:%0x, idx:%v\n", ofs, bmp, idx)
		add = false
		if (bmp >> idx) & 0x01 == 0 {
			node.bitmap[ofs] = bmp | (1 << idx)
			add = true
		}
		if i == 7 {
			if add {
				t.size++
			}
			break  // this is a leaf
		}
		cnt := popcount.Count(bmp & ((1 << idx) - 1))
		for j := byte(0); j < ofs; j++ {
			cnt += popcount.Count(node.bitmap[j])
		}
		if add {
			num  := len(node.children)
			node.children = append(node.children, nil)
			if num > 0 {
				copy(node.children[cnt:num], node.children[cnt+1:])
			}
			next := &Node{}
			node.children[cnt] = next
			node = next
		} else {
			node = node.children[cnt]
		}
		shift -= 8
	}

	return
}
