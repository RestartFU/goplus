// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package format_test

import (
	"go/format"
	"testing"
)

func TestDeferBlock(t *testing.T) {
	src := []byte("package p\nfunc f(){defer {cleanup()}}\n")
	want := "package p\n\nfunc f() {\n\tdefer {\n\t\tcleanup()\n\t}\n}\n"
	got, err := format.Source(src)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("formatted defer block:\n%s\nwant:\n%s", got, want)
	}
}
