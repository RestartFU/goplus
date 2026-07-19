// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestTryStmt(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{
			name: "valid",
			body: `func load() (int, error) { return 0, nil }
func get() (int, error) { try value := load(); return value, nil }`,
		},
		{
			name: "valid bool",
			body: `func load() (int, bool) { return 0, false }
func get() (int, bool) { try value := load(); return value, true }`,
		},
		{
			name: "valid map lookup",
			body: `func get(values map[string]int, key string) (int, bool) {
	try value := values[key]
	return value, true
}`,
		},
		{
			name: "final result is not error",
			body: `func load() (int, string) { return 0, "" }
func get() (int, error) { try value := load(); return value, nil }`,
			want: "final try result must have type error",
		},
		{
			name: "enclosing function has no final error",
			body: `func load() (int, error) { return 0, nil }
func get() int { try value := load(); return value }`,
			want: "try requires the enclosing function to return error as its final result",
		},
		{
			name: "bool propagation requires final bool",
			body: `func load() (int, bool) { return 0, false }
func get() (int, error) { try value := load(); return value, nil }`,
			want: "try requires the enclosing function to return bool as its final result",
		},
		{
			name: "no new user variable",
			body: `func load() (int, error) { return 0, nil }
func get() (int, error) { value := 0; try value := load(); return value, nil }`,
			want: "no new variables on left side of :=",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "try.go", "package p\n"+test.body, parser.SkipObjectResolution)
			if err != nil {
				t.Fatal(err)
			}
			var messages []string
			conf := Config{Error: func(err error) { messages = append(messages, err.Error()) }}
			_, err = conf.Check("p", fset, []*ast.File{file}, nil)
			if test.want == "" {
				if err != nil {
					t.Fatalf("Check failed: %v", err)
				}
				return
			}
			if got := strings.Join(messages, "\n"); !strings.Contains(got, test.want) {
				t.Fatalf("errors:\n%s\nwant substring %q", got, test.want)
			}
		})
	}
}

func TestTryStmtDoesNotExposeTemporary(t *testing.T) {
	const src = `package p
func load() (int, error) { return 0, nil }
func get() (int, error) { try value := load(); return value, nil }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "try.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	info := &Info{Scopes: make(map[ast.Node]*Scope)}
	if _, err := new(Config).Check("p", fset, []*ast.File{file}, info); err != nil {
		t.Fatal(err)
	}
	for _, scope := range info.Scopes {
		for _, name := range scope.Names() {
			if strings.HasPrefix(name, ".try") {
				t.Fatalf("compiler temporary %q leaked into user scope", name)
			}
		}
	}
}
