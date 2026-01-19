# ai-shell

`ai-shell` provides a strongly sandboxed, reproducible, and per-project development environment for AI-assisted coding on
macOS and Linux using Podman.

It implements the isolation principles described in [Thoughts on agentic AI coding](https://blog.verbum.org/2025/10/27/thoughts-on-agentic-ai-coding-as-of-oct-2025/)
by providing:

- **Path Mirroring**: The host directory is mounted at the exact same path inside the container, ensuring absolute paths
  in configurations (like Git `includeIf`) remain valid.
- **Per-Project Persistence**: Each project directory gets its own dedicated Podman volume for the home directory,
  preventing cross-project contamination while preserving tool configurations and shell history.
- **OS Integration**: Handles the "symlink trick" to bridge host paths (`/Users` on macOS, `/home` on Linux) to the
  container's home directory and safely injects Git configurations without interfering with host-specific credential helpers.

## Table of Contents
- [Installation](#installation)
- [Usage](#usage)
- [Platform Support](#platform-support)
- [VS Code Dev Containers](#vs-code-dev-containers)
- [CLI Options](#cli-options)
  - [Persistent Sessions](#persistent-sessions--multi-terminal)
  - [Host Networking](#host-networking)
  - [SSH Access](#ssh-access)
  - [Cleanup](#cleanup)
- [Configuration](#configuration)
  - [Custom Configuration](#custom-configuration)
  - [Automatic Authentication](#automatic-authentication)
- [Architecture Support](#architecture-support)
- [Build Customization](#build-customization)

## Installation



1. Ensure [Podman](https://podman.io/) is installed and the Podman machine is running.

2. Clone this repository.

3. Build and install the binary:

   ```bash

   make build

   # Move to a directory in your PATH, or:

   go install github.com/arewm/ai-shell/cmd/ai-shell@latest

   ```



## Usage



Navigate to any project directory and run:

```bash

ai-shell

```



To build the container image (first time or after updates):

```bash

ai-shell build

```

## Platform Support

- **macOS**: Native support. Uses `/Users` mapping.
- **Linux**: Native support. Uses `/home` mapping.

## VS Code Dev Containers

You can use the `ai-shell` image as a base for Dev Containers.

**Important:** VS Code uses its own launcher, so it bypasses `lib.sh`. You must manually mount the configuration file
and provide environment variables.

Example `.devcontainer/devcontainer.json`:
```json
{
  "name": "ai-shell",
  "image": "ai-shell:latest",
  "overrideCommand": false,
  "runArgs": ["--userns=keep-id", "--env-file", ".devcontainer/.env"],
  "mounts": [
    "source=${localEnv:HOME}/.config/ai-shell/config.yaml,target=/etc/ai-shell/config.yaml,type=bind,readonly"
  ],
  "workspaceMount": "source=${localWorkspaceFolder},target=${localWorkspaceFolder},type=bind",
  "workspaceFolder": "${localWorkspaceFolder}",
  "remoteUser": "ai"
}
```

### Helper: Export Environment Variables
To easily populate the `.env` file for VS Code with your active configuration variables:
```bash
# Print to stdout
ai-shell export-env

# Save to a file
ai-shell export-env .devcontainer/.env
```

## CLI Options

Use the `--verbose` or `-v` flag to see details about the project path and persistent volume:
```bash
ai-shell --verbose
```

### Persistent Sessions & Multi-Terminal
By default, `ai-shell` creates a fresh container for each run, wiping any changes made outside of your home volume.
To reconnect to an existing project container:
```bash
ai-shell --reuse
```

**⚠️ Security Warning:** Reusing a container means you are entering an environment that may have been modified by
previous agent runs. Ensure you trust the state of the container before reusing it for sensitive tasks. Without
`--reuse`, this tool forcefully removes any existing container for the project to ensure a clean, reproducible state.

### Host Networking
To access local services (like KinD clusters on `127.0.0.1`) or host-side VPN connections, use host networking:
```bash
ai-shell --net-host
```
**⚠️ Security Warning:** This flag disables network isolation. The AI agent will have full access to your host's
network interfaces, local services, and potential VPN resources.

### SSH Access
By default, your `$HOME/.ssh` directory is **not** mounted to prevent AI agents from using your host identity. If you
explicitly need SSH access for git or other tools:
```bash
ai-shell --ssh
```
**⚠️ Security Warning:** This mounts your personal SSH keys into the container. A malicious or buggy agent could use
these credentials to authenticate to remote services as you.

When SSH is disabled, `ai-shell` automatically configures Git to use your SCM tokens (like `GH_TOKEN`) for
authentication over HTTPS.

*Note: This reduces your ability to control what the agent may have access to as your ssh credentials may be used to
authenticate to services*  

### Cleanup
To remove the persistent volume and any lingering containers for the current project:
```bash
ai-shell cleanup
```
This is useful if you want to completely reset your project environment (history, tool configs, etc.) or free up disk
space. Use `--verbose` to see specific details.

## Configuration

The environment is based on Fedora 41 and includes standard development tools:
- Git, Zsh, Vim, Neovim
- Node.js, Python 3, Go, ORAS, yq, gh, glab, cosign
- Claude Code (`claude`)
- Gemini CLI (`gemini`)
- Kubernetes CLI (`kubectl`)
- Common CLI utilities (ripgrep, fd, procps, etc.)

### Custom Configuration
You can customize registries and SCM settings by creating a configuration file. `ai-shell` resolves the configuration
file in the following priority order (first match wins):

1. **Environment Variable**: `AI_SHELL_CONFIG`
2. **DevContainer**: `.devcontainer/devcontainer.json` or `.devcontainer.json` (Standard format).
3. **Legacy Local**: `.ai-shell.yaml` in the current directory.
4. **User Global**: `~/.config/ai-shell/config.yaml`

**Security Note:**
When a local configuration file is detected, `ai-shell` uses a **Trust on First Use** model:
1.  **First Run**: You will be prompted to trust the configuration (`[y/N]`).
2.  **Persistence**: If trusted, the file's "fingerprint" (hash) is saved to `~/.local/share/ai-shell/trusted/`. You
    won't be asked again.
3.  **Changes**: If the file content changes, the fingerprint changes, and you will be prompted again.

*   **Non-Interactive / CI**: Local configuration is **ignored** by default. Use the `--trust-config` flag to
    forcefully enable it.

```bash
# Trust the local config in a script
ai-shell --trust-config
```

Example `config.yaml` (or `.ai-shell.yaml`):
```yaml
# Optional: Override the list of environment variables to pass into the shell
# If omitted, a default list (AWS, Google, Azure, etc.) is used.
env_vars:
  - GH_TOKEN
  - KUBECONFIG

# Optional: Add custom bind mounts
# Environment variables in 'source' will be expanded.
mounts:
  - source: "$HOME/.kube"
    target: "$HOME/.kube"
    options: ro
  - source: "$KUBECONFIG"
    target: "$KUBECONFIG"
    options: ro

# Optional: Add extra podman run arguments
podman_args:
  - "--network=host"

registries:
  - registry: "quay.io"
    username_env: "QUAY_USER"
    token_env: "QUAY_TOKEN"

scms:
  - host: "github.com"
    token_env: "GH_TOKEN"
  - host: "gitlab.com"
    token_env: "GITLAB_TOKEN"
    username_env: "GITLAB_USER"
```

### Automatic Authentication
- **Registries**: The shell automatically logs into registries defined in your config if the corresponding environment
  variables are set.
- **Git**: If SSH keys are not mounted, `ai-shell` automatically configures Git to use your SCM tokens (like `GH_TOKEN`)
  for HTTPS rewrites. This allows you to `git pull/push` on repositories that were originally cloned via SSH without
  needing your private keys.

### Architecture Support
The image supports both `x86_64` and `aarch64` (Apple Silicon/ARM). The build process automatically detects your
architecture.

### Build Customization
You can override the default versions of tools during the build process:
```bash
ai-shell build --build-arg GLAB_VERSION=1.52.0
```
Available build arguments:
- `ORAS_VERSION`
- `GLAB_VERSION`
- `YQ_VERSION`
- `COSIGN_V2_VERSION`
- `COSIGN_V3_VERSION`

### Included Tools
- **Cosign**: Both v2 and v3 are installed. `cosign` defaults to v2, while `cosign-v3` is available for newer features.
- **GLab**: The official GitLab CLI.
- **ORAS**: OCI Registry As Storage CLI.

### Persistence
Persistence is achieved via Podman volumes named `ai-home-<hash>`, where the hash is derived from the project's absolute
path.
