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
	index int
}

var EMPTY_REF = Ref{CountedKey{nil,0}, -1}


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
	pool *NodePool
}

// dir calculates the direction for the given key
func (n *Node) dir(key []byte) byte {
	if n.off < len(key) && key[n.off]&n.bit != 0 {
		return 1
	}
	return 0
}

func InitCounter(counter *Counter, pool *NodePool, counted_keys ...CountedKey) *Counter {
	if pool == nil {
		pool = NewNodePool(0)
	}
	*counter = Counter{
		size : 0,
		root : EMPTY_REF,
		pool : pool,
	}
	for _, ckey := range counted_keys {
		counter.IncBy(ckey.Key, ckey.Count)
	}
	return counter
}

func NewCounter(pool *NodePool, counted_keys ...CountedKey) *Counter {
	return InitCounter(&Counter{}, pool, counted_keys...)
}

// Len returns the number of keys in the tree.
func (t *Counter) Len() int {
	return t.size
}

func (t *Counter) Empty() bool {
	return t.root.index == -1 && len(t.root.Key) == 0
}

// Get returns a count associated with the key
func (t *Counter) Get(key []byte) (count int) {
	// test for empty tree
	if t.Empty() {
		return
	}
	// walk for best member
	p := t.root
	var	n *Node

	for p.index != -1 {
		// try next node
		n = &t.pool.Nodes[p.index]
		p = n.child[n.dir(key)]
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

// Set associates a given count with a key. Returns previous count.
func (t *Counter) Set(key []byte, count int) int {
	// TODO: refactor .Set and .IncBy

	// test for empty tree
	if t.Empty() {
		t.root.Key   = key
		t.root.Count = count
		t.size++
		return 0
	}
	// walk for best member
	p := &t.root
	for p.index != -1 {
		// try next node
		node := &t.pool.Nodes[p.index]
		p = &node.child[node.dir(key)]
	}
	// find critical bit
	var off, prev int
	var ch, bit byte
	// find differing byte
	for off = 0; off < len(key); off++ {
		if ch = 0; off < len(p.Key) {
			ch = p.Key[off]
		}
		if keych := key[off]; ch != keych {
			bit = ch ^ keych
			goto ByteFound
		}
	}
	if off < len(p.Key) {
		ch = p.Key[off]
		bit = ch
		goto ByteFound
	}
	// key exists - just increment its counter
	prev = p.Count
	p.Count = count
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
	nn_idx := t.pool.GetNode()
	nn_ptr := &t.pool.Nodes[nn_idx]
	*nn_ptr = Node{off:off, bit:bit, child:[2]Ref{EMPTY_REF, EMPTY_REF}}
	nn_ptr.child[1-ndir].CountedKey = CountedKey{key, count}

	// walk for best insertion node
	wp := &t.root
	for wp.index != -1 {
		n_ptr := &t.pool.Nodes[wp.index]
		if n_ptr.off > off || n_ptr.off == off && n_ptr.bit < bit {
			break
		}
		// try next node
		wp = &n_ptr.child[n_ptr.dir(key)]
	}
	nn_ptr.child[ndir] = *wp
	wp.index = nn_idx
	wp.Key   = nil
	t.size++

	return 0
}

// IncBy incremets a count associated with the key by a given delta and returns it.
func (t *Counter) IncBy(key []byte, delta int) (count int) {
	// test for empty tree
	if t.Empty() {
		t.root.Key   = key
		t.root.Count = delta
		t.size++
		return delta
	}
	// walk for best member
	p := &t.root
	for p.index != -1 {
		// try next node
		node := &t.pool.Nodes[p.index]
		p = &node.child[node.dir(key)]
	}
	// find critical bit
	var off int
	var ch, bit byte
	// find differing byte
	for off = 0; off < len(key); off++ {
		if ch = 0; off < len(p.Key) {
			ch = p.Key[off]
		}
		if keych := key[off]; ch != keych {
			bit = ch ^ keych
			goto ByteFound
		}
	}
	if off < len(p.Key) {
		ch = p.Key[off]
		bit = ch
		goto ByteFound
	}
	// key exists - just increment its counter
	p.Count += delta
	return p.Count
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
	nn_idx := t.pool.GetNode()
	nn_ptr := &t.pool.Nodes[nn_idx]
	*nn_ptr = Node{off:off, bit:bit, child:[2]Ref{EMPTY_REF, EMPTY_REF}}
	nn_ptr.child[1-ndir].CountedKey = CountedKey{key, delta}

	// walk for best insertion node
	wp := &t.root
	for wp.index != -1 {
		n_ptr := &t.pool.Nodes[wp.index]
		if n_ptr.off > off || n_ptr.off == off && n_ptr.bit < bit {
			break
		}
		// try next node
		wp = &n_ptr.child[n_ptr.dir(key)]
	}
	nn_ptr.child[ndir] = *wp
	wp.index = nn_idx
	wp.Key   = nil
	t.size++

	return delta
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
	for p.index != -1 {
		wp = p
		// try next node
		n_ptr := &t.pool.Nodes[p.index]
		dir = n_ptr.dir(key)
		p = &n_ptr.child[dir]
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
		count = t.root.Count
		if t.root.index >= 0 {
			t.pool.PutNode(t.root.index)
		}
		t.root = EMPTY_REF
		return
	}
	idx := wp.index
	*wp = t.pool.Nodes[idx].child[1-dir]
	t.pool.PutNode(idx)
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
	if t.root.index == -1 && len(t.root.Key) == 0 {
		return true
	}
	// shortcut for empty prefix
	if len(prefix) == 0 {
		return t.iterate(t.root, handler)
	}
	// walk for best member
	p, top := t.root, t.root
	for p.index != -1 {
		node := t.pool.Nodes[p.index]
		newtop := node.off < len(prefix)
		// try next node
		p = node.child[node.dir(prefix)]
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
	if p.index != -1 {
		node := t.pool.Nodes[p.index]
		return t.iterate(node.child[0], h) && t.iterate(node.child[1], h)
	}
	return h(p.CountedKey)
}

// Keys returns all keys, as a slice of []byte, in a sorted order.
func (t *Counter) Keys() [][]byte {
	keys := make([][]byte, 0, t.size)

	// empty tree?
	if t.root.index == -1 && len(t.root.Key) == 0 {
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
		if p.index == -1 {
			keys = append(keys, p.Key)
		} else {
			// unshift the children and continue
			node := t.pool.Nodes[p.index]
			to_visit = append(to_visit, &node.child[1], &node.child[0])
		}
	}
	return keys
}

// CountedKeys returns a []CountedKey slice sorted by count (descending)
func (t *Counter) CountedKeys() CountedKeySlice {
	pairs := make(CountedKeySlice, 0, t.size)

	// empty tree?
	if t.root.index == -1 && len(t.root.Key) == 0 {
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
		if p.index == -1 {
			pairs = append(pairs, p.CountedKey)
		} else {
			// unshift the children and continue
			node := t.pool.Nodes[p.index]
			to_visit = append(to_visit, &node.child[1], &node.child[0])
		}
	}

	sort.Sort(pairs)

	return pairs
}

func (t *Counter) debug_dump(idx int, indent string) {
	n := t.pool.Nodes[idx]
	fmt.Println(indent, "NODE", idx, n)
	println(indent, "Left:  off=", n.off, "bit=", n.bit, "key=", string(n.child[0].Key))
	if n.child[0].index != -1 {
		t.debug_dump(n.child[0].index, indent + "  ")
	}
	println(indent, "Right: off=", n.off, "bit=", n.bit, "key=", string(n.child[1].Key))
	if n.child[1].index != -1 {
		t.debug_dump(n.child[1].index, indent + "  ")
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


// --- NodePool ---

type NodePool struct {
	Nodes	[]Node
	FreeIdx	[]int
}

func NewNodePool(pre_alloc int) *NodePool {
	if pre_alloc <= 0 {
		pre_alloc = 256
	}
	return &NodePool{
		Nodes	: make([]Node, 0, pre_alloc),
		FreeIdx	: make([]int,  0, 21),
	}
}

// GetNode allocates a new node (if necessary) and returns
// its index in the .Nodes slice
func (p *NodePool) GetNode() (idx int) {
	if l := len(p.FreeIdx); l > 0 {
		idx = p.FreeIdx[l-1]
		p.FreeIdx = p.FreeIdx[:l-1]
	} else {
		p.Nodes = append(p.Nodes, Node{})
		idx = len(p.Nodes) - 1
	}
	return
}

// PutNode stores a node index in a free-list for a re-use
// by subsequent GetNode calls
func (p *NodePool) PutNode(idx int) {
	p.Nodes[idx] = Node{}  // clear the Node
	p.FreeIdx = append(p.FreeIdx, idx)
}

// Reset forgets about stored nodes and free-list indices (not freeing the memory)
func (p *NodePool) Reset() {
	p.Nodes   = p.Nodes[:0]
	p.FreeIdx = p.FreeIdx[:0]
}
