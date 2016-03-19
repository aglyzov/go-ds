package dict


type RefPath struct {
	Refs, LastRefs []*Ref
	Dirs, LastDirs []uint64  // bitmap
}

func NewRefPath() *RefPath {
	return &RefPath{
		Refs : make([]*Ref, 0, 21),
		Dirs : make([]uint64, 1),
	}
}

func (path *RefPath) GetLeaf() (leaf *Ref) {
	if path == nil {
		return
	}
	num := len(path.Refs)
	if num == 0 {
		return
	}
	leaf = path.Refs[num-1]
	if leaf.node != nil {
		return nil
	}
	return
}
func (path *RefPath) Append(ref *Ref, dir byte) {
	//fmt.Printf("Append(%v, %v)\n", ref, dir)
	idx := uint64(len(path.Refs))
	off := idx >> 6
	path.Refs = append(path.Refs, ref)
	if off >= uint64(len(path.Dirs)) {
		// extend bitmap
		path.Dirs = append(path.Dirs, uint64(0))
	}
	if dir > 0 {
		path.Dirs[off] |= uint64(1) << (idx & 0x3F)  // 3F == 0011 1111
	}
}
func (path *RefPath) Pop() (ref *Ref, dir byte) {
	num := len(path.Refs)
	if num == 0 {
		//fmt.Printf("Pop() -> %v, %v\n", ref, dir)
		return
	}
	idx := uint64(num - 1)
	off := idx >> 6	   // byte index
	bit := idx & 0x3F  // bit  index
	ref = path.Refs[idx]
	dir = byte((path.Dirs[off] >> bit) & 1)
	path.Dirs[off] &= (uint64(1) << bit) - 1  // turn off all bits above
	path.Refs = path.Refs[:idx]
	//fmt.Printf("Pop() -> %v, %v\n", ref, dir)
	return
}
func (path *RefPath) Copy() *RefPath {
	new := RefPath{
		Refs : make([]*Ref,   len(path.Refs), cap(path.Refs)),
		Dirs : make([]uint64, len(path.Dirs), cap(path.Dirs)),
	}
	// copy data
	copy(new.Refs, path.Refs)
	copy(new.Dirs, path.Dirs)

	return &new
}
func (path *RefPath) Backup() {
	// make sure the last-ref slice has enough room
	n_refs := len(path.Refs)
	if l := len(path.LastRefs); l < n_refs {
		if c := cap(path.LastRefs); c < n_refs {
			C := int(float64(n_refs) * 1.5)
			if C < 21 {C = 21}
			//fmt.Printf("Backup(): allocating a larger Refs block: %v(%v) -> %v(%v)\n", l, c, n_refs, C)
			path.LastRefs = make([]*Ref, n_refs, C)  // alloc a larger block
		}
	}
	path.LastRefs = path.LastRefs[:n_refs]  // set the upper bound

	// make sure the last-dir slice has enough room
	n_dirs := len(path.Dirs)
	if l := len(path.LastDirs); l < n_dirs {
		if c := cap(path.LastDirs); c < n_dirs {
			C := n_dirs + 1
			//fmt.Printf("Backup(): allocating a larger Dirs block: %v(%v) -> %v(%v)\n", l, c, n_dirs, C)
			path.LastDirs = make([]uint64, n_dirs, C)  // alloc a larger block
		}
	}
	path.LastDirs = path.LastDirs[:n_dirs]  // extend the upper bound

	// copy data
	copy(path.LastRefs, path.Refs)
	copy(path.LastDirs, path.Dirs)
}
func (path *RefPath) Revert() *Ref {
	path.Refs, path.LastRefs = path.LastRefs, path.Refs
	path.Dirs, path.LastDirs = path.LastDirs, path.Dirs
	if path.LastRefs != nil {
		path.LastRefs = path.LastRefs[:0]
		path.LastDirs = path.LastDirs[:0]
	}
	return path.GetLeaf()
}
func (path *RefPath) TrackNext() (ref *Ref) {
	// discard current leaf
	_, dir := path.Pop()
	// keep ascending while dir is 1 (we were in a right branch)
	ref, ndir := path.Pop()
	for ref != nil && dir == 1 {
		dir = ndir
		ref, ndir = path.Pop()
	}
	// descend one time to the right branch
	if ref != nil && ref.node != nil {
		path.Append(ref, ndir)
		ref = &ref.node.child[1]
		path.Append(ref, 1)
	}
	// keep descending to the left branches (dir is 0)
	for ref != nil && ref.node != nil {
		ref = &ref.node.child[0]
		path.Append(ref, 0)
	}
	//fmt.Printf("TrackNext() -> %v\n", ref)
	return
}
func (path *RefPath) TrackPrev() (ref *Ref) {
	// discard current leaf
	_, dir := path.Pop()
	// keep ascending while dir is 0 (we were in a left branch)
	ref, ndir := path.Pop()
	for ref != nil && dir == 0 {
		dir = ndir
		ref, ndir = path.Pop()
	}
	// descend one time to the left branch
	if ref != nil && ref.node != nil {
		path.Append(ref, ndir)
		ref = &ref.node.child[0]
		path.Append(ref, 0)
	}
	// keep descending to the right branches (dir is 1)
	for ref != nil && ref.node != nil {
		ref = &ref.node.child[1]
		path.Append(ref, 1)
	}
	return
}
