// goplus-installer installs a precompiled Go+ release on Windows.
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

const (
	repository     = "RestartFU/goplus"
	releaseArchive = "goplus-windows-amd64.zip"
	managedMarker  = ".goplus-managed"
)

var releaseTag = "dev"

func main() {
	log.SetFlags(0)
	prefix := flag.String("prefix", "", "installation directory (defaults to %LOCALAPPDATA%\\GoPlus)")
	archive := flag.String("archive", "", "install a local precompiled archive instead of downloading one")
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
	if err := install(*prefix, *archive, *noPathUpdate); err != nil {
		log.Fatalf("install Go+ %s: %v", releaseTag, err)
	}
}

func install(prefix, localArchive string, noPathUpdate bool) error {
	var err error
	prefix, err = installPrefix(prefix)
	if err != nil {
		return err
	}

	archivePath := localArchive
	if archivePath == "" {
		archiveURL, err := releaseArchiveURL(releaseTag)
		if err != nil {
			return err
		}
		work, err := os.MkdirTemp("", "goplus-download-")
		if err != nil {
			return fmt.Errorf("create download directory: %w", err)
		}
		defer os.RemoveAll(work)
		archivePath = filepath.Join(work, releaseArchive)
		fmt.Printf("Downloading Go+ %s...\n", releaseTag)
		if err := download(archiveURL, archivePath); err != nil {
			return err
		}
	}

	fmt.Printf("Installing Go+ %s...\n", releaseTag)
	if err := installArchive(archivePath, prefix); err != nil {
		return err
	}
	if !noPathUpdate {
		if err := updateUserPath(filepath.Join(prefix, "bin")); err != nil {
			return err
		}
	}
	command := exec.Command(filepath.Join(prefix, "bin", "go+.exe"), "version")
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("verify installed go+: %w", err)
	}
	fmt.Printf("Go+ %s installed successfully. Open a new terminal before using it.\n", releaseTag)
	return nil
}

func installPrefix(prefix string) (string, error) {
	if prefix == "" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return "", errors.New("LOCALAPPDATA is not set")
		}
		prefix = filepath.Join(localAppData, "GoPlus")
	}
	absolute, err := filepath.Abs(prefix)
	if err != nil {
		return "", fmt.Errorf("resolve install prefix: %w", err)
	}
	absolute = filepath.Clean(absolute)
	root := filepath.VolumeName(absolute) + string(filepath.Separator)
	home, _ := os.UserHomeDir()
	if strings.EqualFold(absolute, root) || home != "" && strings.EqualFold(absolute, filepath.Clean(home)) {
		return "", fmt.Errorf("refusing unsafe install prefix %q", absolute)
	}
	return absolute, nil
}

func releaseArchiveURL(tag string) (string, error) {
	if tag == "" || tag == "dev" || url.PathEscape(tag) != tag {
		return "", fmt.Errorf("invalid release tag %q", tag)
	}
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repository, tag, releaseArchive), nil
}

func download(sourceURL, destination string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	response, err := client.Get(sourceURL)
	if err != nil {
		return fmt.Errorf("download release archive: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("download release archive: %s", response.Status)
	}

	file, err := os.Create(destination)
	if err != nil {
		return fmt.Errorf("create release archive: %w", err)
	}
	_, copyErr := io.Copy(file, response.Body)
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("save release archive: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close release archive: %w", closeErr)
	}
	return nil
}

func installArchive(archivePath, prefix string) error {
	if err := validateExistingPrefix(prefix); err != nil {
		return err
	}
	parent := filepath.Dir(prefix)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return fmt.Errorf("create install parent: %w", err)
	}
	stage, err := os.MkdirTemp(parent, ".goplus-install-")
	if err != nil {
		return fmt.Errorf("create install staging directory: %w", err)
	}
	defer os.RemoveAll(stage)

	archive, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open release archive: %w", err)
	}
	info, err := archive.Stat()
	if err != nil {
		archive.Close()
		return fmt.Errorf("inspect release archive: %w", err)
	}
	if err := extractArchive(archive, info.Size(), stage); err != nil {
		archive.Close()
		return err
	}
	if err := archive.Close(); err != nil {
		return fmt.Errorf("close release archive: %w", err)
	}
	for _, required := range []string{managedMarker, filepath.Join("bin", "go+.exe")} {
		if _, err := os.Stat(filepath.Join(stage, required)); err != nil {
			return fmt.Errorf("release archive missing %s: %w", required, err)
		}
	}

	backup := fmt.Sprintf("%s.backup.%d", prefix, os.Getpid())
	if _, err := os.Lstat(backup); err == nil {
		return fmt.Errorf("backup path already exists: %s", backup)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect backup path: %w", err)
	}
	hadExisting := false
	if _, err := os.Lstat(prefix); err == nil {
		if err := os.Rename(prefix, backup); err != nil {
			return fmt.Errorf("back up existing installation: %w", err)
		}
		hadExisting = true
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect existing installation: %w", err)
	}
	if err := os.Rename(stage, prefix); err != nil {
		if hadExisting {
			_ = os.Rename(backup, prefix)
		}
		return fmt.Errorf("activate installation: %w", err)
	}
	if hadExisting {
		if err := os.RemoveAll(backup); err != nil {
			return fmt.Errorf("remove previous installation: %w", err)
		}
	}
	return nil
}

func validateExistingPrefix(prefix string) error {
	info, err := os.Lstat(prefix)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect install prefix: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("install prefix is not a regular directory: %s", prefix)
	}
	entries, err := os.ReadDir(prefix)
	if err != nil {
		return fmt.Errorf("read install prefix: %w", err)
	}
	if len(entries) == 0 {
		return nil
	}
	if _, err := os.Stat(filepath.Join(prefix, managedMarker)); err != nil {
		return fmt.Errorf("refusing to replace unmanaged non-empty directory %s", prefix)
	}
	return nil
}

func extractArchive(source io.ReaderAt, size int64, destination string) error {
	archive, err := zip.NewReader(source, size)
	if err != nil {
		return fmt.Errorf("open release archive: %w", err)
	}
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return fmt.Errorf("create staging directory: %w", err)
	}
	for _, entry := range archive.File {
		cleanName := filepath.Clean(filepath.FromSlash(entry.Name))
		if cleanName == "." && entry.FileInfo().IsDir() {
			continue
		}
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

func updateUserPath(binDirectory string) error {
	powerShell, err := findPowerShell()
	if err != nil {
		return err
	}
	const script = `$entries = @([Environment]::GetEnvironmentVariable('Path', 'User') -split ';' | Where-Object { $_ }); if (-not ($entries | Where-Object { $_.TrimEnd('\') -ieq $env:GOPLUS_BIN_DIR.TrimEnd('\') })) { [Environment]::SetEnvironmentVariable('Path', (($env:GOPLUS_BIN_DIR + ';' + ($entries -join ';')).TrimEnd(';')), 'User') }`
	command := exec.Command(powerShell, "-NoLogo", "-NoProfile", "-Command", script)
	command.Env = append(os.Environ(), "GOPLUS_BIN_DIR="+binDirectory)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("update user PATH: %w", err)
	}
	return nil
}

func findPowerShell() (string, error) {
	for _, name := range []string{"pwsh.exe", "powershell.exe"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}
	return "", errors.New("PowerShell is required to update the user PATH")
}
