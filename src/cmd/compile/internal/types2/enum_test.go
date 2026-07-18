// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types2_test

import (
	. "cmd/compile/internal/types2"
	"strings"
	"testing"
)

func TestEnumTypes(t *testing.T) {
	const src = `package p

enum Result {
	Ok { value int }
	Err { err error }
	None
}

func (r Result) Value() int {
	switch r {
	case Ok: return r.value
	case Err: return -1
	case None, nil: return 0
	}
	return 0
}
func makeResult() Result { return Ok{value: 42} }
`
	pkg, err := typecheck(src, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	result := pkg.Scope().Lookup("Result").Type().(*Named)
	ok := pkg.Scope().Lookup("Ok").Type().(*Named)
	errVariant := pkg.Scope().Lookup("Err").Type().(*Named)
	none := pkg.Scope().Lookup("None").Type().(*Named)

	if _, ok := result.Underlying().(*Interface); !ok {
		t.Fatalf("Result underlying type is %T, want *Interface", result.Underlying())
	}
	for _, variant := range []*Named{ok, errVariant, none} {
		if _, ok := variant.Underlying().(*Struct); !ok {
			t.Errorf("%s underlying type is %T, want *Struct", variant.Obj().Name(), variant.Underlying())
		}
		if !AssignableTo(variant, result) {
			t.Errorf("%s is not assignable to Result", variant.Obj().Name())
		}
	}
	if fields := ok.Underlying().(*Struct); fields.NumFields() != 1 || fields.Field(0).Name() != "value" {
		t.Fatalf("Ok fields = %s, want value int", fields)
	}
	if method, _, _ := LookupFieldOrMethod(result, true, pkg, "Value"); method == nil {
		t.Error("method declared on enum Result was not collected")
	}
	if method, _, _ := LookupFieldOrMethod(ok, true, pkg, "Value"); method != nil {
		t.Error("enum method Value unexpectedly belongs to variant Ok")
	}
	variants := result.EnumVariants()
	if len(variants) != 3 || variants[0].Obj().Name() != "Ok" || variants[1].Obj().Name() != "Err" || variants[2].Obj().Name() != "None" {
		t.Fatalf("Result variants = %v, want [Ok Err None]", variants)
	}
	if ok.EnumType() != result {
		t.Fatalf("Ok enum type = %v, want Result", ok.EnumType())
	}
	if AssignableTo(NewPointer(ok), result) {
		t.Error("*Ok is assignable to Result")
	}
}

func TestEnumVariantReceiverRejected(t *testing.T) {
	const src = `package p
enum Result { Ok { value int }; Err }
func (o Ok) Value() int { return o.value }
`
	_, err := typecheck(src, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "cannot define method on enum variant Ok; use enum type Result as receiver") {
		t.Fatalf("variant receiver error = %v", err)
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
	if _, err := typecheck(exhaustive, nil, nil); err != nil {
		t.Fatal(err)
	}

	const missing = `package p
enum Result { Ok; Err; None }
func inspect(r Result) { switch r.(type) { case Ok: } }
`
	_, err := typecheck(missing, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "non-exhaustive enum switch on Result; missing Err, None, nil") {
		t.Fatalf("non-exhaustive switch error = %v", err)
	}

	const withDefault = `package p
enum Result { Ok; Err }
func inspect(r Result) { switch r.(type) { case Ok:; default: } }
`
	if _, err := typecheck(withDefault, nil, nil); err != nil {
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
	if _, err := typecheck(exhaustive, nil, nil); err != nil {
		t.Fatal(err)
	}

	const duplicate = `package p
enum Result { Ok; Err }
func inspect(result Result) { switch result { case Ok:; case Ok:; case Err:; case nil: } }
`
	_, err := typecheck(duplicate, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "duplicate case Ok in enum switch") {
		t.Fatalf("duplicate enum case error = %v", err)
	}
}

func TestEnumRejectsPointerVariant(t *testing.T) {
	const src = `package p
enum Result { Ok }
var _ = Result(&Ok{})
`
	_, err := typecheck(src, nil, nil)
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

func (o Option[T]) Or(zero T) T {
	switch o {
	case Some[T]: return o.value
	case None[T], nil: return zero
	}
	return zero
}

var _ Option[int] = Some[int]{value: 1}
`
	pkg, err := typecheck(src, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	option := pkg.Scope().Lookup("Option").Type().(*Named)
	some := pkg.Scope().Lookup("Some").Type().(*Named)
	none := pkg.Scope().Lookup("None").Type().(*Named)
	optionParam := option.TypeParams().At(0)
	someParam := some.TypeParams().At(0)
	noneParam := none.TypeParams().At(0)
	if optionParam == someParam || optionParam == noneParam || someParam == noneParam {
		t.Fatal("generic enum and variants share type parameter objects")
	}
	if fieldType := some.Underlying().(*Struct).Field(0).Type(); fieldType != someParam {
		t.Fatalf("Some.value type = %v, want Some type parameter %v", fieldType, someParam)
	}
	if method, _, _ := LookupFieldOrMethod(option, true, pkg, "Or"); method == nil {
		t.Fatal("generic enum method Or was not collected")
	}
	marker := some.Method(0).Type().(*Signature)
	recv := marker.Recv().Type().(*Named)
	if recv.TypeArgs().Len() != 1 || marker.RecvTypeParams().Len() != 1 {
		t.Fatalf("marker receiver = %v (type args %d, receiver type params %d)", recv, recv.TypeArgs().Len(), marker.RecvTypeParams().Len())
	}
}

func TestDuplicateEnumVariant(t *testing.T) {
	const src = `package p
enum E { A; A }
`
	_, err := typecheck(src, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "A redeclared") {
		t.Fatalf("duplicate variant error = %v, want redeclaration error", err)
	}
}
