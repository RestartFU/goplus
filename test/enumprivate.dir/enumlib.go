// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enumlib

enum Option[T any] {
	Some { Value T }
	None
	hidden
}
