# Contributing to ai-shell

Thank you for your interest in contributing! This project aims to define a safe, reproducible, and ergonomic standard for "Agentic AI" development environments.

## Project Design Goals

Any contribution must align with these core tenets:

1.  **Security ("The Blast Shield")**: The primary goal is to protect the host system from potentially malicious or erroneous AI actions. We default to "deny" for permissions. We explicitly strictly limit mounts.
    *   *Anti-Pattern:* Mounting the entire host `$HOME` or running as root on the host.
2.  **Project Isolation**: One project should never affect another.
    *   *Pattern:* Per-project persistent home volumes.
    *   *Anti-Pattern:* A single shared global "home" volume.
3.  **Path Fidelity**: The environment must "feel" like the host to the AI.
    *   *Pattern:* Mirroring the `$PWD` inside the container so absolute paths in configs (like `.gitconfig` `includeIf`) resolve correctly.
4.  **Minimal Host Dependencies**: The host should *only* need Podman and this script. All other tools live inside.

## What makes a good contribution?

We welcome PRs that enhance usability without compromising security.

### Reasonable Contributions
- **Universal Tooling**: Adding standard, lightweight tools to the base image (e.g., `rsync`, `curl`, `jq`, `yq`, `glab`, `cosign`, `claude`, `gemini`) that 99% of developers need.
- **Credential Helpers**: Improvements to how we securely forward auth tokens or handle SCM authentication.
- **Platform Support**: Making `lib.sh` compatible with other Linux distributions or shell environments.

## Development Standards

Please adhere to the following when writing code or documentation:

### Code & Tooling
- **Podman Only**: We rely on Podman's rootless architecture. Do not introduce Docker-specific dependencies.
- **Shell**: Write portable Bash or Zsh.
- **Clean History**: Use [Conventional Commits](https://www.conventionalcommits.org/) (e.g., `feat:`, `fix:`, `docs:`).
- **No Sign-offs**: Do not add `Signed-off-by` lines.
- **Attribution**: If you use an AI tool, add `Assisted-by: <tool/model>` to the commit message.
- **Line Length**: Keep lines in documentation (Markdown) under 120 characters for readability.

### UX
- **Verbose Flags**: By default, the tool should be silent/minimal. Use `--verbose` for debugging info.
- **Error Messages**: Be descriptive. "Login failed" is bad. "Login failed: QUAY_TOKEN environment variable is missing" is good.
