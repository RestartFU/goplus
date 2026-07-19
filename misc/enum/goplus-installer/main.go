// goplus-installer installs a tagged Go+ release on Windows.
package main

import (
	"archive/zip"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const repository = "RestartFU/goplus"

var releaseTag = "dev"

func main() {
	log.SetFlags(0)
	prefix := flag.String("prefix", "", "installation directory (defaults to %LOCALAPPDATA%\\GoPlus)")
	noPathUpdate := flag.Bool("no-path-update", false, "do not add the Go+ bin directory to the user PATH")
	showVersion := flag.Bool("version", false, "print the installer version")
	flag.Parse()

	if *showVersion {
		fmt.Println(releaseTag)
		return
	}
	if runtime.GOOS != "windows" {
		log.Fatal("the Go+ Windows installer must run on Windows")
	}
	if err := install(*prefix, *noPathUpdate); err != nil {
		log.Fatalf("install Go+ %s: %v", releaseTag, err)
	}
}

func install(prefix string, noPathUpdate bool) error {
	archiveURL, err := sourceArchiveURL(releaseTag)
	if err != nil {
		return err
	}

	work, err := os.MkdirTemp("", "goplus-installer-")
	if err != nil {
		return fmt.Errorf("create temporary directory: %w", err)
	}
	defer os.RemoveAll(work)

	archivePath := filepath.Join(work, "source.zip")
	fmt.Printf("Downloading Go+ %s...\n", releaseTag)
	if err := download(archiveURL, archivePath); err != nil {
		return err
	}
	archive, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open source archive: %w", err)
	}
	info, err := archive.Stat()
	if err != nil {
		archive.Close()
		return fmt.Errorf("inspect source archive: %w", err)
	}
	sourceDirectory := filepath.Join(work, "source")
	if err := extractArchive(archive, info.Size(), sourceDirectory); err != nil {
		archive.Close()
		return err
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close source archive: %w", err)
	}

	installer, err := findPowerShellInstaller(sourceDirectory)
	if err != nil {
		return err
	}
	powerShell, err := findPowerShell()
	if err != nil {
		return err
	}
	arguments := []string{"-NoLogo", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", installer}
	if prefix != "" {
		arguments = append(arguments, "-Prefix", prefix)
	}
	if noPathUpdate {
		arguments = append(arguments, "-NoPathUpdate")
	}

	fmt.Println("Building and installing Go+...")
	command := exec.Command(powerShell, arguments...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("run Windows installer: %w", err)
	}
	fmt.Printf("Go+ %s installed successfully.\n", releaseTag)
	return nil
}

func sourceArchiveURL(tag string) (string, error) {
	if tag == "" || tag == "dev" || url.PathEscape(tag) != tag {
		return "", fmt.Errorf("invalid release tag %q", tag)
	}
	return fmt.Sprintf("https://github.com/%s/archive/refs/tags/%s.zip", repository, tag), nil
}

func download(sourceURL, destination string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	response, err := client.Get(sourceURL)
	if err != nil {
		return fmt.Errorf("download source archive: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download source archive: %s", response.Status)
	}

	file, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create source archive: %w", err)
	}
	_, copyErr := io.Copy(file, response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("save source archive: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close source archive: %w", closeErr)
	}
	return nil
}

func extractArchive(source io.ReaderAt, size int64, destination string) error {
	archive, err := zip.NewReader(source, size)
	if err != nil {
		return fmt.Errorf("open source archive: %w", err)
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return fmt.Errorf("create source directory: %w", err)
	}
	for _, entry := range archive.File {
		cleanName := filepath.Clean(filepath.FromSlash(entry.Name))
		if cleanName == "." || filepath.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) {
			return fmt.Errorf("unsafe archive path %q", entry.Name)
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("archive contains symbolic link %q", entry.Name)
		}
		target := filepath.Join(destination, cleanName)
		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create archive directory: %w", err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create archive parent: %w", err)
		}
		reader, err := entry.Open()
		if err != nil {
			return fmt.Errorf("open archive entry: %w", err)
		}
		writer, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, entry.Mode().Perm())
		if err != nil {
			reader.Close()
			return fmt.Errorf("create archive entry: %w", err)
		}
		_, copyErr := io.Copy(writer, reader)
		readerCloseErr := reader.Close()
		writerCloseErr := writer.Close()
		if copyErr != nil {
			return fmt.Errorf("extract archive entry: %w", copyErr)
		}
		if readerCloseErr != nil {
			return fmt.Errorf("close archive entry: %w", readerCloseErr)
		}
		if writerCloseErr != nil {
			return fmt.Errorf("close extracted file: %w", writerCloseErr)
		}
	}
	return nil
}

func findPowerShellInstaller(sourceDirectory string) (string, error) {
	matches, err := filepath.Glob(filepath.Join(sourceDirectory, "*", "install-goplus.ps1"))
	if err != nil {
		return "", fmt.Errorf("find Windows installer: %w", err)
	}
	if len(matches) != 1 {
		return "", fmt.Errorf("source archive contains %d Windows installers", len(matches))
	}
	return matches[0], nil
}

func findPowerShell() (string, error) {
	for _, name := range []string{"pwsh.exe", "powershell.exe"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.New("PowerShell is required")
}
