# Implementation Plan - Stripe Webhook & K3s Hetzner Deployment

This plan outlines the steps to verify/complete the Stripe webhook implementation and deploy the **Booking Rush** services to a Hetzner VPS (8GB RAM, 4 vCPU) using **k3s**.

## Phase 1: Verify & Complete Stripe Integration

- [x] **Check existing Stripe Implementation**
    - [x] `StripeGateway` logic mostly implemented in `internal/gateway/stripe_gateway.go`.
    - [x] Webhook handler exists in `internal/handler/webhook_handler.go`.
- [ ] **Verify Webhook Logic**
    - [ ] `view_file` `webhook_handler.go` to ensure it handles `payment_intent.succeeded` and `payment_intent.payment_failed` correctly.
    - [ ] Ensure it triggers the Saga compensation or confirmation (likely via Kafka producer).
- [ ] **Integration Test (Local)**
    - [ ] Add a test case or manual verification steps for the webhook handler signature verification.

## Phase 2: K3s Deployment on Hetzner VPS

- [ ] **Write Deployment Scripts**
    - [ ] Create `scripts/deploy/install_k3s.sh` for the Hetzner VPS (install k3s, helm, kubectl).
    - [ ] Create `scripts/deploy/setup_registries.sh` (if using local registry or configuring docker hub secrets).
- [ ] **Create Helm Chart / Manifests**
    - [ ] Create `infra/k8s/booking-rush/Chart.yaml` (Umbrella chart or individual manifests).
    - [ ] **Infrastructure Components:**
        - [ ] PostgreSQL (StatefulSet + PVC, resource limits for 8GB RAM node).
        - [ ] Redis (Deployment).
        - [ ] Kafka/Redpanda (StatefulSet, single broker for this scale).
    - [ ] **Application Components:**
        - [ ] API Gateway (Deployment + Service).
        - [ ] Booking Service (Deployment, Autoscaling HPA).
        - [ ] Payment Service (Deployment, Secret for Stripe keys).
    - [ ] **Ingress:**
        - [ ] Traefik IngressRoute or Standard Ingress for `api.yourdomain.com`.
- [ ] **Secrets Management**
    - [ ] Template for `kubectl create secret generic app-secrets --from-env-file=.env.prod`.

## Phase 3: Documentation & Handover

- [ ] **Update Documentation**
    - [ ] Create `docs/deployment.md` with instructions for the user to SSH into Hetzner and run the scripts.
    - [ ] Document how to check logs: `kubectl logs -f -l app=booking-service`.
