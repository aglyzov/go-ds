package set

import "testing"
import "bytes"

func keys(tr *Set) (s [][]byte) {
	tr.Iter(nil, func(key []byte) bool {
		s = append(s, key)
		return true
	})
	return
}

func Test_EmptySet(t *testing.T) {
	tr := NewSet()
	if keys(tr) != nil {
		t.Error("must be empty")
	}
	if tr.Has([]byte("a")) {
		t.Errorf("wrong .Has() result: expected false, got true")
	}
	if tr.Del([]byte("a")) {
		t.Errorf("wrong .Del() result: expected false, got true")
	}
}

func Test_KeyOrder(t *testing.T) {
	tests := []struct {
		ins []string
		res []string
	}{
		{
			[]string{"x", "y", "z", "c", "c", "b", "b", "a", "a"},
			[]string{"a", "b", "c", "x", "y", "z"},
		},
		{
			[]string{"aaa", "aa", "a"},
			[]string{"a", "aa", "aaa"},
		},
		{
			[]string{"b", "a", "aa"},
			[]string{"a", "aa", "b"},
		},
		{
			[]string{"aa", "aaa", "aab", "ab", "ba", "bb", "bba", "bbb"},
			[]string{"aa", "aaa", "aab", "ab", "ba", "bb", "bba", "bbb"},
		},
	}
	for i, test := range tests {
		tr := NewSet()
		for _, s := range test.ins {
			t.Logf("inserting %v\n", s)
			tr.Add([]byte(s))
			if tr.Has([]byte(s)) {
				continue
			}
			t.Errorf("test %d: counter of %q is 0 after increment", i, s)
			return
		}
		res := keys(tr)
		if len(res) != len(test.res) || tr.Len() != len(test.res) {
			t.Errorf("test %d unexpected length %d", i, len(res))
			return
			//continue
		}
		for j, s := range test.res {
			t.Logf("checking %v\n", s)
			if bytes.Equal(res[j], []byte(s)) {
				continue
			}
			t.Errorf("test %d unexpected element %q at %d", i, res[j], j)
			return
		}
		for j := len(res) - 1; j >= 0; j-- {
			t.Logf("deleting %s\n", res[j])
			if tr.Del(res[j]) {
				continue
			}
			t.Errorf("test %d: delete %q returned false", i, res[j])
			return
		}
	}
}

func Test_DeleteUnknownKey(t *testing.T) {
	tr := NewSet()
	if ! tr.Add([]byte("aa")) {
		t.Error("wrong result when adding a key to an empty tree, expected true, got false")
	}
	if tr.Del([]byte("ab")) {
		t.Errorf("wrong result when deleting an unknown key, expected false, got true")
	}
}

func Test_Iter(t *testing.T) {
	tr := NewSet()
	keys := []string{"aa", "aaa", "aab", "ab", "ba", "bb", "bba", "bbb"}

	for _, s := range keys {
		tr.Add([]byte(s))
	}
	tests := []struct {
		prefix string
		keys   []string
	}{
		{"", keys},
		{"a", []string{"aa", "aaa", "aab", "ab"}},
		{"aa", []string{"aa", "aaa", "aab"}},
		{"aaa", []string{"aaa"}},
		{"aaaa", nil},
		{"c", nil},
	}
	for i, test := range tests {
		s := test.keys
		tr.Iter([]byte(test.prefix), func(key []byte) bool {
			if len(s) < 1 {
				t.Errorf("test %d: superfluous key %q", i, string(key))
				return true
			}
			if ! bytes.Equal([]byte(s[0]), key) {
				t.Errorf("test %d: got key %q, expected %q", i, string(key), s[0])
			}
			s = s[1:]
			return true
		})
	}
}

func testKeysEq(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if ! bytes.Equal(a[i], b[i]) {
			return false
		}
	}

	return true
}

func Test_Keys0(t *testing.T) {
	tr := NewSet()
	expected := [][]byte{}

	returned_keys := tr.Keys()
	if ! testKeysEq(returned_keys, expected) {
		t.Errorf("Got: %q", returned_keys)
	}
}

func Test_Keys1(t *testing.T) {
	tr := NewSet()
	orig_keys := []string{"aa"}
	expected := [][]byte{[]byte("aa")}

	for _, s := range orig_keys {
		tr.Add([]byte(s))
	}
	returned_keys := tr.Keys()
	if ! testKeysEq(returned_keys, expected) {
		t.Errorf("Got: %q", returned_keys)
	}
}

func Test_KeysMany(t *testing.T) {
	tr := NewSet()
	orig_keys := []string{"zz", "dd", "yy", "cc", "xx", "bb", "ww", "aa"}
	expected := [][]byte{
		[]byte("aa"), []byte("bb"), []byte("cc"), []byte("dd"),
		[]byte("ww"), []byte("xx"), []byte("yy"), []byte("zz")}

	for _, s := range orig_keys {
		tr.Add([]byte(s))
	}
	returned_keys := tr.Keys()
	if ! testKeysEq(returned_keys, expected) {
		t.Errorf("Got: %q", returned_keys)
	}
}

func Test_Merge(t *testing.T) {
	a := NewSet([]byte("ABC"), []byte("DEF"))
	b := NewSet([]byte("ABC"), []byte("GHI"))

	expected := [][]byte{
		[]byte("ABC"), []byte("DEF"), []byte("GHI"),
	}

	a.Merge(b, nil)
	keys := a.Keys()

	if len(keys) != len(expected) {
		t.Errorf("wrong number of counted keys: expected %v, got %v", len(expected), len(keys))
	}
	for i, key := range keys {
		exp := expected[i]
		if ! bytes.Equal(key, exp) {
			t.Errorf("keys don't match: expected %q, got %q", string(exp), string(key))
		}
	}
}
