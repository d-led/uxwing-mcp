package uxwing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store writes downloaded files to disk. It keeps no state: saving the same
// icon twice ends with the same files.
type Store struct{}

func NewStore() *Store { return &Store{} }

// Save writes data into dir under the asset's own file name and returns the
// path it wrote.
func (*Store) Save(dir string, asset Asset, data []byte) (string, error) {
	if !safeFilename(asset.Filename) {
		return "", fmt.Errorf("refusing to save icon as %q", asset.Filename)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create icon directory %q: %w", dir, err)
	}

	path := filepath.Join(dir, asset.Filename)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("write %s: %w", path, err)
	}
	return path, nil
}

// safeFilename accepts a plain file name and nothing that could leave dir.
func safeFilename(name string) bool {
	return name != "" && name != "." && name != ".." &&
		!strings.ContainsAny(name, `/\`) && filepath.Base(name) == name
}
