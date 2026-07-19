// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syntax

import (
	"strings"
	"testing"
)

func TestDeferBlock(t *testing.T) {
	const src = `package p
func f() {
	defer {
		cleanup()
	}
}
`
	file, err := Parse(NewFileBase("deferblock.go"), strings.NewReader(src), nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	fn := file.DeclList[0].(*FuncDecl)
	stmt := fn.Body.List[0].(*CallStmt)
	call, ok := stmt.Call.(*CallExpr)
	if !stmt.DeferBlock || !ok {
		t.Fatalf("defer block = %#v", stmt)
	}
	lit, ok := call.Fun.(*FuncLit)
	if !ok || len(call.ArgList) != 0 || len(lit.Body.List) != 1 {
		t.Fatalf("defer block call = %#v", call)
	}
	const want = "package p; func f() { defer { cleanup() } }"
	if got := String(file); got != want {
		t.Fatalf("printed defer block: %q, want %q", got, want)
	}
}
