package container

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type ProjectInfo struct {
	Hash          string
	Name          string
	VolumeName    string
	ContainerName string
}

func GetProjectInfo(path string) ProjectInfo {
	hash := sha256.Sum256([]byte(path))
	hashStr := fmt.Sprintf("%x", hash)[:12]

	projName := filepath.Base(path)
	reg := regexp.MustCompile("[^a-zA-Z0-9]+")
	projName = reg.ReplaceAllString(projName, "-")
	projName = strings.Trim(projName, "-")

	return ProjectInfo{
		Hash:          hashStr,
		Name:          projName,
		VolumeName:    fmt.Sprintf("ai-home-%s-%s", projName, hashStr),
		ContainerName: fmt.Sprintf("ai-shell-%s-%s", projName, hashStr),
	}
}
