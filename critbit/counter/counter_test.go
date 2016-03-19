package counter

import "testing"
import "bytes"

func keys(tr *Counter) (s [][]byte) {
	tr.Iter(nil, func(ckey CountedKey) bool {
		s = append(s, ckey.Key)
		return true
	})
	return
}

func Test_EmptyCounter(t *testing.T) {
	tr := NewCounter()
	if keys(tr) != nil {
		t.Error("must be empty")
	}
	if c := tr.Get([]byte("a")); c != 0 {
		t.Errorf("wrong .Get() result: expected 0, got %v", c)
	}
	if c := tr.Del([]byte("a")); c != 0 {
		t.Errorf("wrong .Del() result: expected 0, got %v", c)
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
		tr := NewCounter()
		for _, s := range test.ins {
			t.Logf("inserting %v\n", s)
			tr.Inc([]byte(s))
			if tr.Get([]byte(s)) > 0 {
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
			var c int
			if c = tr.Del(res[j]); c > 0 {
				continue
			}
			t.Errorf("test %d: delete %q returned %d", i, res[j], c)
			return
		}
	}
}

func Test_DeleteUnknownKey(t *testing.T) {
	tr := NewCounter()
	if c := tr.Inc([]byte("aa")); c != 1 {
		t.Error("wrong result when increment a key in an empty tree: %v", c)
	}
	if c := tr.Del([]byte("ab")); c != 0 {
		t.Errorf("wrong result when deleting an unknown key: %v", c)
	}
}

func Test_Iter(t *testing.T) {
	tr := NewCounter()
	keys := []string{"aa", "aaa", "aab", "ab", "ba", "bb", "bba", "bbb"}

	for _, s := range keys {
		tr.IncBy([]byte(s), 5)
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
		tr.Iter([]byte(test.prefix), func(ckey CountedKey) bool {
			if len(s) < 1 {
				t.Errorf("test %d: superfluous key %q", i, string(ckey.Key))
				return true
			}
			if ! bytes.Equal([]byte(s[0]), ckey.Key) {
				t.Errorf("test %d: got key %q, expected %q", i, string(ckey.Key), s[0])
			}
			if ckey.Count != 5 {
				t.Errorf("test %d: got count %v, expected 5", i, ckey.Count)
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
	tr := NewCounter()
	expected := [][]byte{}

	returned_keys := tr.Keys()
	if ! testKeysEq(returned_keys, expected) {
		t.Errorf("Got: %q", returned_keys)
	}
}

func Test_Keys1(t *testing.T) {
	tr := NewCounter()
	orig_keys := []string{"aa"}
	expected := [][]byte{[]byte("aa")}

	for _, s := range orig_keys {
		tr.Inc([]byte(s))
	}
	returned_keys := tr.Keys()
	if ! testKeysEq(returned_keys, expected) {
		t.Errorf("Got: %q", returned_keys)
	}
}

func Test_KeysMany(t *testing.T) {
	tr := NewCounter()
	orig_keys := []string{"zz", "dd", "yy", "cc", "xx", "bb", "ww", "aa"}
	expected := [][]byte{
		[]byte("aa"), []byte("bb"), []byte("cc"), []byte("dd"),
		[]byte("ww"), []byte("xx"), []byte("yy"), []byte("zz")}

	for _, s := range orig_keys {
		tr.Inc([]byte(s))
	}
	returned_keys := tr.Keys()
	if ! testKeysEq(returned_keys, expected) {
		t.Errorf("Got: %q", returned_keys)
	}
}

func Test_CountedKeys(t *testing.T) {
	tr := NewCounter()
	keys := []string{"a","b","c","a","bb","ccc","a","ccc"}
	expected := CountedKeySlice{
		{[]byte("a"), 3}, {[]byte("ccc"),2}, {[]byte("b"), 1},
		{[]byte("bb"),1}, {[]byte("c"),  1},
	}

	for _, s := range keys {
		tr.Inc([]byte(s))
	}
	sorted := tr.CountedKeys()
	if len(sorted) != len(expected) {
		t.Errorf("wrong number of counted keys: expected %v, got %v", len(expected), len(sorted))
	}
	for i, ckey := range sorted {
		exp := expected[i]
		if ! bytes.Equal(ckey.Key, exp.Key) {
			t.Errorf("keys don't match: expected %q, got %q", string(exp.Key), string(ckey.Key))
		}
		if ckey.Count != exp.Count {
			t.Errorf("counts of %q don't match: expected %v, got %v", string(ckey.Key), exp.Count, ckey.Count)
		}
	}
}

func Test_Merge(t *testing.T) {
	a := NewCounter(CountedKeySlice{{[]byte("ABC"),3},  {[]byte("DEF"),2}}...)
	b := NewCounter(CountedKeySlice{{[]byte("ABC"),-1}, {[]byte("GHI"),1}}...)

	expected := CountedKeySlice{
		{[]byte("ABC"), 2}, {[]byte("DEF"), 2}, {[]byte("GHI"), 1},
	}

	a.Merge(b, nil)
	sorted := a.CountedKeys()

	if len(sorted) != len(expected) {
		t.Errorf("wrong number of counted keys: expected %v, got %v", len(expected), len(sorted))
	}
	for i, ckey := range sorted {
		exp := expected[i]
		if ! bytes.Equal(ckey.Key, exp.Key) {
			t.Errorf("keys don't match: expected %q, got %q", string(exp.Key), string(ckey.Key))
		}
		if ckey.Count != exp.Count {
			t.Errorf("counts of %q don't match: expected %v, got %v", string(ckey.Key), exp.Count, ckey.Count)
		}
	}
}
