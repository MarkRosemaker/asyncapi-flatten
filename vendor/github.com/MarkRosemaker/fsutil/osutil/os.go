// Package osutil provides convenient functions that operate directly on the real OS filesystem.
package osutil

import (
	"github.com/MarkRosemaker/fsutil"
	"github.com/spf13/afero"
)

// Copy copies a single file in the OS filesystem.
func Copy(src, dst string) error {
	return fsutil.Copy(afero.NewOsFs(), src, dst)
}
