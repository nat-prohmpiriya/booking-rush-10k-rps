#!/bin/bash
set -e

# ============================================================
# Booking Rush Infrastructure Setup Script
#
# This script SSHs to remote server and installs:
# 1. k3s - Lightweight Kubernetes
# 2. Kubero - UI จัดการ apps (เหมือน Coolify)
# ============================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
if [ -f "$SCRIPT_DIR/.env" ]; then
    source "$SCRIPT_DIR/.env"
fi

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Config
HOST="${HOST:-}"
SSH_USER="${SSH_USER:-root}"
K3S_VERSION="${K3S_VERSION:-}"

print_header() {
    echo -e "\n${BLUE}============================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================================${NC}\n"
}

print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }

check_config() {
    if [ -z "$HOST" ]; then
        print_error "HOST is not set in .env"
        exit 1
    fi
    print_success "Target server: $SSH_USER@$HOST"
}

ssh_cmd() {
    ssh -o StrictHostKeyChecking=no "$SSH_USER@$HOST" "$@"
}

check_connection() {
    print_header "Checking SSH Connection"

    if ssh_cmd "echo 'Connected'" &>/dev/null; then
        print_success "SSH connection OK"
    else
        print_error "Cannot connect to $SSH_USER@$HOST"
        echo "Make sure:"
        echo "  1. Server is running"
        echo "  2. SSH key is configured"
        echo "  3. Run: ssh-copy-id $SSH_USER@$HOST"
        exit 1
    fi
}

check_remote_requirements() {
    print_header "Checking Remote Server Requirements"

    # Check OS
    OS_INFO=$(ssh_cmd "cat /etc/os-release 2>/dev/null | grep PRETTY_NAME | cut -d'\"' -f2")
    print_success "OS: $OS_INFO"

    # Check memory
    TOTAL_MEM=$(ssh_cmd "free -m | awk '/^Mem:/{print \$2}'")
    if [ "$TOTAL_MEM" -lt 2048 ]; then
        print_warning "Memory: ${TOTAL_MEM}MB (< 2GB recommended)"
    else
        print_success "Memory: ${TOTAL_MEM}MB"
    fi

    # Check disk
    DISK_SPACE=$(ssh_cmd "df -BG / | awk 'NR==2 {print \$4}' | tr -d 'G'")
    if [ "$DISK_SPACE" -lt 10 ]; then
        print_warning "Disk: ${DISK_SPACE}GB (< 10GB recommended)"
    else
        print_success "Disk: ${DISK_SPACE}GB available"
    fi
}

install_k3s() {
    print_header "Installing k3s on $HOST"

    # Check if already installed
    if ssh_cmd "command -v k3s" &>/dev/null; then
        print_warning "k3s is already installed"
        ssh_cmd "k3s --version"
        read -p "Reinstall? (y/N): " REINSTALL
        if [ "$REINSTALL" != "y" ] && [ "$REINSTALL" != "Y" ]; then
            return
        fi
    fi

    echo "Installing k3s..."
    if [ -n "$K3S_VERSION" ]; then
        ssh_cmd "curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION='$K3S_VERSION' sh -"
    else
        ssh_cmd "curl -sfL https://get.k3s.io | sh -"
    fi

    echo "Waiting for k3s to be ready..."
    sleep 10

    ssh_cmd "until k3s kubectl get nodes | grep -q Ready; do echo 'Waiting...'; sleep 5; done"

    print_success "k3s installed"
    ssh_cmd "k3s kubectl get nodes"
}

setup_kubeconfig() {
    print_header "Setting up kubeconfig"

    ssh_cmd "mkdir -p ~/.kube && cp /etc/rancher/k3s/k3s.yaml ~/.kube/config && chmod 600 ~/.kube/config"

    # Add KUBECONFIG to bashrc if not exists
    ssh_cmd "grep -q 'KUBECONFIG' ~/.bashrc || echo 'export KUBECONFIG=~/.kube/config' >> ~/.bashrc"

    print_success "kubeconfig configured"
}

install_kubero_cli() {
    print_header "Installing Kubero CLI"

    if ssh_cmd "command -v kubero" &>/dev/null; then
        print_warning "Kubero CLI already installed"
        ssh_cmd "kubero version" || true
        return
    fi

    ssh_cmd "curl -fsSL https://get.kubero.dev | bash"

    print_success "Kubero CLI installed"
}

install_kubero() {
    print_header "Installing Kubero"

    if ssh_cmd "k3s kubectl get namespace kubero" &>/dev/null; then
        print_warning "Kubero namespace already exists"
        read -p "Reinstall? (y/N): " REINSTALL
        if [ "$REINSTALL" != "y" ] && [ "$REINSTALL" != "Y" ]; then
            return
        fi
    fi

    echo ""
    echo "Kubero CLI will install:"
    echo "  - Ingress controller"
    echo "  - Metrics server"
    echo "  - Cert-manager (Let's Encrypt)"
    echo "  - Kubero UI"
    echo ""
    echo "Running interactive installer on remote server..."
    echo "You will answer questions in the terminal."
    echo ""

    # Run interactive installer
    ssh -t "$SSH_USER@$HOST" "kubero install"

    print_success "Kubero installed"
}

print_summary() {
    print_header "Installation Summary"

    echo "Server: $HOST"
    echo ""
    ssh_cmd "k3s --version"
    echo ""
    ssh_cmd "k3s kubectl get nodes"
    echo ""
    ssh_cmd "k3s kubectl get pods -A"
    echo ""

    print_success "Installation completed!"
    echo ""
    echo "Next steps:"
    echo "  1. SSH to server: ssh $SSH_USER@$HOST"
    echo "  2. Check Kubero: k3s kubectl get svc -n kubero"
    echo "  3. Access Kubero UI via the configured domain"
}

show_menu() {
    echo ""
    echo "Booking Rush Infrastructure Setup"
    echo "================================="
    echo "Target: $SSH_USER@$HOST"
    echo ""
    echo "1) Full Installation (k3s + Kubero)"
    echo "2) Install k3s only"
    echo "3) Install Kubero only (requires k3s)"
    echo "4) Check server status"
    echo "5) Show installation summary"
    echo "0) Exit"
    echo ""
    read -p "Select an option: " OPTION

    case $OPTION in
        1)
            check_connection
            check_remote_requirements
            install_k3s
            setup_kubeconfig
            install_kubero_cli
            install_kubero
            print_summary
            ;;
        2)
            check_connection
            check_remote_requirements
            install_k3s
            setup_kubeconfig
            ;;
        3)
            check_connection
            install_kubero_cli
            install_kubero
            ;;
        4)
            check_connection
            check_remote_requirements
            ;;
        5)
            check_connection
            print_summary
            ;;
        0)
            echo "Exiting..."
            exit 0
            ;;
        *)
            print_error "Invalid option"
            show_menu
            ;;
    esac
}

# Main
check_config

if [ "$1" == "--full" ]; then
    check_connection
    check_remote_requirements
    install_k3s
    setup_kubeconfig
    install_kubero_cli
    install_kubero
    print_summary
elif [ "$1" == "--k3s" ]; then
    check_connection
    check_remote_requirements
    install_k3s
    setup_kubeconfig
elif [ "$1" == "--kubero" ]; then
    check_connection
    install_kubero_cli
    install_kubero
elif [ "$1" == "--help" ] || [ "$1" == "-h" ]; then
    echo "Usage: $0 [OPTION]"
    echo ""
    echo "SSH to remote server and install k3s + Kubero"
    echo ""
    echo "Options:"
    echo "  --full     Full installation (k3s + Kubero)"
    echo "  --k3s      Install k3s only"
    echo "  --kubero   Install Kubero only"
    echo "  --help     Show this help"
    echo ""
    echo "Config (.env):"
    echo "  HOST       Remote server IP"
    echo "  SSH_USER   SSH username (default: root)"
else
    show_menu
fi
