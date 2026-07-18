// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestTargetFor(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{"go+", []string{"go", "bin", "go.exe"}},
		{"gofmt+", []string{"go", "bin", "gofmt.exe"}},
		{"gopls+", []string{"libexec", "gopls.exe"}},
		{"goimports+", []string{"libexec", "goimports.exe"}},
	}
	for _, test := range tests {
		got, ok := targetFor(test.name, "windows")
		if !ok || !slices.Equal(got, test.want) {
			t.Errorf("targetFor(%q) = %v, %v; want %v, true", test.name, got, ok, test.want)
		}
	}
	if _, ok := targetFor("unknown", "windows"); ok {
		t.Error("targetFor(unknown) succeeded")
	}
}

func TestToolEnv(t *testing.T) {
	t.Setenv("PATH", "old-path")
	got := toolEnv([]string{"GOROOT=old", "Path=old-path", "GOTOOLCHAIN=auto", "KEEP=yes"}, "new-root")
	wantPath := "PATH=" + filepath.Join("new-root", "bin") + string(os.PathListSeparator) + "old-path"
	for _, want := range []string{"KEEP=yes", "GOROOT=new-root", wantPath, "GOTOOLCHAIN=local"} {
		if !slices.Contains(got, want) {
			t.Errorf("toolEnv() = %v, missing %q", got, want)
		}
	}
}
