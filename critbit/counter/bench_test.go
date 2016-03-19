// Copyright 2013 Martin Schnabel. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package counter

import (
	"os"
	"sort"
	"testing"
	"text/scanner"
)

import (
    "reflect"
    "unsafe"
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
	words, err = scan("counter.go")
	if err != nil {
		b.Fatal(err)
	}
	tests, err = scan("counter_test.go")
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
		t := NewCounter()
		for _, w := range words {
			t.Inc(StringToBytes(w))
		}
		count = 0
		for _, w := range tests {
			if t.Get(StringToBytes(w)) == 1 {
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

func BenchmarkTreeSort(b *testing.B) {
	initdata(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		t := NewCounter()
		for _, w := range words {
			t.Inc(StringToBytes(w))
		}
		//s := make([]string, 0, t.Len())
		t.Iter(nil, func(ckey CountedKey) bool {
			//s = append(s, key)
			return true
		})
	}
}
