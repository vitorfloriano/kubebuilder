# Use the Go base as a utility, but we'll strip the branding
ARG VARIANT="2-1.25-bookworm"
FROM mcr.microsoft.com/devcontainers/go:${VARIANT}

ARG TARGETARCH
ARG TARGETOS

# 1. System Setup
# We install fuse-overlayfs for nested storage stability on Fedora hosts
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
    build-essential \
    bash-completion \
    ca-certificates \
    curl \
    sudo \
    fuse-overlayfs \
    iptables \
    iproute2 \
    git

# 2. Install K8s Tools (Pre-built binaries)
# envtest
RUN curl -sSL https://github.com/kubernetes-sigs/controller-runtime/releases/latest/download/setup-envtest-${TARGETOS:-linux}-${TARGETARCH:-amd64} -o /usr/local/bin/setup-envtest \
    && chmod +x /usr/local/bin/setup-envtest \
    && /usr/local/bin/setup-envtest use 1.33.0 --bin-dir /usr/local/bin --force

# controller-gen
RUN curl -sSL https://github.com/kubernetes-sigs/controller-tools/releases/latest/download/controller-gen-${TARGETOS:-linux}-${TARGETARCH:-amd64} -o /usr/local/bin/controller-gen \
    && chmod +x /usr/local/bin/controller-gen

# kustomize
RUN if [ "$TARGETARCH" = "arm64" ]; then ARCH="arm64"; else ARCH="amd64"; fi \
    && curl -sSL https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv5.5.0/kustomize_v5.5.0_${TARGETOS:-linux}_${ARCH}.tar.gz | tar -xz -C /usr/local/bin \
    && chmod +x /usr/local/bin/kustomize

# kind
RUN curl -Lo /usr/local/bin/kind https://kind.sigs.k8s.io/dl/v0.22.0/kind-linux-amd64 \
    && chmod +x /usr/local/bin/kind

# 3. The "Identity Swap" (Renaming vscode to dev)
# This is the anti-branding move. We also fix the sudoers file.
RUN usermod -l dev vscode \
    && groupmod -n dev vscode \
    && usermod -d /home/dev -m dev \
    && sed -i 's/vscode/dev/g' /etc/sudoers.d/vscode \
    && mv /etc/sudoers.d/vscode /etc/sudoers.d/dev

# 4. Handover & Default Shell
# We give 'dev' ownership of the tools so E2E tests can manage them
RUN chown -R dev:dev /usr/local/bin /go \
    && chsh -s /bin/bash dev

USER dev
ENV KUBEBUILDER_ASSETS=/usr/local/bin
