#!/bin/bash
source "$(dirname "$0")/.env"
ssh "$SSH_USER@$HOST"


scp root@5.75.233.23:/etc/rancher/k3s/k3s.yaml ~/.kube/config-booking && sed -i '' 's/127.0.0.1/5.75.233.23/g' ~/.kube/config-booking

kubectl --kubeconfig=~/.kube/config-booking get nodes