// run

// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "reflect"

var enum = 1

enum Result {
	Ok { value int }
	Err { err string }
	None
}

enum Option[T any] {
	Some { value T }
	Nothing
}

func (o Option[T]) Or(zero T) T {
	switch o {
	case Some:
		return o.value
	case Nothing, nil:
		return zero
	}
	return zero
}

func (r Result) Value() int {
	switch r {
	case Ok:
		return r.value
	case Err:
		return -1
	case None, nil:
		return 0
	}
	return 0
}

func inspect(result Result) int {
	switch result {
	case Ok:
		return result.value
	case Err:
		return len(result.err)
	case None:
		return 0
	case nil:
		return -1
	}
	panic("unreachable")
}

func makeResult() Result { return Ok{value: 42} }

func inspectExpression() int {
	switch makeResult() {
	case Ok:
		return 42
	case Err, None, nil:
		return 0
	}
	return 0
}

func unwrap[T any](option Option[T], zero T) T {
	switch option {
	case Some:
		return option.value
	case Nothing, nil:
		return zero
	}
	panic("unreachable")
}

func main() {
	enum := enum + 1
	enum++
	if enum != 3 {
		panic("contextual enum keyword")
	}

	var pointer any = &Result.Ok{value: 42}
	if _, ok := pointer.(Result); ok {
		panic("pointer to enum variant passed runtime assertion")
	}
	if reflect.TypeFor[*Result.Ok]().Implements(reflect.TypeFor[Result]()) {
		panic("pointer to enum variant implements enum through reflection")
	}

	var result Result = Ok{value: 42}
	if inspect(result) != 42 || inspectExpression() != 42 || result.Value() != 42 {
		panic("non-generic enum")
	}

	var option Option[string] = Some{value: "ok"}
	if unwrap(option, "bad") != "ok" || option.Or("bad") != "ok" {
		panic("generic enum")
	}

	enum Local {
		Here { value int }
		Gone
	}
	var local Local = Here{value: 7}
	switch local {
	case Here:
		if local.value != 7 {
			panic("local enum")
		}
	case Gone, nil:
		panic("wrong local variant")
	}
}
