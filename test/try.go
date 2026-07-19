// run

// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "errors"

var errLoad = errors.New("load failed")

type user struct {
	name string
}

func load(ok bool) (user, error) {
	if !ok {
		return user{}, errLoad
	}
	return user{name: "gopher"}, nil
}

func loadPair(ok bool) (user, int, error) {
	if !ok {
		return user{}, 0, errLoad
	}
	return user{name: "gopher"}, 42, nil
}

func loadValue[T any](value T, ok bool) (T, error) {
	if !ok {
		var zero T
		return zero, errLoad
	}
	return value, nil
}

func getUser(ok bool) (user, error) {
	try loaded := load(ok)
	return loaded, nil
}

func getPair(ok bool) (user, int, error) {
	try loaded, count := loadPair(ok)
	return loaded, count, nil
}

func getValue[T any](value T, ok bool) (T, error) {
	try loaded := loadValue(value, ok)
	return loaded, nil
}

func main() {
	try := 1
	try++
	if try != 2 {
		panic("try is not contextual")
	}

	loaded, err := getUser(true)
	if err != nil || loaded.name != "gopher" {
		panic("single success value")
	}

	loaded, count, err := getPair(true)
	if err != nil || loaded.name != "gopher" || count != 42 {
		panic("multiple success values")
	}

	loaded, count, err = getPair(false)
	if !errors.Is(err, errLoad) || loaded != (user{}) || count != 0 {
		panic("error propagation")
	}

	value, err := getValue("generic", true)
	if err != nil || value != "generic" {
		panic("generic propagation")
	}
}
