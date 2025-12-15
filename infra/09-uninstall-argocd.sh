#!/bin/bash
set -e

# ============================================================
# Booking Rush - Uninstall ArgoCD
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
HOST="${HOST:-5.75.233.23}"
SSH_USER="${SSH_USER:-root}"
ARGOCD_NAMESPACE="argocd"
ARGOCD_VERSION="v2.13.2"

print_header() {
    echo -e "\n${BLUE}============================================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}============================================================${NC}\n"
}

print_success() { echo -e "${GREEN}✓ $1${NC}"; }
print_warning() { echo -e "${YELLOW}⚠ $1${NC}"; }
print_error() { echo -e "${RED}✗ $1${NC}"; }

ssh_cmd() {
    ssh -o StrictHostKeyChecking=no "$SSH_USER@$HOST" "export KUBECONFIG=/etc/rancher/k3s/k3s.yaml && $@"
}

# ============================================================
# Uninstall
# ============================================================

print_header "Uninstalling ArgoCD"
echo "Target: $SSH_USER@$HOST"
echo ""

read -p "Are you sure you want to uninstall ArgoCD? (y/N) " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    echo "Aborted."
    exit 1
fi

echo "Deleting ArgoCD applications..."
ssh_cmd "kubectl delete applications --all -n ${ARGOCD_NAMESPACE}" || true

echo "Deleting ArgoCD..."
ssh_cmd "kubectl delete -n ${ARGOCD_NAMESPACE} -f https://raw.githubusercontent.com/argoproj/argo-cd/${ARGOCD_VERSION}/manifests/install.yaml" || true

echo "Deleting namespace..."
ssh_cmd "kubectl delete namespace ${ARGOCD_NAMESPACE}" || true

print_success "ArgoCD uninstalled"
