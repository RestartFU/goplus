package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestReleaseArchiveURL(t *testing.T) {
	tests := []struct {
		name    string
		tag     string
		want    string
		wantErr bool
	}{
		{name: "release", tag: "v0.1.0-alpha.1", want: "https://github.com/RestartFU/goplus/releases/download/v0.1.0-alpha.1/goplus-windows-amd64.zip"},
		{name: "development build", tag: "dev", wantErr: true},
		{name: "empty tag", wantErr: true},
		{name: "path separator", tag: "v0.1/escape", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := releaseArchiveURL(test.tag)
			if (err != nil) != test.wantErr {
				t.Errorf("releaseArchiveURL(%q) error = %v, wantErr %v", test.tag, err, test.wantErr)
			}
			if got != test.want {
				t.Errorf("releaseArchiveURL(%q) = %q, want %q", test.tag, got, test.want)
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
			archive := zipArchive(t, map[string]string{test.entry: "installer"})
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

func TestInstallArchive(t *testing.T) {
	tests := []struct {
		name      string
		prepare   func(*testing.T, string)
		wantErr   bool
		wantValue string
	}{
		{name: "fresh install", wantValue: "new"},
		{
			name: "managed upgrade",
			prepare: func(t *testing.T, prefix string) {
				if err := os.MkdirAll(prefix, 0o755); err != nil {
					t.Errorf("create managed prefix: %v", err)
					return
				}
				if err := os.WriteFile(filepath.Join(prefix, managedMarker), []byte("managed"), 0o644); err != nil {
					t.Errorf("write managed marker: %v", err)
				}
				if err := os.WriteFile(filepath.Join(prefix, "old"), []byte("old"), 0o644); err != nil {
					t.Errorf("write old file: %v", err)
				}
			},
			wantValue: "new",
		},
		{
			name: "unmanaged prefix",
			prepare: func(t *testing.T, prefix string) {
				if err := os.MkdirAll(prefix, 0o755); err != nil {
					t.Errorf("create unmanaged prefix: %v", err)
					return
				}
				if err := os.WriteFile(filepath.Join(prefix, "user-file"), []byte("keep"), 0o644); err != nil {
					t.Errorf("write unmanaged file: %v", err)
				}
			},
			wantErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			prefix := filepath.Join(root, "GoPlus")
			if test.prepare != nil {
				test.prepare(t, prefix)
			}
			archive := zipArchive(t, map[string]string{
				managedMarker: "Go+ managed installation",
				"bin/go+.exe": "new",
			})
			archivePath := filepath.Join(root, "goplus.zip")
			if err := os.WriteFile(archivePath, archive, 0o644); err != nil {
				t.Errorf("write archive: %v", err)
				return
			}

			err := installArchive(archivePath, prefix)
			if (err != nil) != test.wantErr {
				t.Errorf("installArchive() error = %v, wantErr %v", err, test.wantErr)
			}
			if test.wantErr {
				return
			}
			contents, err := os.ReadFile(filepath.Join(prefix, "bin", "go+.exe"))
			if err != nil {
				t.Errorf("read installed executable: %v", err)
				return
			}
			if string(contents) != test.wantValue {
				t.Errorf("installed executable = %q, want %q", contents, test.wantValue)
			}
			if _, err := os.Stat(filepath.Join(prefix, "old")); !os.IsNotExist(err) {
				t.Errorf("old installation file still exists")
			}
		})
	}
}

func zipArchive(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for name, contents := range entries {
		file, err := writer.Create(name)
		if err != nil {
			t.Errorf("create zip entry: %v", err)
			return nil
		}
		if _, err := file.Write([]byte(contents)); err != nil {
			t.Errorf("write zip entry: %v", err)
			return nil
		}
	}
	if err := writer.Close(); err != nil {
		t.Errorf("close zip: %v", err)
		return nil
	}
	return archive.Bytes()
}
