package uxwing

import (
	"os"
	"path/filepath"
	"testing"
)

var robotSVG = Asset{Format: FormatSVG, Filename: "robot-icon.svg"}

func TestSaveWritesTheFileIntoTheDirectory(t *testing.T) {
	dir := t.TempDir()

	path, err := NewStore().Save(dir, robotSVG, []byte("<svg/>"))
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	if want := filepath.Join(dir, "robot-icon.svg"); path != want {
		t.Errorf("Save wrote %q, want %q", path, want)
	}
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved file: %v", err)
	}
	if string(written) != "<svg/>" {
		t.Errorf("saved file holds %q, want the downloaded bytes", written)
	}
}

func TestSaveCreatesMissingDirectories(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "assets", "icons")

	if _, err := NewStore().Save(dir, robotSVG, []byte("<svg/>")); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "robot-icon.svg")); err != nil {
		t.Errorf("Save did not create the target directory: %v", err)
	}
}

func TestSaveRefusesFilenamesThatEscapeTheDirectory(t *testing.T) {
	dir := t.TempDir()
	escaping := Asset{Format: FormatSVG, Filename: "../escape.svg"}

	if _, err := NewStore().Save(dir, escaping, []byte("<svg/>")); err == nil {
		t.Fatal("Save accepted a filename containing a path separator")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(dir), "escape.svg")); err == nil {
		t.Error("Save wrote a file outside the target directory")
	}
}
