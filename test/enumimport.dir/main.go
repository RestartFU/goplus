// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "./enumlib"

func main() {
	if enumlib.NewResult().Or(0) != 7 {
		panic("non-generic imported enum method")
	}
	if enumlib.NewOption().Or(0) != 9 {
		panic("generic imported enum method")
	}
}
