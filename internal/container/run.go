package container

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/arewm/ai-shell/internal/config"
)

type RunOptions struct {
	Verbose    bool
	Reuse      bool
	NetHost    bool
	MountSSH   bool
	Config     *config.Config
	ConfigPath string
	ImageName  string
	Profile    string
}

func Run(opts RunOptions) error {
	pwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// 1. Get Project Info
	info := GetProjectInfo(pwd)
	
	// Append Profile to Container Name to avoid conflicts
	if opts.Profile != "" && opts.Profile != "default" {
		info.ContainerName = fmt.Sprintf("%s-%s", info.ContainerName, opts.Profile)
	}

	// 2. Reuse Logic
	if opts.Reuse {
		// Check if container exists
		checkCmd := exec.Command("podman", "container", "exists", info.ContainerName) //nolint:gosec
		if err := checkCmd.Run(); err == nil {
			// Check if running
			inspectCmd := exec.Command("podman", "container", "inspect", "-f", "{{.State.Running}}", info.ContainerName) //nolint:gosec
			out, _ := inspectCmd.Output()
			isRunning := strings.TrimSpace(string(out)) == "true"

			if isRunning {
				fmt.Println("   Reusing running container...")
				// Must explicitly set user 'ai' because container starts as root
				return execPodman("exec", "-it", "--user", "ai", info.ContainerName, "zsh")
			}
			fmt.Println("   Restarting existing container...")
			return execPodman("start", "-ai", info.ContainerName)
		}
	}

	// 3. Cleanup Old
	_ = exec.Command("podman", "rm", "-f", info.ContainerName).Run() //nolint:gosec

	// 4. Ensure Volume
	_ = exec.Command("podman", "volume", "create", info.VolumeName).Run() //nolint:gosec

	// 5. Construct Flags
	// We must start as root (0:0) to allow configure.sh to setup paths/permissions.
	// The entrypoint will drop privileges to 'ai'.
	args := []string{"run", "-it", "--rm", "--user", "0:0", "--name", info.ContainerName, "--hostname", "ai-box", "--security-opt", "label=disable", "--userns=keep-id"}

	if opts.NetHost {
		args = append(args, "--network=host")
	}

	// Mounts
	home, _ := os.UserHomeDir()
	hostHomeRoot := "/home"
	if runtime.GOOS == "darwin" {
		hostHomeRoot = "/Users"
	}
	user := os.Getenv("USER")
	targetHome := fmt.Sprintf("%s/%s", hostHomeRoot, user)
    fmt.Printf("DEBUG: hostHomeRoot=%s, targetHome=%s\n", hostHomeRoot, targetHome)

	// Runtime Path Info & Standard Mounts
	args = append(args,
		"-e", fmt.Sprintf("HOST_USER=%s", user),
		"-e", fmt.Sprintf("HOST_HOME_ROOT=%s", hostHomeRoot),
		"-v", fmt.Sprintf("%s:%s", pwd, pwd),
		"-w", pwd,
		"-v", fmt.Sprintf("%s:%s", info.VolumeName, targetHome),
		"-v", fmt.Sprintf("%s:%s:ro", filepath.Join(home, ".gitconfig"), "/etc/ai-shell/gitconfig.host"),
	)

	// Config Mounts (Merged)
	if opts.Config != nil {
		// Serialize merged config to temp file
		// We use JSON because yq (in container) can read it and it avoids adding a yaml dep
		configData, err := json.Marshal(opts.Config)
		if err == nil {
			tmpConfig, err := os.CreateTemp("", "ai-shell-config-*.json")
			if err == nil {
				defer os.Remove(tmpConfig.Name())
				if _, err := tmpConfig.Write(configData); err == nil {
					tmpConfig.Close()
					args = append(args, "-v", fmt.Sprintf("%s:/etc/ai-shell/config.yaml:ro", tmpConfig.Name()))
				}
			}
		}
	}

	// Helper to add if exists
	addMount := func(src, target, opts string) {
		if _, err := os.Stat(src); err == nil {
			args = append(args, "-v", fmt.Sprintf("%s:%s:%s", src, target, opts))
		}
	}

	addMount(filepath.Join(home, ".config", "gcloud"), fmt.Sprintf("%s/.config/gcloud", targetHome), "ro")
	addMount(filepath.Join(home, ".claude"), fmt.Sprintf("%s/.claude.host", targetHome), "ro")

	if opts.MountSSH {
		addMount(filepath.Join(home, ".ssh"), fmt.Sprintf("%s/.ssh", targetHome), "ro")
	}

	// Custom Mounts from Config
	if opts.Config != nil {
		for _, m := range opts.Config.Mounts {
			src := os.ExpandEnv(m.Source)
			tgt := os.ExpandEnv(m.Target)
			opt := m.Options
			if opt == "" {
				opt = "ro"
			}
			if _, err := os.Stat(src); err == nil {
				args = append(args, "-v", fmt.Sprintf("%s:%s:%s", src, tgt, opt))
			}
		}
		// Custom Args
		args = append(args, opts.Config.PodmanArgs...)
	}

	// Env Vars
	defaultVars := []string{"CLAUDE_CODE_USE_VERTEX", "CLOUD_ML_REGION", "ANTHROPIC_VERTEX_PROJECT_ID", "GOOGLE_CLOUD_PROJECT", "GEMINI_API_KEY", "GH_TOKEN"}
	varsToPass := defaultVars
	if opts.Config != nil && len(opts.Config.EnvVars) > 0 {
		varsToPass = opts.Config.EnvVars
	}

	for _, v := range varsToPass {
		if val, ok := os.LookupEnv(v); ok && val != "" {
			args = append(args, "-e", v)
		}
	}

	args = append(args, opts.ImageName, "zsh")

	if opts.Verbose {
		fmt.Printf("   Project: %s\n", pwd)
		fmt.Printf("   Persistence Volume: %s\n", info.VolumeName)
		fmt.Printf("   OS: %s (Home Root: %s)\n", runtime.GOOS, hostHomeRoot)
	}

	return execPodman(args...)
}

func execPodman(args ...string) error {
	cmd := exec.Command("podman", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
