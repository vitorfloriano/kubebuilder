#!/bin/bash

# Kubebuilder Development Environment Setup Script
# This script checks for and installs all necessary tools for kubebuilder development

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Get OS and architecture
get_os_arch() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        arm64) ARCH="arm64" ;;
    esac
    
    export OS ARCH
}

# Install kubectl
install_kubectl() {
    if command_exists kubectl; then
        log_success "kubectl already installed: $(kubectl version --client --short 2>/dev/null || kubectl version --client)"
        return
    fi
    
    log_info "Installing kubectl..."
    KUBECTL_VERSION=$(curl -L -s https://dl.k8s.io/release/stable.txt)
    curl -LO "https://dl.k8s.io/release/$KUBECTL_VERSION/bin/${OS}/${ARCH}/kubectl"
    chmod +x kubectl
    sudo mv kubectl /usr/local/bin/
    log_success "kubectl installed: version $KUBECTL_VERSION"
}

# Install kind
install_kind() {
    if command_exists kind; then
        log_success "kind already installed: $(kind version)"
        return
    fi
    
    log_info "Installing kind..."
    local KIND_VERSION="v0.29.0"
    curl -Lo ./kind "https://kind.sigs.k8s.io/dl/${KIND_VERSION}/kind-${OS}-${ARCH}"
    chmod +x ./kind
    sudo mv ./kind /usr/local/bin/kind
    log_success "kind installed: version $KIND_VERSION"
}

# Install helm
install_helm() {
    if command_exists helm; then
        log_success "helm already installed: $(helm version --short)"
        return
    fi
    
    log_info "Installing helm..."
    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    log_success "helm installed"
}

# Install kustomize (if not already available)
install_kustomize() {
    if command_exists kustomize; then
        log_success "kustomize already installed: $(kustomize version --short)"
        return
    fi
    
    log_info "Installing kustomize..."
    go install sigs.k8s.io/kustomize/kustomize/v5@latest
    log_success "kustomize installed"
}

# Install controller-gen
install_controller_gen() {
    if command_exists controller-gen; then
        log_success "controller-gen already installed"
        return
    fi
    
    log_info "Installing controller-gen..."
    go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest
    log_success "controller-gen installed"
}

# Install setup-envtest
install_setup_envtest() {
    if command_exists setup-envtest; then
        log_success "setup-envtest already installed"
        return
    fi
    
    log_info "Installing setup-envtest..."
    go install sigs.k8s.io/controller-runtime/tools/setup-envtest@release-0.20
    log_success "setup-envtest installed"
}

# Install golangci-lint (if not already installed)
install_golangci_lint() {
    if command_exists golangci-lint; then
        log_success "golangci-lint already installed: $(golangci-lint version)"
        return
    fi
    
    log_info "Installing golangci-lint..."
    go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
    log_success "golangci-lint installed"
}

# Install ginkgo CLI
install_ginkgo() {
    if command_exists ginkgo; then
        log_success "ginkgo already installed"
        return
    fi
    
    log_info "Installing ginkgo..."
    go install github.com/onsi/ginkgo/v2/ginkgo@latest
    log_success "ginkgo installed"
}

# Install go-apidiff
install_go_apidiff() {
    if command_exists go-apidiff; then
        log_success "go-apidiff already installed"
        return
    fi
    
    log_info "Installing go-apidiff..."
    go install github.com/joelanford/go-apidiff@v0.6.1
    log_success "go-apidiff installed"
}

# Install yamllint via pip if available
install_yamllint() {
    if command_exists yamllint; then
        log_success "yamllint already installed"
        return
    fi
    
    if command_exists pip3; then
        log_info "Installing yamllint..."
        pip3 install --user yamllint
        log_success "yamllint installed"
    else
        log_warning "pip3 not available, yamllint will be run via Docker in Makefile"
    fi
}

# Setup kind network (if Docker is available)
setup_kind_network() {
    if ! command_exists docker; then
        log_warning "Docker not available, skipping kind network setup"
        return
    fi
    
    log_info "Setting up kind network..."
    if ! docker network ls | grep -q "kind"; then
        docker network create -d=bridge --subnet=172.19.0.0/24 kind || {
            log_warning "Failed to create kind network (may already exist)"
        }
    else
        log_success "kind network already exists"
    fi
}

# Verify Go environment
verify_go_environment() {
    log_info "Verifying Go environment..."
    
    if ! command_exists go; then
        log_error "Go is not installed!"
        exit 1
    fi
    
    GO_VERSION=$(go version | cut -d' ' -f3)
    log_success "Go installed: $GO_VERSION"
    
    # Check Go version compatibility (should be 1.24+)
    GO_MAJOR=$(echo $GO_VERSION | sed 's/go//' | cut -d'.' -f1)
    GO_MINOR=$(echo $GO_VERSION | sed 's/go//' | cut -d'.' -f2)
    
    if [[ $GO_MAJOR -eq 1 && $GO_MINOR -lt 24 ]]; then
        log_warning "Go version $GO_VERSION detected. Kubebuilder requires Go 1.24+."
    fi
    
    log_info "GOPATH: $(go env GOPATH)"
    log_info "GOROOT: $(go env GOROOT)"
}

# Setup envtest if possible
setup_envtest() {
    if command_exists setup-envtest; then
        log_info "Setting up envtest tools..."
        setup-envtest use 1.33.0 --bin-dir /tmp/kubebuilder-envtest || {
            log_warning "Failed to setup envtest tools (will be set up on first test run)"
        }
    fi
}

# Display summary
display_summary() {
    log_info "=== Installation Summary ==="
    
    local tools=(
        "go:Go compiler"
        "kubectl:Kubernetes CLI"
        "kind:Kubernetes in Docker"
        "helm:Helm package manager"
        "kustomize:Kubernetes configuration management"
        "controller-gen:Kubernetes controller code generator"
        "setup-envtest:Controller runtime test setup"
        "golangci-lint:Go linter"
        "ginkgo:BDD testing framework"
        "go-apidiff:API compatibility checker"
        "docker:Container runtime"
    )
    
    for tool_info in "${tools[@]}"; do
        IFS=':' read -r tool desc <<< "$tool_info"
        if command_exists "$tool"; then
            log_success "$desc: ✓ installed"
        else
            log_warning "$desc: ✗ not available"
        fi
    done
    
    echo
    log_info "=== Environment Variables ==="
    echo "GOPATH: $(go env GOPATH)"
    echo "GOROOT: $(go env GOROOT)"
    echo "PATH: $PATH"
    
    echo
    log_info "=== Quick Start ==="
    echo "1. Run 'make help' to see available commands"
    echo "2. Run 'make test' to run unit tests"
    echo "3. Run 'make build' to build kubebuilder"
    echo "4. Run 'make test-e2e-local' to run e2e tests with kind"
}

# Main installation flow
main() {
    log_info "Starting kubebuilder development environment setup..."
    
    get_os_arch
    log_info "Detected OS: $OS, Architecture: $ARCH"
    
    verify_go_environment
    
    # Install core tools
    install_kubectl
    install_kind
    install_helm
    install_kustomize
    install_controller_gen
    install_setup_envtest
    install_golangci_lint
    install_ginkgo
    install_go_apidiff
    install_yamllint
    
    # Setup environment
    setup_kind_network
    setup_envtest
    
    # Final verification
    display_summary
    
    log_success "Kubebuilder development environment setup complete!"
    log_info "You may need to restart your shell or run 'source ~/.bashrc' for PATH changes to take effect."
}

# Run main function
main "$@"