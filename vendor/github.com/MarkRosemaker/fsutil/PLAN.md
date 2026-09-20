### Repository Structure

```bash
fsutil/  # Main package: everything that takes afero.Fs
├── fsutil.go
├── cross.go
├── copydir.go
├── atomic.go
├── mkdir.go
└── doc.go
├── osutil/              # Convenience functions for real OS filesystem
│   └── os.go
├── fstest/          # Testing helpers
│   └── fs.go
├── go.mod
├── README.md
└── fsutil_test.go   # Example tests
```

---

### 2. `cross.go`

```go
package fsutil

import "github.com/spf13/afero"

// CopyBetween copies a file from one filesystem to another (supports different FS types).
func CopyBetween(srcFS, dstFS afero.Fs, src, dst string) error {
	return copyFile(srcFS, dstFS, src, dst)
}

// CopyToOs copies a file from any FS into the real operating system.
func CopyToOs(srcFS afero.Fs, src, dst string) error {
	return CopyBetween(srcFS, afero.NewOsFs(), src, dst)
}

// CopyFromOs copies a file from the real OS into any FS.
func CopyFromOs(dstFS afero.Fs, src, dst string) error {
	return CopyBetween(afero.NewOsFs(), dstFS, src, dst)
}
```

---

### 3. `copydir.go`

```go
package fsutil

import (
	"github.com/spf13/afero"
	"os"
	"path/filepath"
)

// CopyDir recursively copies a directory (and all its contents) from srcFS to dstFS.
func CopyDir(srcFS, dstFS afero.Fs, src, dst string) error {
	return afero.Walk(srcFS, src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := afero.Rel(srcFS, src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return MkdirAll(dstFS, target, info.Mode().Perm())
		}
		return copyFile(srcFS, dstFS, path, target)
	})
}
```

---

### 5. `atomic.go` (bonus)

```go
package fsutil

import (
	"github.com/spf13/afero"
	"os"
)

// WriteFileAtomic writes data to a file using a temporary file + rename (atomic on same FS).
func WriteFileAtomic(fs afero.Fs, name string, data []byte, perm os.FileMode) error {
	f, err := afero.TempFile(fs, filepath.Dir(name), ".tmp-"+filepath.Base(name))
	if err != nil {
		return err
	}
	tmpName := f.Name()
	defer fs.Remove(tmpName) // cleanup on failure

	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}

	if err := fs.Chmod(tmpName, perm); err != nil {
		return err
	}

	return fs.Rename(tmpName, name)
}
```

---

### 6. `fstest/fs.go`

```go
// Package fstest provides testing helpers for filesystem assertions.
package fstest

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/afero"
)

var errMismatch = os.ErrInvalid // used only internally

// Equal asserts that two filesystems have identical structure and content.
func Equal(t *testing.T, fs1, fs2 afero.Fs, checkPermissions bool, msgAndArgs ...interface{}) {
	t.Helper()

	equal, err := equalRecursive(fs1, fs2, ".", checkPermissions)
	if err != nil {
		t.Fatalf("failed to compare filesystems: %v", err)
	}
	if !equal {
		t.Error("filesystems are not equal")
		if len(msgAndArgs) > 0 {
			t.Errorf(msgAndArgs[0].(string), msgAndArgs[1:]...)
		}
	}
}

func equalRecursive(fs1, fs2 afero.Fs, root string, checkPerms bool) (bool, error) {
	err := afero.Walk(fs1, root, func(path string, info1 os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		info2, err := fs2.Stat(path)
		if err != nil {
			return errMismatch // missing file
		}

		if info1.IsDir() != info2.IsDir() {
			return errMismatch
		}

		if info1.IsDir() {
			return nil
		}

		// Compare content
		b1, err := afero.ReadFile(fs1, path)
		if err != nil {
			return err
		}
		b2, err := afero.ReadFile(fs2, path)
		if err != nil {
			return err
		}
		if !bytes.Equal(b1, b2) {
			return errMismatch
		}

		if checkPerms && info1.Mode().Perm() != info2.Mode().Perm() {
			return errMismatch
		}

		return nil
	})

	if err == errMismatch {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	// TODO: Optional second walk to ensure fs2 has no extra files
	return true, nil
}
```

---

### 7. `osutil/os.go`

```go
// Package osutil provides convenient functions that operate directly on the real OS filesystem.
package osutil

import (
	"github.com/spf13/afero"
	"github.com/MarkRosemaker/fsutil"
)


func CopyDir(src, dst string) error {
	return fsutil.CopyDir(afero.NewOsFs(), afero.NewOsFs(), src, dst)
}

func CopyToOs(srcFS afero.Fs, src, dst string) error {
	return fsutil.CopyToOs(srcFS, src, dst)
}

func CopyFromOs(dstFS afero.Fs, src, dst string) error {
	return fsutil.CopyFromOs(dstFS, src, dst)
}

// Add more helpers as needed (MkdirAll, WriteFileAtomic, etc.)
```
