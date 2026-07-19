// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package parser_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestDeferBlock(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "deferblock.go", `package p
func f() {
	defer { cleanup() }
}
`, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	fn := file.Decls[0].(*ast.FuncDecl)
	stmt := fn.Body.List[0].(*ast.DeferStmt)
	call := stmt.Call
	lit, ok := call.Fun.(*ast.FuncLit)
	if !ok || len(call.Args) != 0 || call.Lparen.IsValid() {
		t.Fatalf("defer block call = %#v", call)
	}
	if lit.Type.Func.IsValid() {
		t.Fatalf("synthetic func token has source position %v", lit.Type.Func)
	}
	if len(lit.Body.List) != 1 {
		t.Fatalf("defer block has %d statements, want 1", len(lit.Body.List))
	}
}
