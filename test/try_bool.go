// run

// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

func doubleLookup(values map[string]int, key string) (int, bool) {
	try value := values[key]
	return value * 2, true
}

func load(value int, ok bool) (int, bool) {
	return value, ok
}

func require(value int, ok bool) (int, bool) {
	try loaded := load(value, ok)
	return loaded, true
}

func main() {
	tests := []struct {
		name  string
		value int
		ok    bool
	}{
		{name: "function success", value: 42, ok: true},
		{name: "function failure", value: 0, ok: false},
	}
	for _, test := range tests {
		value, ok := require(42, test.ok)
		if value != test.value || ok != test.ok {
			panic(test.name)
		}
	}

	value, ok := doubleLookup(map[string]int{"answer": 21}, "answer")
	if value != 42 || !ok {
		panic("map success")
	}
	value, ok = doubleLookup(map[string]int{"answer": 21}, "missing")
	if value != 0 || ok {
		panic("map failure")
	}
}
