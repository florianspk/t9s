#!/usr/bin/env bash
set -euo pipefail

TALOS_VERSION="${1:-v1.7.0}"
ISO="talos-amd64.iso"

if [[ -f "$ISO" ]]; then
  echo "$ISO already present, skipping download."
  exit 0
fi

echo "Downloading Talos $TALOS_VERSION metal ISO…"
curl -L --progress-bar \
  "https://github.com/siderolabs/talos/releases/download/${TALOS_VERSION}/metal-amd64.iso" \
  -o "$ISO"
echo "Done: $ISO"
