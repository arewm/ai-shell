# Manual Test Plan

Run these steps to verify the stability and security of `ai-shell` after major changes.

## 1. Clean Slate
- [ ] Run `ai-shell cleanup` to remove existing containers/volumes.
- [ ] Remove the image: `podman rmi ai-shell:latest`.
- [ ] Clear trust store: `rm ~/.local/share/ai-shell/trusted/*`.

## 2. Build & Install
- [ ] Run `make build`.
- [ ] Run `./ai-shell build`.
- [ ] Verify image exists: `podman images | grep ai-shell`.

## 3. Basic Execution & Path Fidelity
- [ ] Run `./ai-shell`.
- [ ] **Check Path:** Inside shell, run `pwd`. It MUST match your host path (e.g., `/Users/name/...`).
- [ ] **Check User:** Run `whoami`. It MUST be `ai`.
- [ ] **Check Write:** Run `touch testfile`. It MUST succeed and be visible on host.

## 4. Tool Verification
- [ ] **Claude:** Run `claude --version`.
- [ ] **Gemini:** Run `gemini --version`.
- [ ] **Kubectl:** Run `kubectl version --client`.
- [ ] **Git:** Run `git status`. It should work without config errors.

## 5. Persistence & Reuse
- [ ] **Exit** the shell (`exit`).
- [ ] Run `./ai-shell --reuse`. It should restart the container (check `podman ps`).
- [ ] **Multi-Terminal:** Open a new host terminal. Run `./ai-shell --reuse`. It should exec into the *same* container.
- [ ] **State:** Files created in `~` (container home) should persist between restarts.

## 6. Security & Trust
- [ ] Create `.ai-shell.yaml` with `env_vars: [TEST_VAR]`.
- [ ] Run `./ai-shell`. It MUST prompt "Do you trust this configuration?".
- [ ] Answer `N`. `echo $TEST_VAR` should be empty.
- [ ] Run again, answer `y`. `echo $TEST_VAR` should be set (if exported on host).
- [ ] Modify `.ai-shell.yaml`. Run again. It MUST prompt again.

## 7. DevContainer Integration
- [ ] Run `./ai-shell export-env .env.test`.
- [ ] Verify `.env.test` contains your host credentials.
