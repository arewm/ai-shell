#!/bin/bash
set -e

# configure.sh: Handles authentication before starting the shell using config.yaml

CONFIG_FILE="/etc/ai-shell/config.yaml"

# -----------------------------------------------------------------------------
# 1. Registry Login
# -----------------------------------------------------------------------------
# Ensure consistent auth file for skopeo, oras, and cosign
export DOCKER_CONFIG="/home/ai/.docker"
export REGISTRY_AUTH_FILE="$DOCKER_CONFIG/config.json"
mkdir -p "$DOCKER_CONFIG"
if [ ! -f "$REGISTRY_AUTH_FILE" ]; then
    echo '{"auths":{}}' > "$REGISTRY_AUTH_FILE"
fi

# -----------------------------------------------------------------------------
# 2. Claude Config Import (Surgical Symlinking)
# -----------------------------------------------------------------------------
if [ -d "$HOME/.claude.host" ]; then
    echo "   Configuring Claude environment..."
    rm -rf "$HOME/.claude"
    mkdir -p "$HOME/.claude"
    # Symlink static/large directories from host (RO)
    for dir in agents plugins todos commands; do
        if [ -d "$HOME/.claude.host/$dir" ] && [ ! -d "$HOME/.claude/$dir" ]; then
            ln -s "$HOME/.claude.host/$dir" "$HOME/.claude/$dir"
        fi
    done
    
    # Projects needs to be writable for new sessions
    mkdir -p "$HOME/.claude/projects"
    
    # Symlink config files (RO)
    for file in settings.json session-env session.json; do
        if [ -f "$HOME/.claude.host/$file" ] && [ ! -f "$HOME/.claude/$file" ]; then
            ln -s "$HOME/.claude.host/$file" "$HOME/.claude/$file"
        fi
    done
    # Note: debug/ and history files are left to be created as local writable files
    
    # Pre-create writable state directories for plugins
    mkdir -p "$HOME/.claude/cache" "$HOME/.claude/memory"
fi

# Use yq to iterate over registries. 
# Output format: "registry|username_env|token_env"
REGISTRIES=$(yq -r '.registries[] | "\(.registry)|\(.username_env)|\(.token_env)"' "$CONFIG_FILE")

while IFS='|' read -r REG USER_VAR TOKEN_VAR; do
    # Skip empty lines
    if [ -z "$REG" ]; then continue; fi
    
    # Get values from environment
    VAL_USER="${!USER_VAR}"
    VAL_TOKEN="${!TOKEN_VAR}"

    if [ -n "$VAL_USER" ] && [ -n "$VAL_TOKEN" ]; then
        echo "   Logging into $REG as $VAL_USER..."
        echo "$VAL_TOKEN" | skopeo login "$REG" --username "$VAL_USER" --password-stdin 2>/dev/null
    fi
done <<< "$REGISTRIES"


# -----------------------------------------------------------------------------
# 2. SSH Detection
# -----------------------------------------------------------------------------
MOUNTED_SSH=false
if [ -d "$HOME/.ssh" ] && [ "$(ls -A "$HOME/.ssh" 2>/dev/null)" ]; then
    MOUNTED_SSH=true
fi


# -----------------------------------------------------------------------------
# 3. SCM Configuration
# -----------------------------------------------------------------------------
# Output format: "host|token_env|username_env"
SCMS=$(yq -r '.scms[] | "\(.host)|\(.token_env)|\(.username_env)"' "$CONFIG_FILE")
GIT_CONFIG_IDX=0

while IFS='|' read -r HOST TOKEN_VAR USER_VAR; do
    if [ -z "$HOST" ]; then continue; fi
    # Handle yq null output for missing optional fields
    if [ "$USER_VAR" = "null" ]; then USER_VAR=""; fi

    VAL_TOKEN="${!TOKEN_VAR}"
    # Use the username env var if provided, otherwise empty
    VAL_USER=""
    if [ -n "$USER_VAR" ]; then
        VAL_USER="${!USER_VAR}"
    fi
    
    if [ -n "$VAL_TOKEN" ]; then
        # 3a. GitHub CLI Specifics
        if [ "$HOST" == "github.com" ] && [ "$TOKEN_VAR" == "GH_TOKEN" ]; then
             if [ "$MOUNTED_SSH" = true ]; then
                echo "SSH keys detected. Configuring GitHub CLI for SSH..."
                gh config set git_protocol ssh
            else
                echo "No SSH keys detected. Configuring GitHub CLI for HTTPS..."
                gh config set git_protocol https
            fi
        fi

        # 3b. Git URL Rewrite (For transparent access without SSH keys)
        if [ "$MOUNTED_SSH" = false ]; then
            echo "   Configuring HTTPS rewrite for $HOST..."
            
            # Construct URL based on whether username is available
            # GitLab often requires: https://user:token@gitlab.com/
            # GitHub accepts:        https://token@github.com/
            if [ -n "$VAL_USER" ]; then
                AUTH_URL="https://${VAL_USER}:${VAL_TOKEN}@${HOST}/"
            else
                AUTH_URL="https://${VAL_TOKEN}@${HOST}/"
            fi

            export GIT_CONFIG_KEY_${GIT_CONFIG_IDX}="url.${AUTH_URL}.insteadOf"
            export GIT_CONFIG_VALUE_${GIT_CONFIG_IDX}="git@${HOST}:"
            GIT_CONFIG_IDX=$((GIT_CONFIG_IDX+1))
        fi
    fi
done <<< "$SCMS"

# Finalize Git Config Count
if [ "$GIT_CONFIG_IDX" -gt 0 ]; then
    export GIT_CONFIG_COUNT=$GIT_CONFIG_IDX
fi

# Execute the command (usually zsh)
exec "$@"