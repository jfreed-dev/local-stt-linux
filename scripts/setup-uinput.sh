#!/usr/bin/env bash
# Setup script for local-stt-linux
# Adds current user to the input group and enables ydotoold

set -euo pipefail

USER="${SUDO_USER:-$USER}"

echo "Adding $USER to input group..."
usermod -aG input "$USER"

echo "Enabling ydotoold service..."
if systemctl list-unit-files ydotoold.service &>/dev/null; then
    systemctl enable --now ydotoold.service
    echo "ydotoold enabled and started."
else
    echo "WARNING: ydotoold.service not found. Install ydotool first:"
    echo "  sudo apt install ydotool  # or your package manager"
fi

echo ""
echo "Done. Log out and back in for group changes to take effect."
echo "Verify with: groups $USER"
