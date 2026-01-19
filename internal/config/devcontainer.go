package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/tailscale/hujson"
)

// DevContainerConfig represents the subset of devcontainer.json we support.
type DevContainerConfig struct {
	Image        string             `json:"image,omitempty"`
	Build        *DevContainerBuild `json:"build,omitempty"`
	RunArgs      []string           `json:"runArgs,omitempty"`
	Mounts       []string           `json:"mounts,omitempty"` // Strings like "source=...,target=..."
	ContainerEnv map[string]string  `json:"containerEnv,omitempty"`
	RemoteEnv    map[string]string  `json:"remoteEnv,omitempty"`
	// We could add 'features' later if we support them
}

type DevContainerBuild struct {
	Dockerfile string            `json:"dockerfile,omitempty"`
	Context    string            `json:"context,omitempty"`
	Args       map[string]string `json:"args,omitempty"`
}

// ParseDevContainer reads and parses a JSONC file.
func ParseDevContainer(path string) (*DevContainerConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Standardize (remove comments, trailing commas)
	stdData, err := hujson.Standardize(data)
	if err != nil {
		return nil, fmt.Errorf("invalid jsonc format: %w", err)
	}

	var cfg DevContainerConfig
	if err := json.Unmarshal(stdData, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal devcontainer: %w", err)
	}

	return &cfg, nil
}

// ConvertDevContainerToConfig adapts the standard format to our internal Config.
func (dc *DevContainerConfig) ToConfig() *Config {
	c := &Config{}

	// Env Vars (Combine ContainerEnv and RemoteEnv)
	// Our Config uses a list of keys to pass through OR key=value?
	// Currently Config.EnvVars is []string.
	// If the user provided a map in devcontainer, we assume they want to SET values.
	// But our current logic is "Pass these vars from host".
	// DevContainer logic is "Set these vars in container".

	// Hybrid Approach:
	// If value is empty or matches key ("VAR": "${localEnv:VAR}"), we treat it as pass-through.
	// If value is set, we might need to handle it.
	// For now, we only support passing variables listed as keys.

	seen := make(map[string]bool)
	for k := range dc.ContainerEnv {
		if !seen[k] {
			c.EnvVars = append(c.EnvVars, k)
			seen[k] = true
		}
	}
	for k := range dc.RemoteEnv {
		if !seen[k] {
			c.EnvVars = append(c.EnvVars, k)
			seen[k] = true
		}
	}

	// Podman Args
	c.PodmanArgs = dc.RunArgs

	// Mounts
	// DevContainer format: "source=${localWorkspaceFolder},target=/workspace,type=bind"
	// Our format: Mount struct { Source, Target, Options }
	for _, mStr := range dc.Mounts {
		m := parseMountString(mStr)
		if m != nil {
			c.Mounts = append(c.Mounts, *m)
		}
	}

	return c
}

func parseMountString(s string) *Mount {
	// Very basic parser for "source=S,target=T,type=bind"
	// This is fragile but sufficient for a start.
	// We assume comma separation.

	m := &Mount{Options: "rw"} // Default to rw if not specified? Or ro? DevContainer default is rw.

	parts := splitComma(s)
	for _, p := range parts {
		kv := splitEquals(p)
		if len(kv) != 2 {
			continue
		}
		k, v := kv[0], kv[1]

		switch k {
		case "source", "src":
			m.Source = v
		case "target", "dst", "destination":
			m.Target = v
		case "type":
			// ignore for now, assume bind
		case "readonly":
			if v == "true" {
				m.Options = "ro"
			}
		}
	}

	if m.Source == "" || m.Target == "" {
		return nil
	}
	return m
}

func splitComma(s string) []string {
	// naive split, doesn't handle escaped commas
	// TODO: Improve if needed
	var res []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			res = append(res, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		res = append(res, s[start:])
	}
	return res
}

func splitEquals(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '=' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}
