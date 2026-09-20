package flatten

import "fmt"

func uniqueName[M ~map[string]V, V any](m M, name string) string {
	idx := 1
	altName := name

	for {
		if _, ok := m[altName]; !ok {
			return altName
		}

		idx++
		altName = fmt.Sprintf("%s%d", name, idx)
	}
}
