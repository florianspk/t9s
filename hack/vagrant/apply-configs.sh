#!/usr/bin/env bash
# Generates machine configs and bootstraps the cluster.
# Run after `vagrant up` once both VMs are in Talos maintenance mode.
set -euo pipefail

TALOS_VERSION="${1:-v1.7.0}"
CP_IP="192.168.56.10"
WORKER_IP="192.168.56.11"
CFG="talosconfig"

echo "Generating machine configs for Talos $TALOS_VERSION…"
talosctl gen config talos-vagrant "https://${CP_IP}:6443" \
  --output-dir . \
  --talos-version "$TALOS_VERSION" \
  --with-examples=false \
  --with-docs=false

echo "Applying controlplane config…"
talosctl apply-config \
  --talosconfig "$CFG" \
  --nodes "$CP_IP" \
  --file controlplane.yaml \
  --insecure

echo "Applying worker config…"
talosctl apply-config \
  --talosconfig "$CFG" \
  --nodes "$WORKER_IP" \
  --file worker.yaml \
  --insecure

echo "Waiting for controlplane API (this takes ~2 min)…"
talosctl --talosconfig "$CFG" -n "$CP_IP" health \
  --wait-timeout 5m --server=false 2>/dev/null || true

echo "Bootstrapping etcd…"
talosctl --talosconfig "$CFG" -n "$CP_IP" bootstrap

echo ""
echo "Cluster ready. Use:"
echo "  ./../../t9s --talosconfig $PWD/$CFG"
echo ""
echo "Upgrade test (v1.7 → v1.8):"
echo "  Press U on the controlplane node in t9s"
echo "  Image: ghcr.io/siderolabs/installer:v1.8.0"
