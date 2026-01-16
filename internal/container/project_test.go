package container

import (
	"strings"
	"testing"
)

func TestGetProjectInfo(t *testing.T) {
	tests := []struct {
		path         string
		expectedName string
	}{
		{"/Users/alice/projects/my-app", "my-app"},
		{"/home/bob/src/cool_project", "cool_project"},
		{"/var/www/html/weird@name#test", "weird-name-test"},
		{"/tmp/trailing-slash/", "trailing-slash"},
	}

	for _, tt := range tests {
		info := GetProjectInfo(tt.path)

		// Check Name Sanitization
		if info.Name != tt.expectedName {
			t.Errorf("Path: %s, Expected Name: %s, Got: %s", tt.path, tt.expectedName, info.Name)
		}

		// Check Hash Length
		if len(info.Hash) != 12 {
			t.Errorf("Hash length should be 12, got %d", len(info.Hash))
		}

		// Check Format
		expectedVol := "ai-home-" + info.Name + "-" + info.Hash
		if info.VolumeName != expectedVol {
			t.Errorf("Volume name mismatch. Got: %s", info.VolumeName)
		}

		// Check Container Name
		expectedCont := "ai-shell-" + info.Name + "-" + info.Hash
		if info.ContainerName != expectedCont {
			t.Errorf("Container name mismatch. Got: %s", info.ContainerName)
		}

		// Check consistency
		if !strings.HasSuffix(info.VolumeName, info.Hash) {
			t.Error("Volume name should end with hash")
		}
	}
}
