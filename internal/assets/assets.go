package assets

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed all:files
var files embed.FS

// WriteToDir writes the embedded build assets (Containerfile, configure.sh, config.default.yaml)
// to the specified directory.
func WriteToDir(dir string) error {
	entries, err := files.ReadDir("files")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		data, err := files.ReadFile(filepath.Join("files", entry.Name()))
		if err != nil {
			return err
		}

		path := filepath.Join(dir, entry.Name())
		// config.default.yaml -> config.yaml for the build context if needed,
		// but Containerfile expects config.default.yaml, so keep names.
		// Wait, Containerfile COPYs config.default.yaml to /etc/ai-shell/config.yaml.
		// So we just write exactly what we have.

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
