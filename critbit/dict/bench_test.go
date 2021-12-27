// Copyright 2013 Martin Schnabel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dict

import (
	"os"
	"reflect"
	"sort"
	"testing"
	"text/scanner"
	"unsafe"

	"github.com/brianvoe/gofakeit/v6"
)

func BytesToString(b []byte) string {
    bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))
    sh := reflect.StringHeader{bh.Data, bh.Len}
    return *(*string)(unsafe.Pointer(&sh))
}

func StringToBytes(s string) []byte {
    sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
    bh := reflect.SliceHeader{sh.Data, sh.Len, 0}
    return *(*[]byte)(unsafe.Pointer(&bh))
}


var words, tests []string

func initdata(b *testing.B) {
	if words != nil {
		return
	}
	var err error
	words, err = scan("dict.go")
	if err != nil {
		b.Fatal(err)
	}
	tests, err = scan("dict_test.go")
	if err != nil {
		b.Fatal(err)
	}
	b.Logf("data size:  words %v, tests %v", len(words), len(tests))
}

func scan(name string) (w []string, err error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var s scanner.Scanner
	s.Init(f)
	for t := s.Scan(); t != scanner.EOF; t = s.Scan() {
		w = append(w, s.TokenText())
	}
	return
}

func BenchmarkMap(b *testing.B) {
	initdata(b)
	var count int
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := make(map[string]struct{})
		for _, w := range words {
			m[w] = struct{}{}
		}
		count = 0
		for _, w := range tests {
			if _, ok := m[w]; ok {
				count++
			}
		}
	}
}

func BenchmarkTree(b *testing.B) {
	initdata(b)
	var count int
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewDict()
		for _, w := range words {
			t.Set(StringToBytes(w), nil)
		}
		count = 0
		for _, w := range tests {
			if _, ok := t.Get(StringToBytes(w)); ok {
				count++
			}
		}
	}
}

func BenchmarkMapSort(b *testing.B) {
	initdata(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := make(map[string]struct{})
		for _, w := range words {
			m[w] = struct{}{}
		}
		s := make([]string, 0, len(m))
		for w := range m {
			s = append(s, w)
		}
		sort.Strings(s)
	}
}

func BenchmarkGoMap_Set(b *testing.B) {
	var (
		keys = getKeys(b.N)
		m    = make(map[string]interface{})
	)

	b.ResetTimer()

	for i, key := range keys {
		m[key] = i
	}
}

func BenchmarkGoMap_Get(b *testing.B) {
	var (
		keys = getKeys(b.N)
		m    = make(map[string]interface{})
	)

	for i, key := range keys {
		m[key] = i
	}

	b.ResetTimer()

	for _, key := range keys {
		_ = m[key]
	}
}

func BenchmarkTreeSort(b *testing.B) {
	initdata(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewDict()
		for _, w := range words {
			t.Set(StringToBytes(w), nil)
		}
		//s := make([]string, 0, t.Len())
		t.Iter(nil, func(item Item) bool {
			//s = append(s, key)
			return true
		})
	}
}

func BenchmarkDict_Set(b *testing.B) {
	var (
		keys = getKeys(b.N)
		dict = NewDict()
	)

	b.ResetTimer()

	for i, key := range keys {
		dict.Set(StringToBytes(key), i)
	}
}

func BenchmarkDict_Get(b *testing.B) {
	var (
		keys = getKeys(b.N)
		dict = NewDict()
	)

	for i, key := range keys {
		dict.Set(StringToBytes(key), i)
	}

	b.ResetTimer()

	for _, key := range keys {
		_, _ = dict.Get(StringToBytes(key))
	}
}

func getKeys(total int) []string {
	const seed = 1234567890

	var (
		faker = gofakeit.New(seed)
		keys  = make([]string, total)
	)

	for i := range keys {
		keys[i] = faker.Sentence(4)
	}

	return keys
}
