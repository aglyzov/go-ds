package set

import "fmt"


// Ref holds either a Key or a Node pointer
type Ref struct {
	Key  []byte
	node *Node
}

type Node struct {
	child [2]Ref
	// off is the offset of the differing byte
	off   int
	// bit contains the single crit bit in the differing byte
	bit   byte
}


type Set struct {
	size int
	root Ref
}

// dir calculates the direction for the given key
func (n *Node) dir(key []byte) byte {
	if n.off < len(key) && key[n.off]&n.bit != 0 {
		return 1
	}
	return 0
}

func InitSet(set *Set, keys ...[]byte) *Set {
	*set = Set{}
	for _, key := range keys {
		set.Add(key)
	}
	return set
}

func NewSet(keys ...[]byte) *Set {
	return InitSet(&Set{}, keys...)
}

// Len returns the number of keys in the tree.
func (t *Set) Len() int {
	return t.size
}

func (t *Set) Empty() bool {
	return t.root.node == nil && len(t.root.Key) == 0
}

// Get returns a count associated with the key
func (t *Set) Has(key []byte) bool {
	// test for empty tree
	if t.Empty() {
		return false
	}
	// walk for best member
	p := t.root

	for p.node != nil {
		// try next node
		p = p.node.child[p.node.dir(key)]
	}
	// check for membership
	klen := len(key)
	if klen != len(p.Key) {
		return false
	}
	for i, b := range p.Key {
		if b != key[i] {
			return false
		}
	}
	return true
}

// Set associates a given count with a key. Returns previous count.
func (t *Set) Add(key []byte) bool {
	// test for empty tree
	if t.Empty() {
		t.root.Key = key
		t.size++
		return true
	}
	// walk for best member
	p := &t.root
	for p.node != nil {
		// try next node
		p = &p.node.child[p.node.dir(key)]
	}
	// find critical bit
	var off int
	var ch, bit byte
	var klen = len(key)
	var plen = len(p.Key)
	// find differing byte
	for off = 0; off < klen; off++ {
		if ch = 0; off < plen {
			ch = p.Key[off]
		}
		if keych := key[off]; ch != keych {
			bit = ch ^ keych
			goto ByteFound
		}
	}
	if off < plen {
		ch = p.Key[off]
		bit = ch
		goto ByteFound
	}
	// key exists
	return false
ByteFound:
	// find differing bit
	bit |= bit >> 1
	bit |= bit >> 2
	bit |= bit >> 4
	bit = bit &^ (bit >> 1)
	var ndir byte
	if ch&bit != 0 {
		ndir++
	}
	// insert new node
	nn := Node{off:off, bit:bit}
	nn.child[1-ndir].Key = key

	// walk for best insertion node
	wp := &t.root
	for wp.node != nil {
		n := wp.node
		if n.off > off || n.off == off && n.bit < bit {
			break
		}
		// try next node
		wp = &n.child[n.dir(key)]
	}
	nn.child[ndir] = *wp
	wp.node = &nn
	wp.Key  = nil
	t.size++

	return true
}

// Del removes the key from the tree and returns its counter
func (t *Set) Del(key []byte) bool {
	// test for empty tree
	if t.Empty() {
		return false
	}
	// walk for best member
	var dir byte
	var wp  *Ref
	p := &t.root
	for p.node != nil {
		wp = p
		// try next node
		dir = p.node.dir(key)
		p = &p.node.child[dir]
	}
	// check for membership
	klen := len(key)
	if klen != len(p.Key) {
		return false
	}
	for i, b := range p.Key {
		if b != key[i] {
			return false
		}
	}
	// delete from the tree
	t.size--
	if wp == nil {
		t.root = Ref{}
		return true
	}
	*wp = wp.node.child[1-dir]
	return true
}

// Merge merges another Set into this one. Returns itself.
func (t *Set) Merge(other *Set, prefix []byte) *Set {
	if other != nil {
		adder := func(key []byte) bool {
			t.Add(key)
			return true
		}
		other.Iter(prefix, adder)
	}
	return t
}

// Iter calls a handler for all keys with a given prefix.
// It returns whether all prefixed keys were iterated.
// The handler can continue the process by returning true or abort with false.
func (t *Set) Iter(prefix []byte, handler func([]byte) bool) bool {
	// test empty tree
	if t.Empty() {
		return true
	}
	// shortcut for empty prefix
	if len(prefix) == 0 {
		return t.iterate(t.root, handler)
	}
	// walk for best member
	p, top := t.root, t.root
	for p.node != nil {
		newtop := p.node.off < len(prefix)
		// try next node
		p = p.node.child[p.node.dir(prefix)]
		if newtop {
			top = p
		}
	}
	if len(p.Key) < len(prefix) {
		return true
	}
	for i := 0; i < len(prefix); i++ {
		if p.Key[i] != prefix[i] {
			return true
		}
	}
	return t.iterate(top, handler)
}

// iterate calls the key handler or traverses both node children unless aborted.
func (t *Set) iterate(p Ref, h func([]byte) bool) bool {
	if p.node != nil {
		return t.iterate(p.node.child[0], h) && t.iterate(p.node.child[1], h)
	}
	return h(p.Key)
}

// Keys returns all keys, as a slice of []byte, in a sorted order.
func (t *Set) Keys() [][]byte {
	keys := make([][]byte, 0, t.size)

	// empty tree?
	if t.Empty() {
		return keys
	}

	// Walk the tree without function recursion
	to_visit := make([]*Ref, 1)

	// Walk the left side of the root
	p := &t.root
	to_visit[0] = p

	for l := len(to_visit); l > 0; l = len(to_visit) {
		// shift the list to get the first item

		p = to_visit[l-1]
		to_visit = to_visit[:l-1]

		// leaf?
		if p.node == nil {
			keys = append(keys, p.Key)
		} else {
			// unshift the children and continue
			to_visit = append(to_visit, &p.node.child[1], &p.node.child[0])
		}
	}
	return keys
}

func (t *Set) debug_dump(node *Node, indent string) {
	fmt.Println(indent, "NODE", node)
	println(indent, "Left:  off=", node.off, "bit=", node.bit, "key=", string(node.child[0].Key))
	if node.child[0].node != nil {
		t.debug_dump(node.child[0].node, indent + "  ")
	}
	println(indent, "Right: off=", node.off, "bit=", node.bit, "key=", string(node.child[1].Key))
	if node.child[1].node != nil {
		t.debug_dump(node.child[1].node, indent + "  ")
	}
}

