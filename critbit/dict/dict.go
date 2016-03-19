package dict

import "fmt"


type Item struct {
	Key []byte
	Val interface{}
}
type ItemSlice []Item


// Ref holds either a Item or a Node index
type Ref struct {
	Item
	node *Node
}

func (ref *Ref) String() string {
	if ref == nil {
		return "Ref(nil)"
	}
	if ref.node != nil {
		return fmt.Sprintf("<Ref NODE off=%v, mask=%08b>", ref.node.off, ref.node.bit)
	} else {
		return fmt.Sprintf("<Ref LEAF key=%q, val=%v>", ref.Key, ref.Val)
	}
}


type Node struct {
	child [2]Ref
	// off is the offset of the differing byte
	off   int
	// bit contains the single crit bit in the differing byte
	bit   byte
}


type Dict struct {
	size int
	root Ref
}

// dir calculates the direction for the given key
func (n *Node) dir(key []byte) byte {
	if n.off < len(key) && key[n.off] & n.bit != 0 {
		//fmt.Printf("dir() -> 1   off=%v  byte=%08b  bit=%08b\n", n.off, key[n.off], n.bit)
		return 1
	}
	//fmt.Printf("dir() -> 0   off=%v  key=%v  bit=%08b\n", n.off, key, n.bit)
	return 0
}

func InitDict(dict *Dict, items ...Item) *Dict {
	*dict = Dict{}
	for _, item := range items {
		dict.Set(item.Key, item.Val)
	}
	return dict
}

func NewDict(items ...Item) *Dict {
	return InitDict(&Dict{}, items...)
}

// Len returns the number of keys in the tree.
func (t *Dict) Len() int {
	return t.size
}

func (t *Dict) Empty() bool {
	return t.root.node == nil && len(t.root.Key) == 0
}

// Get returns a value associated with the key
func (t *Dict) Get(key []byte) (val interface{}, ok bool) {
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
	val = p.Val
	ok  = true
	return
}

// Replace applies a func to a previous value of a key and replaces it with the result.
// Returns the previous value.
func (t *Dict) Replace(key []byte, replace func(interface{}) interface{}) interface{} {
	// test for empty tree
	if t.Empty() {
		t.root.Key = key
		t.root.Val = replace(nil)
		t.size++
		return nil
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
	var prev interface{}
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
	// key exists - just increment its dict
	prev  = p.Val
	p.Val = replace(prev)
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
	nn.child[1-ndir].Item = Item{key, replace(nil)}

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

	return nil
}

// Set associates a given value with a key. Returns previous value (if any).
func (t *Dict) Set(key []byte, val interface{}) interface{} {
	return t.Replace(key, func(interface{}) interface{} {return val})
}

// Del removes the key from the tree and returns its value (if any)
func (t *Dict) Del(key []byte) (val interface{}) {
	// test for empty tree
	if t.Empty() {
		return
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
	val = p.Val
	// delete from the tree
	t.size--
	if wp == nil {
		val = t.root.Val
		t.root = Ref{}
		return
	}
	*wp = wp.node.child[1-dir]
	return
}

// Merge merges another Dict into this one. Dicts of common keys are added up.
// Returns itself.
func (t *Dict) Merge(other *Dict, prefix []byte) *Dict {
	if other != nil {
		adder := func(item Item) bool {
			t.Set(item.Key, item.Val)
			return true
		}
		other.Iter(prefix, adder)
	}
	return t
}


// FindPathGE returns a path to a Ref that is greater-or-equal to the key
func (t *Dict) FindPathGE(key []byte) (path *RefPath) {
	// test empty tree
	if t.Empty() {
		return
	}
	// descend to the closest leaf
	path = NewRefPath()
	ref := &t.root
	dir := byte(1)
	path.Append(ref, dir)

	for ref.node != nil {
		dir = ref.node.dir(key)
		ref = &ref.node.child[dir]
		path.Append(ref, dir)
	}
	// fine tune the path
	lkey := len(key)
	went_right := false
	went_left := false

	finetune:
	for ref != nil {
		//fmt.Printf("Fine-tune: path=%v\n", path.GetLeaf())
		lref := len(ref.Key)
		for i, b := range key {
			if i >= lref {
				// this leaf is less (bad)
				if went_left {
					// we couldn't find a lesser GE key - revert and stop
					ref = path.Revert()
					break finetune
				}
				// look further right
				went_right = true
				ref = path.TrackNext()
				continue finetune
			}
			B := ref.Key[i]
			switch {
			case b < B:
				// this leaf is greater (good)
				if went_right {
					// we have found the lowest GE key - stop
					break finetune
				}
				// check if the previous is also GE (better)
				went_left = true
				path.Backup()
				ref = path.TrackPrev()
				continue finetune
			case b > B:
				// this leaf is less (bad)
				if went_left {
					// we couldn't find a lesser GE key - revert and stop
					ref = path.Revert()
					break finetune
				}
				// look further right
				went_right = true
				ref = path.TrackNext()
				continue finetune
			}
		}
		if lref == lkey {
			// full match
			break finetune
		}
	    // this leaf is greater (good)
		if went_right {
			// we have found the lowest GE key - stop
			break finetune
		}
		// check if a lesser key is also GE (good)
		went_left = true
		path.Backup()
		ref = path.TrackPrev()
	}
	if ref == nil && went_left {
		ref = path.Revert()
	}
	return
}

// FindPathLE returns a path to a Ref that is less-or-equal to the key
func (t *Dict) FindPathLE(key []byte) (path *RefPath) {
	// test empty tree
	if t.Empty() {
		return
	}
	// descend to the closest leaf
	path = NewRefPath()
	ref := &t.root
	dir := byte(1)
	path.Append(ref, dir)

	for ref.node != nil {
		dir = ref.node.dir(key)
		ref = &ref.node.child[dir]
		path.Append(ref, dir)
	}
	// fine tune the path
	lkey := len(key)
	went_left  := false
	went_right := false

	finetune:
	for ref != nil {
		//fmt.Printf("Fine-tune: path=%v\n", path.GetLeaf())
		lref := len(ref.Key)
		for i, b := range key {
			if i >= lref {
				// this leaf is less (good)
				if went_left {
					// we have found the greatest LE key - stop
					break finetune
				}
				// check if the next is also LE (better)
				went_right = true
				path.Backup()
				ref = path.TrackNext()
				continue finetune
			}
			B := ref.Key[i]
			switch {
			case b < B:
				// this leaf is greater (bad)
				if went_right {
					// we couldn't find a bigger LE key - revert and stop
					ref = path.Revert()
					break finetune
				}
				// look further left
				went_left = true
				ref = path.TrackPrev()
				continue finetune
			case b > B:
				// this leaf is less (good)
				if went_left {
					// we have found the greatest LE key - stop
					break finetune
				}
				// check if the next is also LE (better)
				went_right = true
				path.Backup()
				ref = path.TrackNext()
				continue finetune
			}
		}
		if lref == lkey {
			// full match
			break finetune
		}
	    // this leaf is greater (bad)
		if went_right {
			// we couldn't find a bigger LE key - revert and stop
			ref = path.Revert()
			break finetune
		}
		// look further left
		went_left = true
		ref = path.TrackPrev()
	}
	if ref == nil && went_right {
		ref = path.Revert()
	}
	return
}

// FindPathRange returns a pair of paths to a min/max Refs having a given prefix
func (t *Dict) FindPathRange(prefix []byte) (min, max *RefPath) {
	// test empty tree
	if t.Empty() {
		return
	}
	// descend to the closest node/leaf 
	lpref := len(prefix)
	ref   := &t.root
	dir   := byte(1)
	path  := NewRefPath()
	path.Append(ref, dir)

	if lpref > 0 {
		for ref.node != nil {
			append := ref.node.off < lpref
			dir = ref.node.dir(prefix)
			ref = &ref.node.child[dir]
			if append {
				path.Append(ref, dir)
			}
		}

		// check the prefix for match
		if len(ref.Key) < lpref {
			return
		}

		for i, b := range prefix {
			if ref.Key[i] != b {
				return
			}
		}
	}

	min = path
	max = path.Copy()

	if path.GetLeaf() != nil {
		// we only have a single leaf
		return
	}

	top := path.Refs[len(path.Refs)-1]

	// descend to the leftmost leaf
	ref = top
	for ref.node != nil {
		ref = &ref.node.child[0]
		min.Append(ref, 0)
	}

	// descend to the rightmost leaf
	ref = top
	for ref.node != nil {
		ref = &ref.node.child[1]
		max.Append(ref, 1)
	}

	return
}


// Iter calls a handler for all keys with a given prefix.
// It returns whether all prefixed keys were iterated.
// The handler can continue the process by returning true or abort with false.
func (t *Dict) Iter(prefix []byte, handler func(Item) bool) bool {
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
func (t *Dict) iterate(p Ref, h func(Item) bool) bool {
	if p.node != nil {
		return t.iterate(p.node.child[0], h) && t.iterate(p.node.child[1], h)
	}
	return h(p.Item)
}

// Keys returns all keys, as a slice of []byte, in a sorted order.
func (t *Dict) Keys() [][]byte {
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

// Items returns a []Item slice sorted by count (descending)
func (t *Dict) Items() (items ItemSlice) {
	// empty tree?
	if t.Empty() {
		return items
	}

	items = make(ItemSlice, 0, t.size)

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
			items = append(items, p.Item)
		} else {
			// unshift the children and continue
			to_visit = append(to_visit, &p.node.child[1], &p.node.child[0])
		}
	}

	return items
}

func (t *Dict) DebugDump() {
	t.debug_dump(&t.root, "T:", 0, "")
}

func (t *Dict) debug_dump(ref *Ref, tag string, off int, indent string) {
	if ref.node == nil {
		critbyte := "  [        ]"
		if off < len(ref.Key) {
			critbyte = fmt.Sprintf("%c [%08b]", ref.Key[off], ref.Key[off])
		}
		fmt.Printf("%s%s LEAF byte=%s key=%q val=%v\n", indent, tag, critbyte, ref.Key, ref.Val)
	} else {
		fmt.Printf("%s%s NODE off=%v mask=%08b\n", indent, tag, ref.node.off, ref.node.bit)

		t.debug_dump(&ref.node.child[0], "L:", ref.node.off, indent + "  ")
		t.debug_dump(&ref.node.child[1], "R:", ref.node.off, indent + "  ")
	}
}

