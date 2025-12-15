#!/bin/bash
set -e

# ============================================================
# Booking Rush - Install ArgoCD
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

scp_file() {
    scp -o StrictHostKeyChecking=no "$1" "$SSH_USER@$HOST:$2"
}

# ============================================================
# Install Functions
# ============================================================

install_argocd() {
    print_header "Installing ArgoCD ${ARGOCD_VERSION}"

    echo "Creating namespace..."
    ssh_cmd "kubectl create namespace ${ARGOCD_NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -"

    echo "Installing ArgoCD..."
    ssh_cmd "kubectl apply -n ${ARGOCD_NAMESPACE} -f https://raw.githubusercontent.com/argoproj/argo-cd/${ARGOCD_VERSION}/manifests/install.yaml"

    print_success "ArgoCD installed"
}

wait_for_argocd() {
    print_header "Waiting for ArgoCD to be Ready"

    echo "Waiting for ArgoCD server..."
    ssh_cmd "kubectl rollout status deployment/argocd-server -n ${ARGOCD_NAMESPACE} --timeout=300s"

    echo "Waiting for ArgoCD repo server..."
    ssh_cmd "kubectl rollout status deployment/argocd-repo-server -n ${ARGOCD_NAMESPACE} --timeout=300s"

    echo "Waiting for ArgoCD application controller..."
    ssh_cmd "kubectl rollout status deployment/argocd-applicationset-controller -n ${ARGOCD_NAMESPACE} --timeout=300s"

    print_success "ArgoCD is ready"
}

configure_argocd() {
    print_header "Configuring ArgoCD"

    # Patch argocd-server to use insecure mode (for ingress without TLS termination)
    echo "Patching argocd-server for insecure mode..."
    ssh_cmd "kubectl patch deployment argocd-server -n ${ARGOCD_NAMESPACE} --type='json' -p='[{\"op\": \"add\", \"path\": \"/spec/template/spec/containers/0/args/-\", \"value\": \"--insecure\"}]'" || true

    # Wait for rollout
    ssh_cmd "kubectl rollout status deployment/argocd-server -n ${ARGOCD_NAMESPACE} --timeout=120s"

    print_success "ArgoCD configured"
}

deploy_ingress() {
    print_header "Deploying ArgoCD Ingress"

    scp_file "$SCRIPT_DIR/argocd/ingress.yaml" "/tmp/argocd-ingress.yaml"
    ssh_cmd "kubectl apply -f /tmp/argocd-ingress.yaml"

    print_success "ArgoCD Ingress deployed"
}

deploy_application() {
    print_header "Deploying ArgoCD Application"

    scp_file "$SCRIPT_DIR/argocd/application.yaml" "/tmp/argocd-application.yaml"
    ssh_cmd "kubectl apply -f /tmp/argocd-application.yaml"

    print_success "ArgoCD Application deployed"
}

get_admin_password() {
    print_header "ArgoCD Admin Credentials"

    echo "Username: admin"
    echo -n "Password: "
    ssh_cmd "kubectl -n ${ARGOCD_NAMESPACE} get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d"
    echo ""
    echo ""
    print_warning "Please change this password after first login!"
}

show_status() {
    print_header "ArgoCD Status"

    echo "Pods:"
    ssh_cmd "kubectl get pods -n ${ARGOCD_NAMESPACE}"
    echo ""
    echo "Services:"
    ssh_cmd "kubectl get svc -n ${ARGOCD_NAMESPACE}"
    echo ""
    echo "Ingress:"
    ssh_cmd "kubectl get ingress -n ${ARGOCD_NAMESPACE}" || true
    echo ""
    echo "Applications:"
    ssh_cmd "kubectl get applications -n ${ARGOCD_NAMESPACE}" || true
}

show_access_info() {
    print_header "Access Information"

    echo "ArgoCD UI: https://argocd.booking-rush.dackbox.com"
    echo ""
    echo "Or use port-forward locally:"
    echo "  ssh -L 8443:localhost:8443 ${SSH_USER}@${HOST} 'kubectl port-forward svc/argocd-server -n argocd 8443:443'"
    echo "  Then open: https://localhost:8443"
    echo ""
}

# ============================================================
# Main
# ============================================================

print_header "ArgoCD Installation for Booking Rush"
echo "Target: $SSH_USER@$HOST"
echo ""

case "${1:-}" in
    --install)
        install_argocd
        wait_for_argocd
        configure_argocd
        deploy_ingress
        get_admin_password
        show_access_info
        ;;
    --app)
        deploy_application
        ;;
    --password)
        get_admin_password
        ;;
    --status)
        show_status
        ;;
    --ingress)
        deploy_ingress
        ;;
    --all)
        install_argocd
        wait_for_argocd
        configure_argocd
        deploy_ingress
        deploy_application
        get_admin_password
        show_access_info
        show_status
        ;;
    *)
        echo "Usage: $0 [OPTION]"
        echo ""
        echo "Options:"
        echo "  --install    Install ArgoCD only"
        echo "  --app        Deploy booking-rush application"
        echo "  --password   Show admin password"
        echo "  --status     Show ArgoCD status"
        echo "  --ingress    Deploy ingress only"
        echo "  --all        Install everything (ArgoCD + App)"
        echo ""
        ;;
esac
