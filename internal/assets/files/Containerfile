FROM registry.fedoraproject.org/fedora:41

# 1. Install core tools + gh + skopeo + jq
RUN dnf install -y 'dnf-command(config-manager)' \
    && dnf config-manager addrepo --from-repofile=https://cli.github.com/packages/rpm/gh-cli.repo \
    && dnf install -y \
        git \
        gh \
        skopeo \
        jq \
        zsh \
        vim \
        neovim \
        ripgrep \
        fd-find \
        nodejs \
        python3 \
        python3-pip \
        golang \
        sudo \
        which \
        man-db \
        procps-ng \
        findutils \
        iputils \
        hostname \
        wget \
        sqlite \
        kubernetes-client \
    && dnf clean all

# 1b. Install AI Agent CLI tools via npm
RUN npm install -g @anthropic-ai/claude-code @google/gemini-cli

# 2. Install ORAS, glab, yq, and cosign (with Arch Detection)
ARG ORAS_VERSION="1.3.0"
ARG GLAB_VERSION="1.80.4"
ARG YQ_VERSION="4.50.1"
ARG COSIGN_V2_VERSION="2.6.1"
ARG COSIGN_V3_VERSION="3.0.3"

RUN ARCH=$(uname -m) && \
    case "$ARCH" in \
        x86_64)  BIN_ARCH="amd64" ;; \
        aarch64) BIN_ARCH="arm64" ;; \
        *) echo "Unsupported architecture: $ARCH"; exit 1 ;; \
    esac && \
    # Install ORAS
    curl -sSL "https://github.com/oras-project/oras/releases/download/v${ORAS_VERSION}/oras_${ORAS_VERSION}_linux_${BIN_ARCH}.tar.gz" -o oras.tar.gz && \
    mkdir -p /usr/local/bin/ && \
    tar -zxf oras.tar.gz -C /usr/local/bin/ oras && \
    rm oras.tar.gz && \
    # Install glab
    curl -sSL "https://gitlab.com/gitlab-org/cli/-/releases/v${GLAB_VERSION}/downloads/glab_${GLAB_VERSION}_linux_${BIN_ARCH}.tar.gz" -o glab.tar.gz && \
    tar -zxf glab.tar.gz -C /usr/local/bin/ bin/glab && \
    rm glab.tar.gz && \
    # Install yq
    curl -sSL "https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION}/yq_linux_${BIN_ARCH}" -o /usr/bin/yq && \
    chmod +x /usr/bin/yq && \
    # Install cosign v2
    curl -sSL "https://github.com/sigstore/cosign/releases/download/v${COSIGN_V2_VERSION}/cosign-linux-${BIN_ARCH}" -o /usr/local/bin/cosign-v2 && \
    chmod +x /usr/local/bin/cosign-v2 && \
    # Install cosign v3
    curl -sSL "https://github.com/sigstore/cosign/releases/download/v${COSIGN_V3_VERSION}/cosign-linux-${BIN_ARCH}" -o /usr/local/bin/cosign-v3 && \
    chmod +x /usr/local/bin/cosign-v3 && \
    # Default cosign to v2
    ln -s /usr/local/bin/cosign-v2 /usr/local/bin/cosign && \
    # Install Starship prompt
    curl -sS https://starship.rs/install.sh | sh -s -- -y >/dev/null

# 4. Create the 'ai' user
RUN useradd -m -s /bin/zsh -u 1000 ai \
    && echo "ai ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/ai

# 5. Path Fidelity Prep
# Create the host-matching home path and ensure 'ai' owns it.
ARG HOST_USER
ARG HOST_HOME_ROOT
RUN mkdir -p ${HOST_HOME_ROOT}/${HOST_USER} \
    && chown ai:ai ${HOST_HOME_ROOT}/${HOST_USER} \
    && echo 'eval "$(starship init zsh)"' >> /home/ai/.zshrc \
    && echo 'bindkey "^[[H" beginning-of-line' >> /home/ai/.zshrc \
    && echo 'bindkey "^[[F" end-of-line' >> /home/ai/.zshrc \
    && echo 'bindkey "^[[3~" delete-char' >> /home/ai/.zshrc \
    && echo 'bindkey "^[[1;5C" forward-word' >> /home/ai/.zshrc \
    && echo 'bindkey "^[[1;5D" backward-word' >> /home/ai/.zshrc \
    && ln -s /home/ai/.zshrc ${HOST_HOME_ROOT}/${HOST_USER}/.zshrc

# 6. Git Credential Fix
RUN git config --system credential.helper store

# 7. Setup Entrypoint
COPY configure.sh /usr/local/bin/configure.sh
RUN chmod +x /usr/local/bin/configure.sh

# 7b. Default Config
RUN mkdir -p /etc/ai-shell
COPY config.default.yaml /etc/ai-shell/config.yaml

USER ai
WORKDIR ${HOST_HOME_ROOT}/${HOST_USER}

# 8. Initialize the wrapper config
RUN touch /home/ai/.gitconfig.host \
    && ln -s /home/ai/.gitconfig.host ${HOST_HOME_ROOT}/${HOST_USER}/.gitconfig.host
RUN echo "[include]" > /home/ai/.gitconfig && \
    echo "  path = ${HOST_HOME_ROOT}/${HOST_USER}/.gitconfig.host" >> /home/ai/.gitconfig && \
    echo "[credential]" >> /home/ai/.gitconfig && \
    echo "  helper = store" >> /home/ai/.gitconfig
RUN ln -s /home/ai/.gitconfig ${HOST_HOME_ROOT}/${HOST_USER}/.gitconfig

ENTRYPOINT ["/usr/local/bin/configure.sh"]
CMD ["/bin/zsh"]