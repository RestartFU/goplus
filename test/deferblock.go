// run

// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

func deferred() (value int) {
	defer {
		value++
	}
	value = 1
	return
}

func recovered() (ok bool) {
	defer {
		ok = recover() == "boom"
	}
	panic("boom")
}

func returns() {
	defer {
		return
	}
}

func main() {
	if deferred() != 2 {
		panic("defer block")
	}
	if !recovered() {
		panic("defer block recover")
	}
	returns()
}
