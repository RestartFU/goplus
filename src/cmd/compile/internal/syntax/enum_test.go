// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syntax

import (
	"bytes"
	"strings"
	"testing"
)

func TestEnumSyntax(t *testing.T) {
	const src = `package p

enum Result[T any] {
	Ok {
		value T
	}
	Err {
		err error
	}
	None
	Empty {}
}
`

	file, err := Parse(NewFileBase("enum.go"), strings.NewReader(src), nil, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(file.DeclList) != 1 {
		t.Fatalf("got %d declarations, want 1", len(file.DeclList))
	}
	decl, ok := file.DeclList[0].(*EnumDecl)
	if !ok {
		t.Fatalf("declaration has type %T, want *EnumDecl", file.DeclList[0])
	}
	if decl.Name.Value != "Result" || len(decl.TParamList) != 1 {
		t.Fatalf("enum header = %s with %d type parameters", decl.Name.Value, len(decl.TParamList))
	}
	if len(decl.VariantList) != 4 {
		t.Fatalf("got %d variants, want 4", len(decl.VariantList))
	}
	if decl.VariantList[2].HasPayload {
		t.Error("None unexpectedly has a payload")
	}
	if empty := decl.VariantList[3]; !empty.HasPayload || len(empty.FieldList) != 0 {
		t.Errorf("Empty payload = (%t, %d fields), want explicit empty payload", empty.HasPayload, len(empty.FieldList))
	}

	var variants int
	Inspect(file, func(n Node) bool {
		if _, ok := n.(*EnumVariant); ok {
			variants++
		}
		return true
	})
	if variants != 4 {
		t.Errorf("Inspect visited %d variants, want 4", variants)
	}

	var printed bytes.Buffer
	if _, err := Fprint(&printed, file, 0); err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSuffix(src, "\n")
	if got := printed.String(); got != want {
		t.Errorf("printed enum:\n%q\nwant:\n%q", got, want)
	}
	if _, err := Parse(NewFileBase("printed.go"), strings.NewReader(printed.String()), nil, nil, 0); err != nil {
		t.Fatalf("printed enum does not parse: %v", err)
	}
}
