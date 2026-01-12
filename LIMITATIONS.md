# Known Limitations

This document lists known constraints and behaviors of `ai-shell` that may affect specific tools or workflows. These are often intentional trade-offs for security and isolation.

## Claude Code (Anthropic)

### Marketplace Updates
- **Issue**: Claude may report "Failed to install Anthropic marketplace" on startup.
- **Cause**: `ai-shell` mounts your host's `~/.claude` directory as Read-Only to prevent AI agents from modifying your global configuration or exfiltrating data.
- **Impact**: Claude cannot update its marketplace components or install new plugins while running *inside* the shell.
- **Workaround**: If you need to update Claude plugins or the marketplace, run `claude` once on your host machine to perform the update. The container will see the updated files immediately due to the symlink and mount.

## Filesystem & Portability

### Home Directory Mapping (Personalized Image)
- **Behavior**: The container mirrors your host's home path (e.g., `/Users/yourname`) and is built with your current `$USER`.
- **Limitation**: This provides excellent path fidelity (important for tools like Claude session IDs), but it means the resulting image is personalized to your environment.
- **Workaround**: It is intended that each user builds the image locally using `ai-shell-build`. This ensures the paths match their specific host environment.
