package set

import "testing"

func Test_EmptySetHas(t *testing.T) {
	s := NewSet()
	if s.Has(0) {
		t.Error("s.Has(0) returned true on an empty set")
	}
	if s.Has(1234567890) {
		t.Error("s.Has(1234567890) returned true on an empty set")
	}
	if s.Has(0xFFFFFFFFFFFFFFFF) {
		t.Error("s.Has(0xFFFFFFFFFFFFFFFF) returned true on an empty set")
	}
}

func Test_AdhocSetHas(t *testing.T) {
	s := NewSet()

	// prepare an ad-hoc set with three entries: 0, 2 and 255
	node := s.root
	for i := 0; i < 7; i++ {
		node.bitmap[0] = 0x1
		next := &Node{}
		node.children  = append(node.children, next)
		node = next
	}
	node.bitmap[0] = 0x0000000000000005 // 0000 .... 0000 0101
	node.bitmap[3] = 0x8000000000000000 // 1000 0000 .... 0000

	if ! s.Has(0) {
		t.Error("s.Has(0) returned false")
	}
	if ! s.Has(2) {
		t.Error("s.Has(2) returned false")
	}
	if ! s.Has(255) {
		t.Error("s.Has(255) returned false")
	}

	if s.Has(1) {
		t.Error("s.Has(1) returned true")
	}
	if s.Has(1234567890) {
		t.Error("s.Has(1234567890) returned true")
	}
	if s.Has(0xFFFFFFFFFFFFFFFF) {
		t.Error("s.Has(0xFFFFFFFFFFFFFFFF) returned true")
	}
}

func Test_SetAdd(t *testing.T) {
	s := NewSet()

	// add 0
	if ! s.Add(0) {
		t.Error("s.Add(0) returned false the first time")
	}
	if s.Add(0) {
		t.Error("s.Add(0) returned true the second time")
	}
	if s.size != 1 {
		t.Errorf("s.size is not 1 as expected, instead: %v", s.size)
	}
	if b := s.root.bitmap[0]; b != 0x0000000000000001 {
		t.Errorf("s.root.bitmap[0] is not 1 as expected, instead: %#x", b)
	}
	if n := len(s.root.children); n != 1 {
		t.Errorf("len(s.root.children) is not 1 as expected, instead: %v", n)
	}

	// add 256
	if ! s.Add(256) {
		t.Error("s.Add(256) returned false the first time")
	}
	if b := s.root.bitmap[0]; b != 0x0000000000000001 {
		t.Errorf("s.root.bitmap[1] is not 1 as expected, instead: %#x", b)
	}
	if s.size != 2 {
		t.Errorf("s.size is not 2 as expected, instead: %v", s.size)
	}

	// add 0xFFFF FFFF FFFF FFFF
	if ! s.Add(0xFFFFFFFFFFFFFFFF) {
		t.Error("s.Add(0xFFFFFFFFFFFFFFFF) returned false the first time")
	}
	if b := s.root.bitmap[3]; b != 0x8000000000000000 {
		t.Errorf("s.root.bitmap[3] is not 0x8000000000000000 as expected, instead: %#x", b)
	}
	if s.size != 3 {
		t.Errorf("s.size is not 3 as expected, instead: %v", s.size)
	}
}
