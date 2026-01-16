package assets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteToDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ai-shell-assets-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	if err := WriteToDir(tmpDir, "base"); err != nil {
		t.Fatalf("WriteToDir failed: %v", err)
	}

	expectedFiles := []string{
		"Containerfile",
		"configure.sh",
		"config.default.yaml",
	}

	for _, file := range expectedFiles {
		path := filepath.Join(tmpDir, file)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("File %s not found: %v", file, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("File %s is empty", file)
		}

		if file == "configure.sh" {
			// Check executable permission (0755)
			// Note: on some filesystems/OS this check might be tricky, but basic check:
			if info.Mode()&0111 == 0 {
				t.Errorf("configure.sh should be executable")
			}
		}
	}
}
