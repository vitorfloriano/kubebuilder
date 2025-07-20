# Use the official Go image as base
FROM golang:1.24-bullseye

# Set environment variables
ENV DEBIAN_FRONTEND=noninteractive
ENV GOPROXY=https://proxy.golang.org/
ENV GOSUMDB=sum.golang.org
ENV CGO_ENABLED=0
ENV PATH=/usr/local/go/bin:/go/bin:$PATH

# Install system dependencies
RUN apt-get update && apt-get install -y \
    curl \
    wget \
    git \
    make \
    build-essential \
    ca-certificates \
    gnupg \
    lsb-release \
    sudo \
    vim \
    nano \
    jq \
    unzip \
    bash-completion \
    && rm -rf /var/lib/apt/lists/*

# Create a non-root user
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

RUN groupadd --gid $USER_GID $USERNAME \
    && useradd --uid $USER_UID --gid $USER_GID -m $USERNAME \
    && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME

# Set up Go workspace
RUN mkdir -p /go/bin /go/src /go/pkg && chown -R $USERNAME:$USERNAME /go

# Switch to non-root user
USER $USERNAME

# Install Go tools commonly used in kubebuilder development
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2 && \
    go install golang.org/x/tools/cmd/goimports@latest && \
    go install github.com/onsi/ginkgo/v2/ginkgo@latest && \
    go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest && \
    go install sigs.k8s.io/kustomize/kustomize/v5@latest && \
    go install github.com/joelanford/go-apidiff@latest

# Set working directory
WORKDIR /workspace

# Default command
CMD ["bash"]