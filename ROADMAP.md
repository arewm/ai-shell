# Roadmap

This roadmap outlines the evolution of `ai-shell` from a personal productivity tool into a standardized, cross-platform environment for AI agent development.

## Phase 1: Robust Foundation (Complete)
- [x] **Go Binary**: Ported bash logic to a compiled binary for speed and safety.
- [x] **Path Fidelity**: Solved tool compatibility by mirroring host home paths in the container.
- [x] **Security Defaults**: Established a "Blast Shield" with read-only mounts and default-deny for SSH.
- [x] **TOFU (Trust on First Use)**: Prevented accidental credential exfiltration via project-local configs.
- [x] **Cross-Platform**: Support for macOS and Linux.

## Phase 2: Unification & Profiles (Next)

### Ecosystem Consistency (Aligning with DevContainers/devaipod)
- [ ] **Standard Schema**: Migrate from `config.yaml` to standard `devcontainer.json` for configuration.
- [ ] **Sharable Images**: Move path mirroring from Build-Time to Runtime (Entrypoint).
    - *Approach*: Container starts as root to create `/Users/$USER` symlinks, then drops privileges to `ai`.
    - *Benefit*: One team-wide binary image can adapt to any user's local path structure.
- [ ] **Dotfiles Support**: Implement a standard `dotfiles/` directory or repo mapping, consistent with `devpod` patterns.

### New Functionality
- [ ] **Named Profiles**: Support `~/.config/ai-shell/<name>/` directories for specialized environments.
    - Example: `ai-shell --profile data-science` or `ai-shell --profile k8s-audit`.
- [ ] **Profile Composition**: Allow project configs to inherit from a system profile.
- [ ] **Network Control**: Implement per-container egress filtering (allow models, block generic web/internal ips).

## Phase 3: Deep Convergence (Merging Architectures)

### Unifying with [`devaipod`](https://github.com/cgwalters/devaipod)
- [ ] **Dual Mode**: Support both Interactive Shell (Human + Agent) and Autonomous Task (Headless Agent) modes in one tool.
- [ ] **Nested Isolation**: Adopt `bubblewrap` (bwrap) or microVMs inside the container for Defense in Depth against rogue agents.
- [ ] **Unified Backend**: Align on a common execution engine (e.g., sharing the DevPod backend or a unified Podman wrapper) to support both local execution and cloud provisioning.

### Advanced Security
- [ ] **Intercept & Audit**: Log and optionally intercept agent commands before execution for human review.
- [ ] **Network Control**: Granular allow-listing for network egress.
