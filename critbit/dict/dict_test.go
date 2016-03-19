package dict

import "testing"
import "bytes"

func keys(tr *Dict) (s [][]byte) {
	tr.Iter(nil, func(item Item) bool {
		s = append(s, item.Key)
		return true
	})
	return
}

func Test_EmptyDict(t *testing.T) {
	tr := NewDict()
	if keys(tr) != nil {
		t.Error("must be empty")
	}
	if _, ok := tr.Get([]byte("a")); ok {
		t.Errorf("wrong .Get() result: expected false, got %v", ok)
	}
	if old := tr.Del([]byte("a")); old != nil {
		t.Errorf("wrong .Del() result: expected nil, got %v", old)
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
		tr := NewDict()
		for _, s := range test.ins {
			t.Logf("inserting %v\n", s)
			tr.Set([]byte(s), 1)
			var v interface{}
			var ok bool
			if v, ok = tr.Get([]byte(s)); v == 1 && ok {
				continue
			}
			t.Errorf("test %d: wrong .Get(%q) result, expected (1, true), got (%v, %v)", i, s, v, ok)
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
			var c interface{}
			if c = tr.Del(res[j]); c == 1 {
				continue
			}
			t.Errorf("test %d: wrong .Del(%q) result, expected 1, got %v", i, res[j], c)
			return
		}
	}
}

func Test_DeleteUnknownKey(t *testing.T) {
	tr := NewDict()
	if c := tr.Set([]byte("aa"), 2); c != nil {
		t.Error("wrong result when setting into an empty tree: %v", c)
	}
	if c := tr.Del([]byte("ab")); c != nil {
		t.Errorf("wrong result when deleting an unknown key: %v", c)
	}
}

func Test_Iter(t *testing.T) {
	tr := NewDict()
	keys := []string{"aa", "aaa", "aab", "ab", "ba", "bb", "bba", "bbb"}

	for _, s := range keys {
		tr.Set([]byte(s), 5)
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
		tr.Iter([]byte(test.prefix), func(item Item) bool {
			if len(s) < 1 {
				t.Errorf("test %d: superfluous key %q", i, string(item.Key))
				return true
			}
			if ! bytes.Equal([]byte(s[0]), item.Key) {
				t.Errorf("test %d: got key %q, expected %q", i, string(item.Key), s[0])
			}
			if item.Val != 5 {
				t.Errorf("test %d: got val %v, expected 5", i, item.Val)
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
	tr := NewDict()
	expected := [][]byte{}

	returned_keys := tr.Keys()
	if ! testKeysEq(returned_keys, expected) {
		t.Errorf("Got: %q", returned_keys)
	}
}

func Test_Keys1(t *testing.T) {
	tr := NewDict()
	orig_keys := []string{"aa"}
	expected := [][]byte{[]byte("aa")}

	for _, s := range orig_keys {
		tr.Set([]byte(s), "1")
	}
	returned_keys := tr.Keys()
	if ! testKeysEq(returned_keys, expected) {
		t.Errorf("Got: %q", returned_keys)
	}
}

func Test_KeysMany(t *testing.T) {
	tr := NewDict()
	orig_keys := []string{"zz", "dd", "yy", "cc", "xx", "bb", "ww", "aa"}
	expected := [][]byte{
		[]byte("aa"), []byte("bb"), []byte("cc"), []byte("dd"),
		[]byte("ww"), []byte("xx"), []byte("yy"), []byte("zz")}

	for _, s := range orig_keys {
		tr.Set([]byte(s), "2")
	}
	returned_keys := tr.Keys()
	if ! testKeysEq(returned_keys, expected) {
		t.Errorf("Got: %q", returned_keys)
	}
}

func Test_Items(t *testing.T) {
	tr := NewDict()
	keys := []string{"a","b","c","a","bb","ccc","a","ccc"}
	expected := ItemSlice{
		{[]byte("a"), 6}, {[]byte("b"), 1}, {[]byte("bb"), 4},
		{[]byte("c"), 2}, {[]byte("ccc"),7},
	}

	for i, s := range keys {
		tr.Set([]byte(s), i)
	}

	items := tr.Items()
	if len(items) != len(expected) {
		t.Errorf("wrong number of items: expected %v, got %v", len(expected), len(items))
	}
	for i, item := range items {
		exp := expected[i]
		if ! bytes.Equal(item.Key, exp.Key) {
			t.Errorf("keys don't match: expected %q, got %q", string(exp.Key), string(item.Key))
		}
		if item.Val != exp.Val {
			t.Errorf("vals of %q don't match: expected %v, got %v", string(item.Key), exp.Val, item.Val)
		}
	}
}

func Test_Merge(t *testing.T) {
	a := NewDict(ItemSlice{{[]byte("ABC"),"@"}, {[]byte("DEF"),'H'}}...)
	b := NewDict(ItemSlice{{[]byte("ABC"),-1},  {[]byte("GHI"),0.3}}...)

	expected := ItemSlice{
		{[]byte("ABC"), -1}, {[]byte("DEF"), 'H'}, {[]byte("GHI"), 0.3},
	}

	a.Merge(b, nil)
	items := a.Items()

	if len(items) != len(expected) {
		t.Errorf("wrong number of items: expected %v, got %v", len(expected), len(items))
	}
	for i, item := range items {
		exp := expected[i]
		if ! bytes.Equal(item.Key, exp.Key) {
			t.Errorf("keys don't match: expected %q, got %q", string(exp.Key), string(item.Key))
		}
		if item.Val != exp.Val {
			t.Errorf("vals of %q don't match: expected %v, got %v", string(item.Key), exp.Val, item.Val)
		}
	}
}
