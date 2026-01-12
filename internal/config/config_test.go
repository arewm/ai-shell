package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindUpward(t *testing.T) {
	// Setup temporary directory structure
	// /tmp/root/
	// /tmp/root/project/
	// /tmp/root/config.yaml

	tmpDir, err := os.MkdirTemp("", "ai-shell-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	projectDir := filepath.Join(tmpDir, "project")
	if mkErr := os.Mkdir(projectDir, 0755); mkErr != nil {
		t.Fatal(mkErr)
	}

	configFile := filepath.Join(tmpDir, ".ai-shell.yaml")
	if wfErr := os.WriteFile(configFile, []byte(""), 0600); wfErr != nil {
		t.Fatal(wfErr)
	}

	// Test finding from project dir
	found, err := findUpward(projectDir, ".ai-shell.yaml")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if found != configFile {
		t.Errorf("Expected %s, got %s", configFile, found)
	}

	// Test finding from root dir
	found, err = findUpward(tmpDir, ".ai-shell.yaml")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if found != configFile {
		t.Errorf("Expected %s, got %s", configFile, found)
	}

	// Test missing
	found, err = findUpward(tmpDir, "nonexistent")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if found != "" {
		t.Errorf("Expected empty string, got %s", found)
	}
}
