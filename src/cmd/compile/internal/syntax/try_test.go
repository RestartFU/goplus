// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syntax

import (
	"bytes"
	"strings"
	"testing"
)

func TestTrySyntax(t *testing.T) {
	const src = `package p
func load() (string, int, error) { return "", 0, nil }
func get() (string, int, error) {
	try value, count := load()
	return value, count, nil
}
`

	file, err := Parse(NewFileBase("try.go"), strings.NewReader(src), nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	fn := file.DeclList[1].(*FuncDecl)
	stmt, ok := fn.Body.List[0].(*TryStmt)
	if !ok {
		t.Fatalf("statement has type %T, want *TryStmt", fn.Body.List[0])
	}
	if got := len(UnpackListExpr(stmt.Lhs)); got != 2 {
		t.Fatalf("try binds %d values, want 2", got)
	}

	var printed bytes.Buffer
	if _, err := Fprint(&printed, file, 0); err != nil {
		t.Fatal(err)
	}
	const want = `package p

func load() (string, int, error) {
	return "", 0, nil
}

func get() (string, int, error) {
	try value, count := load()
	return value, count, nil
}`
	if got := printed.String(); got != want {
		t.Errorf("printed try statement:\n%s\nwant:\n%s", got, want)
	}
}

func TestTryIsContextualKeyword(t *testing.T) {
	tests := []string{
		"try := 1",
		"try++",
		"try()",
		"_ = try",
	}
	for _, stmt := range tests {
		t.Run(stmt, func(t *testing.T) {
			src := "package p\nfunc f() { " + stmt + " }\n"
			if _, err := Parse(NewFileBase("try.go"), strings.NewReader(src), nil, nil, 0); err != nil {
				t.Fatal(err)
			}
		})
	}
}
