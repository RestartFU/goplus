// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"
)

func TestWalkEnum(t *testing.T) {
	const src = `package p

type Result[T any] enum {
	Ok { value T }
	None
}
`
	file, err := parser.ParseFile(token.NewFileSet(), "enum.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	var variants int
	new(File).walk(file, ctxProg, func(_ *File, x any, _ astContext) {
		if _, ok := x.(*ast.EnumVariant); ok {
			variants++
		}
	})
	if variants != 2 {
		t.Fatalf("walk visited %d variants, want 2", variants)
	}
}

func TestWalkTry(t *testing.T) {
	const src = `package p
func load() (int, error) { return 0, nil }
func get() (int, error) {
	try value := load()
	return value, nil
}
`
	file, err := parser.ParseFile(token.NewFileSet(), "try.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	var tries int
	new(File).walk(file, ctxProg, func(_ *File, x any, _ astContext) {
		if _, ok := x.(*ast.TryStmt); ok {
			tries++
		}
	})
	if tries != 1 {
		t.Fatalf("walk visited %d try statements, want 1", tries)
	}
}
