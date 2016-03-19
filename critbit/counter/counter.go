package counter

import "fmt"
import "sort"
import "bytes"



type CountedKey struct {
	Key   []byte
	Count int
}
type CountedKeySlice []CountedKey

// Ref holds either a CountedKey or a Node index
type Ref struct {
	CountedKey
	node *Node
}

type Node struct {
	child [2]Ref
	// off is the offset of the differing byte
	off   int
	// bit contains the single crit bit in the differing byte
	bit   byte
}


type Counter struct {
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

func InitCounter(counter *Counter, counted_keys ...CountedKey) *Counter {
	*counter = Counter{}
	for _, ckey := range counted_keys {
		counter.IncBy(ckey.Key, ckey.Count)
	}
	return counter
}

func NewCounter(counted_keys ...CountedKey) *Counter {
	return InitCounter(&Counter{}, counted_keys...)
}

// Len returns the number of keys in the tree.
func (t *Counter) Len() int {
	return t.size
}

func (t *Counter) Empty() bool {
	return t.root.node == nil && len(t.root.Key) == 0
}

// Get returns a count associated with the key
func (t *Counter) Get(key []byte) (count int) {
	// test for empty tree
	if t.Empty() {
		return
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
		return
	}
	for i, b := range p.Key {
		if b != key[i] {
			return
		}
	}
	count = p.Count
	return
}

// Replace applies a func to a previous count of a key and replaces the value with return value
func (t *Counter) Replace(key []byte, replace func(int) int) int {
	// test for empty tree
	if t.Empty() {
		t.root.Key   = key
		t.root.Count = replace(0)
		t.size++
		return 0
	}
	// walk for best member
	p := &t.root
	for p.node != nil {
		// try next node
		p = &p.node.child[p.node.dir(key)]
	}
	// find critical bit
	var off, prev int
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
	// key exists - just replace its counter
	prev = p.Count
	p.Count = replace(prev)
	return prev
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
	nn.child[1-ndir].CountedKey = CountedKey{key, replace(0)}

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

	return 0
}

// Set associates a given count with a key. Returns previous count.
func (t *Counter) Set(key []byte, count int) int {
	return t.Replace(key, func(int) int {return count})
}

// IncBy incremets a count associated with the key by a given delta and returns it.
func (t *Counter) IncBy(key []byte, delta int) int {
	return t.Replace(key, func(prev int) int {return prev + delta}) + delta
}

// Inc incremets a count associated with the key by 1 and returns it.
func (t *Counter) Inc(key []byte) int {
	return t.IncBy(key, 1)
}

// Dec decremets a count associated with the key by 1 and returns it.
func (t *Counter) Dec(key []byte) int {
	return t.IncBy(key, -1)
}

// Del removes the key from the tree and returns its counter
func (t *Counter) Del(key []byte) (count int) {
	// test for empty tree
	if t.Empty() {
		return 0
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
		return
	}
	for i, b := range p.Key {
		if b != key[i] {
			return
		}
	}
	count = p.Count
	// delete from the tree
	t.size--
	if wp == nil {
		count  = t.root.Count
		t.root = Ref{}
		return
	}
	*wp = wp.node.child[1-dir]
	return
}

// Merge merges another Counter into this one. Counters of common keys are added up.
// Returns itself.
func (t *Counter) Merge(other *Counter, prefix []byte) *Counter {
	if other != nil {
		adder := func(ckey CountedKey) bool {
			t.IncBy(ckey.Key, ckey.Count)
			return true
		}
		other.Iter(prefix, adder)
	}
	return t
}

// Iter calls a handler for all keys with a given prefix.
// It returns whether all prefixed keys were iterated.
// The handler can continue the process by returning true or abort with false.
func (t *Counter) Iter(prefix []byte, handler func(CountedKey) bool) bool {
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
func (t *Counter) iterate(p Ref, h func(CountedKey) bool) bool {
	if p.node != nil {
		return t.iterate(p.node.child[0], h) && t.iterate(p.node.child[1], h)
	}
	return h(p.CountedKey)
}

// Keys returns all keys, as a slice of []byte, in a sorted order.
func (t *Counter) Keys() [][]byte {
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

// CountedKeys returns a []CountedKey slice sorted by count (descending)
func (t *Counter) CountedKeys() CountedKeySlice {
	pairs := make(CountedKeySlice, 0, t.size)

	// empty tree?
	if t.Empty() {
		return pairs
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
			pairs = append(pairs, p.CountedKey)
		} else {
			// unshift the children and continue
			to_visit = append(to_visit, &p.node.child[1], &p.node.child[0])
		}
	}

	sort.Sort(pairs)

	return pairs
}

func (t *Counter) debug_dump(node *Node, indent string) {
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


// -- CountedKeySlice sort interface --

func (v CountedKeySlice) Len() int           { return len(v) }
func (v CountedKeySlice) Swap(i, j int)      { v[i], v[j] = v[j], v[i] }
func (v CountedKeySlice) Less(i, j int) bool {
	if v[i].Count == v[j].Count {
		return bytes.Compare(v[i].Key, v[j].Key) < 0
	}
	return v[i].Count > v[j].Count  // inverted logic
}

