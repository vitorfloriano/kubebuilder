#!/bin/bash
set -e

echo "🚀 Starting Agnostic Post-Install Setup..."

# 1. Agnostic "Self-Healing" for Nested Docker & Kubelet
# If we can't run a simple container, we switch to VFS (storage) and cgroupfs (driver).
# This bypasses the systemd bus requirement for the Kubelet on Fedora/Cgroupv2 hosts.
if ! sudo docker run --rm hello-world >/dev/null 2>&1; then
    echo "🛠️  Heal: Adjusting Docker for nested Cgroup and Storage stability..."
    sudo service docker stop

    # cgroupfs is used because systemd isn't running as an init system inside the container
    sudo bash -c 'echo "{\"exec-opts\": [\"native.cgroupdriver=cgroupfs\"], \"storage-driver\": \"vfs\"}" > /etc/docker/daemon.json'

    # Wipe existing metadata to prevent driver mismatch errors
    sudo rm -rf /var/lib/docker/*
    sudo service docker start
fi

# 2. Git Security Handshake (Fixes Go build exit status 128)
# Marks the workspace as safe regardless of the user ID mismatch between host and container
echo "🔒 Marking workspace as safe for Git..."
git config --global --add safe.directory /workspaces/kubebuilder

# 3. Universal Permissions
# Ensures the 'dev' user owns the Go paths and workspace files
echo "🔑 Finalizing file permissions..."
sudo chmod 666 /var/run/docker.sock
sudo chown -R "$(id -u):$(id -g)" /go /workspaces

# 4. Project Initialization
echo "📦 Downloading Go dependencies..."
go mod download
echo "🔨 Running initial build..."
make build

# 5. Agnostic Shell Completion
echo "⌨️  Configuring shell completions..."
COMPLETIONS_DIR="$HOME/.local/share/bash-completion/completions"
mkdir -p "$COMPLETIONS_DIR"
for tool in kubectl kind helm; do
    if command -v "$tool" >/dev/null; then
        "$tool" completion bash > "$COMPLETIONS_DIR/$tool"
    fi
done

# Source completions in .bashrc if not already present
if ! grep -q "bash-completion" ~/.bashrc; then
    echo '[[ -r "/usr/share/bash-completion/bash_completion" ]] && . "/usr/share/bash-completion/bash_completion"' >> ~/.bashrc
    echo "for f in $COMPLETIONS_DIR/*; do [ -f \"\$f\" ] && . \"\$f\"; done" >> ~/.bashrc
fi

echo "✅ Setup Complete! You are ready to run: make test-e2e-local"
