#!/bin/bash

# ai-shell.sh: Portable, sandboxed development environment launcher.

function _ai_shell_get_project_info() {
    # Calculate hash of full path for uniqueness
    PROJECT_HASH=$(echo -n "$PWD" | shasum -a 256 | cut -c1-12)
    
    # Get directory name, replace non-alnum with -, squeeze repeating dashes
    PROJECT_NAME=$(basename "$PWD" | sed 's/[^a-zA-Z0-9]/-/g' | tr -s '-')
    
    VOL_NAME="ai-home-${PROJECT_NAME}-${PROJECT_HASH}"
    CONTAINER_NAME="ai-shell-${PROJECT_NAME}-${PROJECT_HASH}"
}

function _ai_shell_resolve_config() {
    # 1. Resolve Config File
    # Priority: Env Var -> Local Project Config (upward search) -> Global User Config
    if [ -n "$AI_SHELL_CONFIG" ]; then
        HOST_CONFIG="$AI_SHELL_CONFIG"
    else
        # Look for .ai-shell.yaml in current and parent directories
        local curr="$PWD"
        while [ "$curr" != "/" ]; do
            if [ -f "$curr/.ai-shell.yaml" ]; then
                local LOCAL_CONFIG="$curr/.ai-shell.yaml"
                local TRUST_DIR="$HOME/.local/share/ai-shell/trusted"
                mkdir -p "$TRUST_DIR"

                # Calculate Hash (Portable: macOS 'shasum', Linux 'sha256sum')
                local FILE_HASH
                if command -v shasum >/dev/null 2>&1; then
                    FILE_HASH=$(shasum -a 256 "$LOCAL_CONFIG" | awk '{print $1}')
                else
                    FILE_HASH=$(sha256sum "$LOCAL_CONFIG" | awk '{print $1}')
                fi

                # Check Trust (Flag OR Previously Trusted)
                # Note: We access TRUST_CONFIG from caller scope
                if [ "$TRUST_CONFIG" = true ] || [ -f "$TRUST_DIR/$FILE_HASH" ]; then
                    HOST_CONFIG="$LOCAL_CONFIG"
                else
                    # Interactive Prompt
                    if [ -t 0 ]; then
                        echo "⚠️  Found project configuration: $LOCAL_CONFIG"
                        echo "   This file can modify environment variables and registry credentials."
                        echo "   Fingerprint: $FILE_HASH"
                        read -r -p "   Do you trust this configuration? [y/N] " response
                        if [[ "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
                            HOST_CONFIG="$LOCAL_CONFIG"
                            touch "$TRUST_DIR/$FILE_HASH" # Persist trust
                        else
                            echo "   Skipping local configuration."
                        fi
                    else
                        echo "⚠️  Ignoring untrusted local configuration in non-interactive mode."
                        echo "   Use --trust-config to enable."
                    fi
                fi
                break
            fi
            curr=$(dirname "$curr")
        done
    fi
    
    # Fallback to global config
    if [ -z "$HOST_CONFIG" ] && [ -f "$HOME/.config/ai-shell/config.yaml" ]; then
        HOST_CONFIG="$HOME/.config/ai-shell/config.yaml"
    fi

    # 2. Parse Config Content
    if [ -n "$HOST_CONFIG" ] && [ -f "$HOST_CONFIG" ]; then
        if [ "$VERBOSE" = true ]; then
            echo "   Config: $HOST_CONFIG"
        fi

        # Check for yq
        if ! command -v yq >/dev/null 2>&1; then
            echo "⚠️  'yq' not found. Advanced configuration (env_vars, mounts, args) will be ignored."
        else
            # Mount the config itself
            CRED_MOUNTS+=("-v" "$HOST_CONFIG:/etc/ai-shell/config.yaml:ro")
            
            # Custom Env Vars
            local CONFIG_ENV_VARS
            CONFIG_ENV_VARS=$(yq -r '.env_vars[]' "$HOST_CONFIG" 2>/dev/null | tr '\n' ' ')
            if [ -n "$CONFIG_ENV_VARS" ]; then
                VARS_TO_PASS="$CONFIG_ENV_VARS"
                if [ "$VERBOSE" = true ]; then
                    echo "   🔑 Custom Env Vars: Loaded from config"
                fi
            fi

            # Custom Mounts
            local CUSTOM_MOUNTS
            CUSTOM_MOUNTS=$(yq -r '.mounts[] | "\(.source)|\(.target)|\(.options // "ro")"' "$HOST_CONFIG" 2>/dev/null)
            
            if [ -n "$CUSTOM_MOUNTS" ]; then
                local OLD_IFS="$IFS"
                IFS=$'\n'
                for mount in $CUSTOM_MOUNTS; do
                    local src=$(echo "$mount" | cut -d'|' -f1)
                    local tgt=$(echo "$mount" | cut -d'|' -f2)
                    local opts=$(echo "$mount" | cut -d'|' -f3)
                    
                    eval "src=\"$src\""
                    eval "tgt=\"$tgt\""

                    if [ -e "$src" ]; then
                        CRED_MOUNTS+=("-v" "$src:$tgt:$opts")
                        if [ "$VERBOSE" = true ]; then
                            echo "   💾 Custom Mount: $src -> $tgt ($opts)"
                        fi
                    fi
                done
                IFS="$OLD_IFS"
            fi

            # Podman Args
            local CUSTOM_ARGS
            CUSTOM_ARGS=$(yq -r '.podman_args[]' "$HOST_CONFIG" 2>/dev/null)
            if [ -n "$CUSTOM_ARGS" ]; then
                local OLD_IFS="$IFS"
                IFS=$'\n'
                for arg in $CUSTOM_ARGS; do
                    EXTRA_PODMAN_ARGS+=("$arg")
                    if [ "$VERBOSE" = true ]; then
                        echo "   ⚙️  Podman Arg: $arg"
                    fi
                done
                IFS="$OLD_IFS"
            fi
        fi
    fi
}


function ai-shell() {
    local IMAGE_NAME="ai-shell:latest"
    local VERBOSE=false
    local MOUNT_SSH=false
    local TRUST_CONFIG=false
    local REUSE=false
    local NET_HOST=false

    # OS Detection
    local OS_NAME
    OS_NAME=$(uname -s)
    local HOST_HOME_ROOT
    if [ "$OS_NAME" = "Darwin" ]; then
        HOST_HOME_ROOT="/Users"
    else
        HOST_HOME_ROOT="/home"
    fi

    while [[ "$#" -gt 0 ]]; do
        case $1 in
            -v|--verbose) VERBOSE=true ;;
            --ssh) MOUNT_SSH=true ;;
            --trust-config) TRUST_CONFIG=true ;;
            --reuse) REUSE=true ;;
            --net-host) NET_HOST=true ;;
            *) echo "Unknown parameter passed: $1"; return 1 ;;
        esac
        shift
    done
    
    _ai_shell_get_project_info
    
    echo "Entering AI Shell..."
    
    # Reuse Logic
    if [ "$REUSE" = true ] && podman container exists "$CONTAINER_NAME"; then
        if [ "$(podman container inspect -f '{{.State.Running}}' "$CONTAINER_NAME")" = "true" ]; then
            echo "   Reusing running container..."
            podman exec -it "$CONTAINER_NAME" zsh
            return 0
        else
            echo "   Restarting existing container..."
            podman start -ai "$CONTAINER_NAME"
            return 0
        fi
    fi

    local EXTRA_PODMAN_ARGS=()
    if [ "$NET_HOST" = true ]; then
        EXTRA_PODMAN_ARGS+=("--network=host")
    fi

    local FINAL_ENV_FLAGS=()
    if [ "$VERBOSE" = true ]; then
        FINAL_ENV_FLAGS+=("-e" "AI_SHELL_VERBOSE=true")
        echo "   Project: $PWD"
        echo "   Persistence Volume: $VOL_NAME"
        echo "   OS: $OS_NAME (Home Root: $HOST_HOME_ROOT)"
        if [ "$MOUNT_SSH" = true ]; then
            echo "   SSH Agent: Mounted"
        else
            echo "   SSH Agent: Not mounted (use --ssh to enable)"
        fi
    fi

    # 1. Environment Variable Handling
    local DEFAULT_VARS="CLAUDE_CODE_USE_VERTEX CLOUD_ML_REGION ANTHROPIC_VERTEX_PROJECT_ID GOOGLE_CLOUD_PROJECT VERTEX_LOCATION GEMINI_API_KEY QUAY_USER QUAY_TOKEN RH_REGISTRY_USER RH_REGISTRY_TOKEN GH_USER GH_TOKEN"
    local VARS_TO_PASS="${AI_SHELL_ENV_VARS:-$DEFAULT_VARS}"
    local ENV_FLAGS=""
    
    # 2. Config Mounts (Read-Only)
    local CRED_MOUNTS=()
    if [ -d "$HOME/.config/gcloud" ]; then
        CRED_MOUNTS+=("-v" "$HOME/.config/gcloud:$HOST_HOME_ROOT/$USER/.config/gcloud:ro")
    fi
    if [ -d "$HOME/.claude" ]; then
        CRED_MOUNTS+=("-v" "$HOME/.claude:$HOST_HOME_ROOT/$USER/.claude.host:ro")
    fi
    
    # Mount SSH if requested
    if [ "$MOUNT_SSH" = true ]; then
        CRED_MOUNTS+=("-v" "$HOME/.ssh:$HOST_HOME_ROOT/$USER/.ssh:ro")
    fi

    local HOST_CONFIG=""
    _ai_shell_resolve_config

    local EXTRA_PODMAN_ARGS=()

    if [ -n "$HOST_CONFIG" ] && [ -f "$HOST_CONFIG" ]; then
        # Check for yq requirement
        if ! command -v yq >/dev/null 2>&1; then
            echo "⚠️  'yq' not found. Advanced configuration (env_vars, mounts, args) will be ignored."
            echo "   Please install yq (https://github.com/mikefarah/yq) to enable these features."
        else
            CRED_MOUNTS+=("-v" "$HOST_CONFIG:/etc/ai-shell/config.yaml:ro")
            if [ "$VERBOSE" = true ]; then
                echo "   Config: $HOST_CONFIG"
            fi
            
            # 1. Custom Env Vars
            # yq: extract list items as space-separated string
            local CONFIG_ENV_VARS
            CONFIG_ENV_VARS=$(yq -r '.env_vars[]' "$HOST_CONFIG" 2>/dev/null | tr '\n' ' ')
            if [ -n "$CONFIG_ENV_VARS" ]; then
                VARS_TO_PASS="$CONFIG_ENV_VARS"
                if [ "$VERBOSE" = true ]; then
                    echo "   🔑 Custom Env Vars: Loaded from config"
                fi
            fi

            # 2. Custom Mounts
            # Format: source|target|options
            local CUSTOM_MOUNTS
            CUSTOM_MOUNTS=$(yq -r '.mounts[] | "\(.source)|\(.target)|\(.options // "ro")"' "$HOST_CONFIG" 2>/dev/null)
            
            if [ -n "$CUSTOM_MOUNTS" ]; then
                local OLD_IFS="$IFS"
                IFS=$'\n'
                for mount in $CUSTOM_MOUNTS; do
                    local src=$(echo "$mount" | cut -d'|' -f1)
                    local tgt=$(echo "$mount" | cut -d'|' -f2)
                    local opts=$(echo "$mount" | cut -d'|' -f3)
                    
                    # Expand environment variables in path (e.g. $HOME, $KUBECONFIG)
                    eval "src=\"$src\""
                    eval "tgt=\"$tgt\""

                    if [ -e "$src" ]; then
                        CRED_MOUNTS+=("-v" "$src:$tgt:$opts")
                        if [ "$VERBOSE" = true ]; then
                            echo "   💾 Custom Mount: $src -> $tgt ($opts)"
                        fi
                    fi
                done
                IFS="$OLD_IFS"
            fi

            # 3. Podman Args
            local CUSTOM_ARGS
            CUSTOM_ARGS=$(yq -r '.podman_args[]' "$HOST_CONFIG" 2>/dev/null)
            if [ -n "$CUSTOM_ARGS" ]; then
                local OLD_IFS="$IFS"
                IFS=$'\n'
                for arg in $CUSTOM_ARGS; do
                    EXTRA_PODMAN_ARGS+=("$arg")
                    if [ "$VERBOSE" = true ]; then
                        echo "   ⚙️  Podman Arg: $arg"
                    fi
                done
                IFS="$OLD_IFS"
            fi
        fi
    fi

    # 3. Ensure Volume Exists
    podman volume inspect "$VOL_NAME" > /dev/null 2>&1 || podman volume create "$VOL_NAME" > /dev/null

    # Cleanup any lingering container for this project
    if podman container exists "$CONTAINER_NAME"; then
        podman rm -f "$CONTAINER_NAME" >/dev/null 2>&1
    fi

    # 4. Run Podman
    # Portable list splitting (Zsh vs Bash)
    local VAR_LIST
    if [ -n "$ZSH_VERSION" ]; then
        VAR_LIST=(${=VARS_TO_PASS})
    else
        VAR_LIST=($VARS_TO_PASS)
    fi

    local FINAL_ENV_FLAGS=()
    for var in "${VAR_LIST[@]}"; do
        eval "val=\"\${$var}\""
        if [ -n "$val" ]; then
            FINAL_ENV_FLAGS+=("-e" "$var")
            if [ "$VERBOSE" = true ]; then
                echo "   🔑 Passing: $var"
            fi
        fi
    done

    podman run -it --rm \
        --name "$CONTAINER_NAME" \
        --hostname "ai-box" \
        --security-opt label=disable \
        --userns=keep-id \
        "${FINAL_ENV_FLAGS[@]}" \
        "${CRED_MOUNTS[@]}" \
        "${EXTRA_PODMAN_ARGS[@]}" \
        -v "$PWD":"$PWD" \
        -v "$VOL_NAME":"$HOST_HOME_ROOT/$USER" \
        -v "$HOME/.gitconfig":"$HOST_HOME_ROOT/$USER/.gitconfig.host:ro" \
        -w "$PWD" \
        "$IMAGE_NAME" \
        zsh
}

function ai-shell-export-env() {
    # 1. Environment Variable Handling
    local DEFAULT_VARS="CLAUDE_CODE_USE_VERTEX CLOUD_ML_REGION ANTHROPIC_VERTEX_PROJECT_ID GOOGLE_CLOUD_PROJECT VERTEX_LOCATION GEMINI_API_KEY QUAY_USER QUAY_TOKEN RH_REGISTRY_USER RH_REGISTRY_TOKEN GH_USER GH_TOKEN"
    local VARS_TO_PASS="${AI_SHELL_ENV_VARS:-$DEFAULT_VARS}"
    
    # Init vars for helper
    local CRED_MOUNTS=()
    local EXTRA_PODMAN_ARGS=()
    local HOST_CONFIG=""
    local TRUST_CONFIG=false 
    local VERBOSE=false

    _ai_shell_resolve_config

    # Output .env format
    echo "# Generated by ai-shell for: $PWD"
    if [ -n "$HOST_CONFIG" ]; then
        echo "# Source Config: $HOST_CONFIG"
    fi
    
    local COUNT=0
    # Portable list splitting (Zsh vs Bash)
    local VAR_LIST
    if [ -n "$ZSH_VERSION" ]; then
        VAR_LIST=(${=VARS_TO_PASS})
    else
        VAR_LIST=($VARS_TO_PASS)
    fi

    for var in "${VAR_LIST[@]}"; do
        # Portable indirect expansion
        eval "val=\"\${$var}\""
        if [ -n "$val" ]; then
            echo "$var=$val"
            COUNT=$((COUNT + 1))
        fi
    done
    
    if [ "$COUNT" -eq 0 ]; then
        echo "# No matching environment variables found in host session."
    fi
}

function ai-shell-cleanup() {
    local VERBOSE=false
    if [[ "$1" == "-v" || "$1" == "--verbose" ]]; then
        VERBOSE=true
    fi

    _ai_shell_get_project_info

    echo "Cleaning up ai-shell environment..."
    if [ "$VERBOSE" = true ]; then
        echo "   Project: $PWD"
        echo "   Name: $PROJECT_NAME"
        echo "   Hash: $PROJECT_HASH"
    fi

    # Remove Container
    if podman container exists "$CONTAINER_NAME" >/dev/null 2>&1; then
        if [ "$VERBOSE" = true ]; then echo "   Removing container: $CONTAINER_NAME"; fi
        podman rm -f "$CONTAINER_NAME" >/dev/null 2>&1
    fi

    # Remove Volume
    if podman volume exists "$VOL_NAME" >/dev/null 2>&1; then
        if [ "$VERBOSE" = true ]; then echo "   Removing volume: $VOL_NAME"; fi
        podman volume rm "$VOL_NAME" >/dev/null 2>&1
    fi

    echo "   Cleanup complete."
}

function ai-shell-build() {
    # Resolve script directory to build from anywhere
    local SCRIPT_DIR
    if [ -n "$BASH_SOURCE" ]; then
        SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    elif [ -n "$ZSH_VERSION" ]; then
        SCRIPT_DIR="$(cd "$(dirname "${(%):-%x}")" && pwd)"
    else
        SCRIPT_DIR="$PWD"
    fi

    # OS Detection
    local OS_NAME
    OS_NAME=$(uname -s)
    local HOST_HOME_ROOT
    if [ "$OS_NAME" = "Darwin" ]; then
        HOST_HOME_ROOT="/Users"
    else
        HOST_HOME_ROOT="/home"
    fi

    echo "Building AI Shell Image ($OS_NAME)..."
    podman build -t ai-shell:latest \
        --build-arg "HOST_USER=$USER" \
        --build-arg "HOST_HOME_ROOT=$HOST_HOME_ROOT" \
        -f "$SCRIPT_DIR/Containerfile" "$SCRIPT_DIR"
}
