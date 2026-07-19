// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build compiler_bootstrap

package main

// walkExtensions is empty while cmd/cgo is built against bootstrap go/ast,
// which predates Go+ AST nodes. Bootstrap inputs cannot contain Go+ syntax.
func (*File) walkExtensions(any, func(*File, any, astContext)) bool { return false }
