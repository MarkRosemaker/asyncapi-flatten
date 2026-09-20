// Package fsutil provides utilities for working with afero.Fs (and real OS filesystem via the os subpackage).
package fsutil

import (
	"io"
	"path/filepath"

	"github.com/spf13/afero"
)

// Copy copies a single file within the same filesystem.
func Copy(fs afero.Fs, src, dst string) error {
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
