package domain

import (
	"path/filepath"
)

func MakeFileName(dir, name string) string {
	return filepath.Join(dir, name)
}
