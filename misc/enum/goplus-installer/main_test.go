package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestSourceArchiveURL(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		want    string
		wantErr bool
	}{
		{name: "release", tag: "v0.1.0-alpha.1", want: "https://github.com/RestartFU/goplus/archive/refs/tags/v0.1.0-alpha.1.zip"},
		{name: "development build", tag: "dev", wantErr: true},
		{name: "empty tag", wantErr: true},
		{name: "path separator", tag: "v0.1/escape", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := sourceArchiveURL(test.tag)
			if (err != nil) != test.wantErr {
				t.Errorf("sourceArchiveURL(%q) error = %v, wantErr %v", test.tag, err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("sourceArchiveURL(%q) = %q, want %q", test.tag, got, test.want)
			}
		})
	}
}

func TestExtractArchive(t *testing.T) {
	tests := []struct {
		name    string
		entry   string
		wantErr bool
	}{
		{name: "source file", entry: "goplus-tag/install-goplus.ps1"},
		{name: "parent traversal", entry: "../escape", wantErr: true},
		{name: "nested traversal", entry: "goplus-tag/../../escape", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archive := zipArchive(t, test.entry, "installer")
			destination := t.TempDir()
			err := extractArchive(bytes.NewReader(archive), int64(len(archive)), destination)
			if (err != nil) != test.wantErr {
				t.Errorf("extractArchive() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}
			contents, err := os.ReadFile(filepath.Join(destination, filepath.FromSlash(test.entry)))
			if err != nil {
				t.Errorf("read extracted file: %v", err)
				return
			}
			if string(contents) != "installer" {
				t.Errorf("extracted contents = %q, want installer", contents)
			}
		})
	}
}

func zipArchive(t *testing.T, name, contents string) []byte {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	file, err := writer.Create(name)
	if err != nil {
		t.Errorf("create zip entry: %v", err)
		return nil
	}
	if _, err := file.Write([]byte(contents)); err != nil {
		t.Errorf("write zip entry: %v", err)
		return nil
	}
	if err := writer.Close(); err != nil {
		t.Errorf("close zip: %v", err)
		return nil
	}
	return archive.Bytes()
}
