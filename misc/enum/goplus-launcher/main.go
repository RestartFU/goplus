// Copyright 2026 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command goplus-launcher runs a Go+ tool with the private Go+ GOROOT.
// The installer copies this binary to go+, gofmt+, gopls+, and goimports+.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func main() {
	executable, err := os.Executable()
	if err != nil {
		fatal(err)
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		fatal(err)
	}
	name := strings.TrimSuffix(filepath.Base(executable), filepath.Ext(executable))
	rel, ok := targetFor(name, runtime.GOOS)
	if !ok {
		fatal(fmt.Errorf("unknown Go+ launcher name %q", name))
	}

	root := filepath.Dir(filepath.Dir(executable))
	goroot := filepath.Join(root, "go")
	target := filepath.Join(append([]string{root}, rel...)...)
	cmd := exec.Command(target, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = toolEnv(os.Environ(), goroot)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fatal(err)
	}
}

func targetFor(name, goos string) ([]string, bool) {
	suffix := ""
	if goos == "windows" {
		suffix = ".exe"
	}
	switch name {
	case "go+":
		return []string{"go", "bin", "go" + suffix}, true
	case "gofmt+":
		return []string{"go", "bin", "gofmt" + suffix}, true
	case "gopls+":
		return []string{"libexec", "gopls" + suffix}, true
	case "goimports+":
		return []string{"libexec", "goimports" + suffix}, true
	default:
		return nil, false
	}
}

func toolEnv(environ []string, goroot string) []string {
	pathValue := filepath.Join(goroot, "bin")
	if old := os.Getenv("PATH"); old != "" {
		pathValue += string(os.PathListSeparator) + old
	}
	result := make([]string, 0, len(environ)+3)
	for _, entry := range environ {
		name, _, _ := strings.Cut(entry, "=")
		if strings.EqualFold(name, "GOROOT") || strings.EqualFold(name, "PATH") || strings.EqualFold(name, "GOTOOLCHAIN") {
			continue
		}
		result = append(result, entry)
	}
	return append(result, "GOROOT="+goroot, "PATH="+pathValue, "GOTOOLCHAIN=local")
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "goplus:", err)
	os.Exit(1)
}
