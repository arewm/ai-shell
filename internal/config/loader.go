package config

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// LoadConfigWithTrust resolves and merges configuration.
// Priority: Project Config merges into Global Config.
func LoadConfigWithTrust(startDir string, autoTrust bool) (*Config, string, error) {
	// 1. Load Global Config
	globalCfg := &Config{}
	home, err := os.UserHomeDir()
	if err == nil {
		globalPath := filepath.Join(home, ".config", "ai-shell", "config.yaml")
		if _, err := os.Stat(globalPath); err == nil {
			c, _, err := loadFile(globalPath)
			if err != nil {
				return nil, "", fmt.Errorf("failed to load global config: %w", err)
			}
			globalCfg = c
		}
	}

	// 2. Resolve Project Config Path
	projectPath := ""
	isDevContainer := false

	if envPath := os.Getenv("AI_SHELL_CONFIG"); envPath != "" {
		projectPath = envPath
	} else {
		// Priority 1: .devcontainer/devcontainer.json
		dcPath, err := findUpward(startDir, ".devcontainer/devcontainer.json")
		if err == nil && dcPath != "" {
			projectPath = dcPath
			isDevContainer = true
		} else {
			// Priority 2: .devcontainer.json
			dcPath, err = findUpward(startDir, ".devcontainer.json")
			if err == nil && dcPath != "" {
				projectPath = dcPath
				isDevContainer = true
			} else {
				// Priority 3: .ai-shell.yaml
				localPath, err := findUpward(startDir, ".ai-shell.yaml")
				if err == nil && localPath != "" {
					projectPath = localPath
				}
			}
		}
	}

	// 3. Load and Merge Project Config
	if projectPath != "" {
		trusted, err := checkTrust(projectPath, autoTrust)
		if err != nil {
			return nil, "", err
		}
		if trusted {
			var projectCfg *Config
			if isDevContainer {
				dc, err := ParseDevContainer(projectPath)
				if err != nil {
					return nil, projectPath, err
				}
				projectCfg = dc.ToConfig()
			} else {
				c, _, err := loadFile(projectPath)
				if err != nil {
					return nil, projectPath, err
				}
				projectCfg = c
			}

			mergeConfig(globalCfg, projectCfg)
			return globalCfg, projectPath, nil
		}
		
		fmt.Println("   Skipping local configuration.")
	}

	return globalCfg, "", nil
}

func mergeConfig(base, override *Config) {
	// EnvVars: Append unique
	seen := make(map[string]bool)
	for _, v := range base.EnvVars {
		seen[v] = true
	}
	for _, v := range override.EnvVars {
		if !seen[v] {
			base.EnvVars = append(base.EnvVars, v)
			seen[v] = true
		}
	}

	// Mounts: Append (Project overrides happen naturally if Podman respects last flag,
	// but strictly we should dedupe by Target. For simplicity, append works as last-mount-wins in Podman usually).
	base.Mounts = append(base.Mounts, override.Mounts...)

	// Args: Append
	base.PodmanArgs = append(base.PodmanArgs, override.PodmanArgs...)

	// Registries: Append
	base.Registries = append(base.Registries, override.Registries...)

	// SCMs: Append
	base.SCMs = append(base.SCMs, override.SCMs...)
}
func loadFile(path string) (*Config, string, error) {
	v := viper.New()
	v.SetConfigFile(path)
	if err := v.ReadInConfig(); err != nil {
		return nil, path, fmt.Errorf("failed to read config: %w", err)
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, path, fmt.Errorf("failed to parse config: %w", err)
	}
	return &cfg, path, nil
}

func checkTrust(path string, autoTrust bool) (bool, error) {
	if autoTrust {
		return true, nil
	}

	// Calculate Hash
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() {
		_ = f.Close()
	}()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, err
	}
	hash := fmt.Sprintf("%x", h.Sum(nil))

	// Check Trust Store
	home, _ := os.UserHomeDir()
	trustDir := filepath.Join(home, ".local", "share", "ai-shell", "trusted")
	trustFile := filepath.Join(trustDir, hash)

	if _, err := os.Stat(trustFile); err == nil {
		return true, nil
	}

	// Prompt
	// Check if interactive
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		fmt.Println("⚠️  Ignoring untrusted local configuration in non-interactive mode.")
		fmt.Println("   Use --trust-config to enable.")
		return false, nil
	}

	fmt.Printf("⚠️  Found project configuration: %s\n", path)
	fmt.Println("   This file can modify environment variables and registry credentials.")
	fmt.Printf("   Fingerprint: %s\n", hash)
	fmt.Print("   Do you trust this configuration? [y/N] ")

	reader := bufio.NewReader(os.Stdin)
	response, _ := reader.ReadString('\n')
	response = strings.TrimSpace(response)

	if strings.EqualFold(response, "y") || strings.EqualFold(response, "yes") {
		if err := os.MkdirAll(trustDir, 0755); err != nil {
			return false, fmt.Errorf("failed to create trust dir: %w", err)
		}
		if err := os.WriteFile(trustFile, []byte{}, 0600); err != nil {
			return false, fmt.Errorf("failed to save trust: %w", err)
		}
		return true, nil
	}

	return false, nil
}

func findUpward(startDir, filename string) (string, error) {
	curr := startDir
	for {
		path := filepath.Join(curr, filename)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			return "", nil // Reached root
		}
		curr = parent
	}
}
