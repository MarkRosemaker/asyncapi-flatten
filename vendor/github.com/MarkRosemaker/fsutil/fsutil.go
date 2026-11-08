// Package fsutil provides utilities for working with afero.Fs (and real OS filesystem via the os subpackage).
package fsutil

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/spf13/afero"
)

// ErrSameFile is returned when the source and the destination of a copy are one file, which copying would empty.
var ErrSameFile = errors.New("source and destination are the same file")

// Copy copies a single file within the same filesystem.
func Copy(fs afero.Fs, src, dst string) error {
	if err := sameFile(fs, src, dst); err != nil {
		return err
	}

	return copyFile(fs, fs, src, dst)
}

// copyFile is the internal shared implementation.
func copyFile(srcFS, dstFS afero.Fs, src, dst string) error {
	if err := MkdirAll(dstFS, filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	srcFile, err := srcFS.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close() //nolint:errcheck

	dstFile, err := dstFS.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close() //nolint:errcheck

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Best-effort permission preservation
	if fi, err := srcFile.Stat(); err == nil {
		_ = dstFS.Chmod(dst, fi.Mode().Perm())
	}

	return dstFile.Sync()
}

// sameFile reports ErrSameFile where creating dst would truncate src before it is read.
func sameFile(fs afero.Fs, src, dst string) error {
	if filepath.Clean(src) == filepath.Clean(dst) {
		return fmt.Errorf("copying %s to %s: %w", src, dst, ErrSameFile)
	}

	// The same file under another name, such as a hard link, on a filesystem that can tell.
	srcInfo, err := fs.Stat(src)
	if err != nil {
		return nil //nolint:nilerr // opening src reports it
	}

	if dstInfo, err := fs.Stat(dst); err == nil && os.SameFile(srcInfo, dstInfo) {
		return fmt.Errorf("copying %s to %s: %w", src, dst, ErrSameFile)
	}

	return nil
}
