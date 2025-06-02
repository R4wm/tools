d#!/bin/bash

# Create a systemd user service for clipmenud
SERVICE_FILE="$HOME/.config/systemd/user/clipmenud.service"

mkdir -p "$(dirname "$SERVICE_FILE")"

cat >"$SERVICE_FILE" <<EOF
[Unit]
Description=clipmenud clipboard manager daemon

[Service]
ExecStart=$(command -v clipmenud)
Restart=on-failure

[Install]
WantedBy=default.target
EOF

echo "Enabling and starting clipmenud systemd user service..."
systemctl --user daemon-reload
systemctl --user enable --now clipmenud.service

echo "Done. Service status:"
systemctl --user status clipmenud.service
