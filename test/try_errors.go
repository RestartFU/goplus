// errorcheck

// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p

func load() (int, error) { return 0, nil }

func loadString() (int, string) { return 0, "" }

func loadBool() (int, bool) { return 0, false }

func loadValueOnly() int { return 0 }

func wrongTryResult() (int, error) {
	try value := loadString() // ERROR "final try result must have type error"
	return value, nil
}

func wrongErrorOnlyTryResult() error {
	try loadValueOnly() // ERROR "final try result must have type error"
	return nil
}

func wrongFunctionResult() int {
	try value := load() // ERROR "try requires the enclosing function to return error as its final result"
	return value
}

func wrongBoolFunctionResult() (int, error) {
	try value := loadBool() // ERROR "try requires the enclosing function to return bool as its final result"
	return value, nil
}

func noNewVariable() (int, error) {
	value := 0
	try value := load() // ERROR "no new variables on left side of :="
	return value, nil
}

func wrongBindingCount() (int, error) {
	try value, extra := load() // ERROR "assignment mismatch"
	return value + extra, nil
}
