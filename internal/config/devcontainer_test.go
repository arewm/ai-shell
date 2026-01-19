package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDevContainer(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "devcontainer-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	jsonContent := `
	{
		"image": "my-image:latest",
		"runArgs": ["--network=host"],
		"containerEnv": {
			"MY_VAR": "value"
		},
		"mounts": [
			"source=${localWorkspaceFolder},target=/workspace,type=bind"
		]
		// Comments are allowed
	}
	`
	path := filepath.Join(tmpDir, "devcontainer.json")
	if err = os.WriteFile(path, []byte(jsonContent), 0600); err != nil {
		t.Fatal(err)
	}

	dc, err := ParseDevContainer(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	cfg := dc.ToConfig()

	if len(cfg.PodmanArgs) != 1 || cfg.PodmanArgs[0] != "--network=host" {
		t.Errorf("RunArgs mismatch: %v", cfg.PodmanArgs)
	}

	if len(cfg.EnvVars) != 1 || cfg.EnvVars[0] != "MY_VAR" {
		t.Errorf("EnvVars mismatch: %v", cfg.EnvVars)
	}

	if len(cfg.Mounts) != 1 {
		t.Errorf("Mounts mismatch: %v", cfg.Mounts)
	} else {
		m := cfg.Mounts[0]
		if m.Source != "${localWorkspaceFolder}" || m.Target != "/workspace" {
			t.Errorf("Mount parsing failed: %+v", m)
		}
	}
}
