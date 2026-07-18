// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"
)

func checkEnumPackage(t *testing.T, src string) (*types.Package, error) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "enum.go", src, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	return new(types.Config).Check("p", fset, []*ast.File{file}, nil)
}

func TestEnumTypes(t *testing.T) {
	const src = `package p

enum Result {
	Ok { value int }
	Err { err error }
	None
}

func (o Ok) Value() int { return o.value }
func makeResult() Result { return Ok{value: 42} }
`
	pkg, err := checkEnumPackage(t, src)
	if err != nil {
		t.Fatal(err)
	}
	result := pkg.Scope().Lookup("Result").Type().(*types.Named)
	ok := pkg.Scope().Lookup("Ok").Type().(*types.Named)
	errVariant := pkg.Scope().Lookup("Err").Type().(*types.Named)
	none := pkg.Scope().Lookup("None").Type().(*types.Named)

	if _, ok := result.Underlying().(*types.Interface); !ok {
		t.Fatalf("Result underlying type is %T, want *types.Interface", result.Underlying())
	}
	for _, variant := range []*types.Named{ok, errVariant, none} {
		if _, ok := variant.Underlying().(*types.Struct); !ok {
			t.Errorf("%s underlying type is %T, want *types.Struct", variant.Obj().Name(), variant.Underlying())
		}
		if !types.AssignableTo(variant, result) {
			t.Errorf("%s is not assignable to Result", variant.Obj().Name())
		}
	}
	if fields := ok.Underlying().(*types.Struct); fields.NumFields() != 1 || fields.Field(0).Name() != "value" {
		t.Fatalf("Ok fields = %s, want value int", fields)
	}
	if method, _, _ := types.LookupFieldOrMethod(ok, true, pkg, "Value"); method == nil {
		t.Error("method declared on variant Ok was not collected")
	}
	variants := result.EnumVariants()
	if len(variants) != 3 || variants[0].Obj().Name() != "Ok" || variants[1].Obj().Name() != "Err" || variants[2].Obj().Name() != "None" {
		t.Fatalf("Result variants = %v, want [Ok Err None]", variants)
	}
	if ok.EnumType() != result {
		t.Fatalf("Ok enum type = %v, want Result", ok.EnumType())
	}
	if types.AssignableTo(types.NewPointer(ok), result) {
		t.Error("*Ok is assignable to Result")
	}
}

func TestEnumTypeSwitch(t *testing.T) {
	const exhaustive = `package p
enum Result { Ok { value int }; Err { err error }; None }
func inspect(r Result) int {
	switch v := r.(type) {
	case Ok: return v.value
	case Err: return len(v.err.Error())
	case None: return 0
	case nil: return -1
	}
	return -1
}
`
	if _, err := checkEnumPackage(t, exhaustive); err != nil {
		t.Fatal(err)
	}

	const missing = `package p
enum Result { Ok; Err; None }
func inspect(r Result) { switch r.(type) { case Ok: } }
`
	_, err := checkEnumPackage(t, missing)
	if err == nil || !strings.Contains(err.Error(), "non-exhaustive enum switch on Result; missing Err, None, nil") {
		t.Fatalf("non-exhaustive switch error = %v", err)
	}

	const withDefault = `package p
enum Result { Ok; Err }
func inspect(r Result) { switch r.(type) { case Ok:; default: } }
`
	if _, err := checkEnumPackage(t, withDefault); err != nil {
		t.Fatal(err)
	}
}

func TestEnumValueSwitchNarrowing(t *testing.T) {
	const exhaustive = `package p
enum Result { Ok { value int }; Err { err error }; None }
func inspect(result Result) int {
	switch result {
	case Ok: return result.value
	case Err: return len(result.err.Error())
	case None: return 0
	case nil: return -1
	}
	return -2
}
`
	if _, err := checkEnumPackage(t, exhaustive); err != nil {
		t.Fatal(err)
	}

	const duplicate = `package p
enum Result { Ok; Err }
func inspect(result Result) { switch result { case Ok:; case Ok:; case Err:; case nil: } }
`
	_, err := checkEnumPackage(t, duplicate)
	if err == nil || !strings.Contains(err.Error(), "duplicate case Ok in enum switch") {
		t.Fatalf("duplicate enum case error = %v", err)
	}
}

func TestEnumRejectsPointerVariant(t *testing.T) {
	const src = `package p
enum Result { Ok }
var _ = Result(&Ok{})
`
	_, err := checkEnumPackage(t, src)
	if err == nil || !strings.Contains(err.Error(), "pointer to enum variant is not an enum value") {
		t.Fatalf("pointer variant error = %v", err)
	}
}

func TestGenericEnumTypes(t *testing.T) {
	const src = `package p

enum Option[T any] {
	Some { value T }
	None
}

var _ Option[int] = Some[int]{value: 1}
`
	pkg, err := checkEnumPackage(t, src)
	if err != nil {
		t.Fatal(err)
	}
	option := pkg.Scope().Lookup("Option").Type().(*types.Named)
	some := pkg.Scope().Lookup("Some").Type().(*types.Named)
	none := pkg.Scope().Lookup("None").Type().(*types.Named)
	optionParam := option.TypeParams().At(0)
	someParam := some.TypeParams().At(0)
	noneParam := none.TypeParams().At(0)
	if optionParam == someParam || optionParam == noneParam || someParam == noneParam {
		t.Fatal("generic enum and variants share type parameter objects")
	}
	if fieldType := some.Underlying().(*types.Struct).Field(0).Type(); fieldType != someParam {
		t.Fatalf("Some.value type = %v, want Some type parameter %v", fieldType, someParam)
	}
	marker := some.Method(0).Type().(*types.Signature)
	recv := marker.Recv().Type().(*types.Named)
	if recv.TypeArgs().Len() != 1 || marker.RecvTypeParams().Len() != 1 {
		t.Fatalf("marker receiver = %v (type args %d, receiver type params %d)", recv, recv.TypeArgs().Len(), marker.RecvTypeParams().Len())
	}
}

func TestDuplicateEnumVariant(t *testing.T) {
	const src = `package p
enum E { A; A }
`
	_, err := checkEnumPackage(t, src)
	if err == nil || !strings.Contains(err.Error(), "A redeclared") {
		t.Fatalf("duplicate variant error = %v, want redeclaration error", err)
	}
}
