package assets

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed all:files
var files embed.FS

// WriteToDir writes the embedded build assets from a specific subdirectory
// (e.g., "base", "profiles/default") to the specified directory.
func WriteToDir(targetDir, subDir string) error {
	srcDir := filepath.Join("files", subDir)
	entries, err := files.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Skip subdirectories
		}
		data, err := files.ReadFile(filepath.Join(srcDir, entry.Name()))
		if err != nil {
			return err
		}

		path := filepath.Join(targetDir, entry.Name())

		perm := os.FileMode(0644)
		if entry.Name() == "configure.sh" {
			perm = 0755
		}

		if err := os.WriteFile(path, data, perm); err != nil {
			return fmt.Errorf("failed to write asset %s: %w", entry.Name(), err)
		}
	}
	return nil
}
