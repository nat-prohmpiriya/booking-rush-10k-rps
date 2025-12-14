#!/bin/bash
set -e

# ============================================================
# Booking Rush - Install Add-ons
#
# Bitnami Helm charts:
# - PostgreSQL (relational database)
# - Redis (cache & session)
# - MongoDB (document database)
# - Redpanda (Kafka-compatible message queue)
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
NAMESPACE="booking-rush"

# PostgreSQL
export PG_USERNAME="${PG_USERNAME:-postgres}"
export PG_PASSWORD="${PG_PASSWORD:?PG_PASSWORD is required. Set it in .env or environment}"
export PG_DATABASE="${PG_DATABASE:-booking_rush}"

# Redis
export REDIS_PASSWORD="${REDIS_PASSWORD:?REDIS_PASSWORD is required. Set it in .env or environment}"

# MongoDB
export MONGO_USERNAME="${MONGO_USERNAME:-booking_user}"
export MONGO_PASSWORD="${MONGO_PASSWORD:?MONGO_PASSWORD is required. Set it in .env or environment}"
export MONGO_DATABASE="${MONGO_DATABASE:-booking_rush}"

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
# Install Helm (if not exists)
# ============================================================
install_helm() {
    print_header "Installing Helm"

    if ssh_cmd "command -v helm" &>/dev/null; then
        print_warning "Helm already installed"
        ssh_cmd "helm version --short"
        return
    fi

    ssh_cmd "curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash"
    print_success "Helm installed"
}

# ============================================================
# Create Namespace
# ============================================================
create_namespace() {
    print_header "Creating namespace: $NAMESPACE"

    ssh_cmd "kubectl create namespace $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -"
    print_success "Namespace ready"
}

# ============================================================
# Install PostgreSQL (Helm)
# ============================================================
install_postgresql() {
    print_header "Installing PostgreSQL (Helm)"

    ssh_cmd "helm repo add bitnami https://charts.bitnami.com/bitnami || true"
    ssh_cmd "helm repo update"

    # Copy values file and substitute env vars
    envsubst < "$SCRIPT_DIR/helm-values/postgresql.yaml" > /tmp/postgresql-values.yaml
    scp -o StrictHostKeyChecking=no /tmp/postgresql-values.yaml "$SSH_USER@$HOST:/tmp/postgresql-values.yaml"

    if ssh_cmd "helm upgrade --install booking-rush-pg bitnami/postgresql \
        --namespace $NAMESPACE \
        -f /tmp/postgresql-values.yaml \
        --wait --timeout=300s"; then
        print_success "PostgreSQL installed"
    else
        print_error "PostgreSQL installation failed!"
        return 1
    fi
    echo ""
    echo "Connection info:"
    echo "  Host: booking-rush-pg-postgresql.$NAMESPACE.svc.cluster.local"
    echo "  Port: 5432"
    echo "  Database: booking_rush"
    echo "  Username: postgres"
    echo "  Password: $PG_PASSWORD"
}

# ============================================================
# Install Redis (Helm)
# ============================================================
install_redis() {
    print_header "Installing Redis (Helm)"

    ssh_cmd "helm repo add bitnami https://charts.bitnami.com/bitnami || true"
    ssh_cmd "helm repo update"

    # Copy values file and substitute env vars
    envsubst < "$SCRIPT_DIR/helm-values/redis.yaml" > /tmp/redis-values.yaml
    scp -o StrictHostKeyChecking=no /tmp/redis-values.yaml "$SSH_USER@$HOST:/tmp/redis-values.yaml"

    if ssh_cmd "helm upgrade --install booking-rush-redis bitnami/redis \
        --namespace $NAMESPACE \
        -f /tmp/redis-values.yaml \
        --wait --timeout=300s"; then
        print_success "Redis installed"
    else
        print_error "Redis installation failed!"
        return 1
    fi
    echo ""
    echo "Connection info:"
    echo "  Host: booking-rush-redis-master.$NAMESPACE.svc.cluster.local"
    echo "  Port: 6379"
    echo "  Password: $REDIS_PASSWORD"
}

# ============================================================
# Install MongoDB (Helm)
# ============================================================
install_mongodb() {
    print_header "Installing MongoDB (Helm)"

    ssh_cmd "helm repo add bitnami https://charts.bitnami.com/bitnami || true"
    ssh_cmd "helm repo update"

    # Copy values file and substitute env vars
    envsubst < "$SCRIPT_DIR/helm-values/mongodb.yaml" > /tmp/mongodb-values.yaml
    scp -o StrictHostKeyChecking=no /tmp/mongodb-values.yaml "$SSH_USER@$HOST:/tmp/mongodb-values.yaml"

    if ssh_cmd "helm upgrade --install booking-rush-mongodb bitnami/mongodb \
        --namespace $NAMESPACE \
        -f /tmp/mongodb-values.yaml \
        --wait --timeout=300s"; then
        print_success "MongoDB installed"
    else
        print_error "MongoDB installation failed!"
        return 1
    fi
    echo ""
    echo "Connection info:"
    echo "  Host: booking-rush-mongodb.$NAMESPACE.svc.cluster.local"
    echo "  Port: 27017"
    echo "  Database: booking_rush"
    echo "  Username: booking_user (or root)"
    echo "  Password: $MONGO_PASSWORD"
}

# ============================================================
# Install Redpanda (Kafka-compatible)
# ============================================================
install_redpanda() {
    print_header "Installing Redpanda"

    ssh_cmd "helm repo add redpanda https://charts.redpanda.com || true"
    ssh_cmd "helm repo update"

    # Copy values file to server
    scp -o StrictHostKeyChecking=no "$SCRIPT_DIR/helm-values/redpanda.yaml" "$SSH_USER@$HOST:/tmp/redpanda-values.yaml"

    # Install with values file
    if ssh_cmd "helm upgrade --install booking-rush-redpanda redpanda/redpanda \
        --namespace $NAMESPACE \
        -f /tmp/redpanda-values.yaml \
        --wait --timeout=600s"; then
        print_success "Redpanda installed"
    else
        print_error "Redpanda installation failed!"
        return 1
    fi
    echo ""
    echo "Connection info:"
    echo "  Brokers: booking-rush-redpanda.$NAMESPACE.svc.cluster.local:9092"
}

# ============================================================
# Install Node Exporter (Monitoring)
# ============================================================
install_node_exporter() {
    print_header "Installing Node Exporter (Helm)"

    ssh_cmd "helm repo add prometheus-community https://prometheus-community.github.io/helm-charts || true"
    ssh_cmd "helm repo update"

    # Copy values file to server
    scp -o StrictHostKeyChecking=no "$SCRIPT_DIR/helm-values/node-exporter.yaml" "$SSH_USER@$HOST:/tmp/node-exporter-values.yaml"

    if ssh_cmd "helm upgrade --install booking-rush-node-exporter prometheus-community/prometheus-node-exporter \
        --namespace $NAMESPACE \
        -f /tmp/node-exporter-values.yaml \
        --wait --timeout=120s"; then
        print_success "Node Exporter installed"
    else
        print_error "Node Exporter installation failed!"
        return 1
    fi
    echo ""
    echo "Connection info:"
    echo "  Metrics: booking-rush-node-exporter.$NAMESPACE.svc.cluster.local:9100/metrics"
}

# ============================================================
# Print Summary
# ============================================================
print_summary() {
    print_header "Installation Summary"

    echo "Namespace: $NAMESPACE"
    echo ""

    ssh_cmd "kubectl get pods -n $NAMESPACE"

    echo ""
    echo -e "${GREEN}=== Connection Info ===${NC}"
    echo ""
    echo "PostgreSQL:"
    echo "  Host: booking-rush-pg-postgresql.$NAMESPACE.svc.cluster.local"
    echo "  Port: 5432"
    echo "  Database: booking_rush"
    echo "  Username: postgres"
    echo "  Password: $PG_PASSWORD"
    echo ""
    echo "Redis:"
    echo "  Host: booking-rush-redis-master.$NAMESPACE.svc.cluster.local"
    echo "  Port: 6379"
    echo "  Password: $REDIS_PASSWORD"
    echo ""
    echo "MongoDB:"
    echo "  Host: booking-rush-mongodb.$NAMESPACE.svc.cluster.local"
    echo "  Port: 27017"
    echo "  Database: booking_rush"
    echo "  Username: booking_user (or root)"
    echo "  Password: $MONGO_PASSWORD"
    echo ""
    echo "Redpanda (Kafka):"
    echo "  Brokers: booking-rush-redpanda.$NAMESPACE.svc.cluster.local:9092"
    echo ""

    print_success "All add-ons installed!"
}

# ============================================================
# Menu
# ============================================================
show_menu() {
    echo ""
    echo "Booking Rush - Install Add-ons"
    echo "==============================="
    echo "Target: $SSH_USER@$HOST"
    echo "Namespace: $NAMESPACE"
    echo ""
    echo "1) Install ALL (PostgreSQL + Redis + MongoDB + Redpanda + Node Exporter)"
    echo "2) Install PostgreSQL (Helm)"
    echo "3) Install Redis (Helm)"
    echo "4) Install MongoDB (Helm)"
    echo "5) Install Redpanda (Helm)"
    echo "6) Install Node Exporter (Monitoring)"
    echo "7) Show status"
    echo "0) Exit"
    echo ""
    read -p "Select an option: " OPTION

    case $OPTION in
        1)
            install_helm
            create_namespace
            install_postgresql
            install_redis
            install_mongodb
            install_redpanda
            install_node_exporter
            print_summary
            ;;
        2)
            install_helm
            create_namespace
            install_postgresql
            ;;
        3)
            install_helm
            create_namespace
            install_redis
            ;;
        4)
            install_helm
            create_namespace
            install_mongodb
            ;;
        5)
            install_helm
            create_namespace
            install_redpanda
            ;;
        6)
            install_helm
            create_namespace
            install_node_exporter
            ;;
        7)
            ssh_cmd "kubectl get pods -n $NAMESPACE"
            ssh_cmd "kubectl get svc -n $NAMESPACE"
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

# ============================================================
# Main
# ============================================================
if [ "$1" == "--all" ]; then
    install_helm
    create_namespace
    install_postgresql
    install_redis
    install_mongodb
    install_redpanda
    install_node_exporter
    print_summary
elif [ "$1" == "--pg" ]; then
    install_helm
    create_namespace
    install_postgresql
elif [ "$1" == "--redis" ]; then
    install_helm
    create_namespace
    install_redis
elif [ "$1" == "--mongodb" ]; then
    install_helm
    create_namespace
    install_mongodb
elif [ "$1" == "--redpanda" ]; then
    install_helm
    create_namespace
    install_redpanda
elif [ "$1" == "--node-exporter" ]; then
    install_helm
    create_namespace
    install_node_exporter
elif [ "$1" == "--status" ]; then
    ssh_cmd "kubectl get pods -n $NAMESPACE"
    ssh_cmd "kubectl get svc -n $NAMESPACE"
elif [ "$1" == "--help" ] || [ "$1" == "-h" ]; then
    echo "Usage: $0 [OPTION]"
    echo ""
    echo "Install production-ready add-ons on k3s"
    echo ""
    echo "Options:"
    echo "  --all            Install all (PostgreSQL + Redis + MongoDB + Redpanda + Node Exporter)"
    echo "  --pg             Install PostgreSQL (Helm)"
    echo "  --redis          Install Redis (Helm)"
    echo "  --mongodb        Install MongoDB (Helm)"
    echo "  --redpanda       Install Redpanda (Helm)"
    echo "  --node-exporter  Install Node Exporter (Monitoring)"
    echo "  --status         Show pods status"
    echo "  --help           Show this help"
else
    show_menu
fi
