package fsutil

import (
	"os"

	"github.com/spf13/afero"
)

// MkdirAll creates a directory and all parent directories.
func MkdirAll(fs afero.Fs, path string, perm os.FileMode) error {
	return fs.MkdirAll(path, perm)
}
