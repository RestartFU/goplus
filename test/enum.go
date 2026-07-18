// run

// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

enum Result {
	Ok { value int }
	Err { err string }
	None
}

enum Option[T any] {
	Some { value T }
	Nothing
}

func (o Ok) Value() int { return o.value }

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

func unwrap[T any](option Option[T], zero T) T {
	switch option {
	case Some[T]:
		return option.value
	case Nothing[T], nil:
		return zero
	}
	panic("unreachable")
}

func main() {
	var result Result = Ok{value: 42}
	if inspect(result) != 42 || result.(Ok).Value() != 42 {
		panic("non-generic enum")
	}

	var option Option[string] = Some[string]{value: "ok"}
	if unwrap(option, "bad") != "ok" {
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
