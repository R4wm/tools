#!/bin/bash
# Install cloudflare-ddns as a self-running systemd service (Restart=always).
# No cron involved: the daemon loops on its own and updates Cloudflare only
# when your public IP changes.
set -e

SERVICE=cloudflare-ddns
BIN="$HOME/bin/$SERVICE"
CONF_DIR=/etc/$SERVICE
CONF="$CONF_DIR/config.env"
INTERVAL="${INTERVAL:-5m}"

echo "========================================"
echo "cloudflare-ddns deployment"
echo "========================================"

# 1. Ensure the binary is installed
if [ ! -x "$BIN" ]; then
    echo "[1/5] Binary not found at $BIN — building & installing..."
    ( cd "$(git -C "$(dirname "$0")" rev-parse --show-toplevel)" && make cloudflare-ddns install )
else
    echo "[1/5] Found binary: $BIN"
fi

# 2. Config file
echo "[2/5] Setting up config at $CONF ..."
sudo mkdir -p "$CONF_DIR"
if [ ! -f "$CONF" ]; then
    sudo cp "$(dirname "$0")/config.example.env" "$CONF"
    sudo chmod 600 "$CONF"
    echo "    Created $CONF from the example."
    echo "    >>> EDIT IT NOW with your token/domain, then re-run this script. <<<"
    echo "    sudo \$EDITOR $CONF"
    exit 0
fi
sudo chmod 600 "$CONF"
echo "    Config present."

# 3. systemd unit
echo "[3/5] Writing /etc/systemd/system/$SERVICE.service ..."
sudo tee /etc/systemd/system/$SERVICE.service > /dev/null <<EOF
[Unit]
Description=Cloudflare Dynamic DNS updater
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$USER
EnvironmentFile=$CONF
ExecStart=$BIN -interval $INTERVAL
Restart=always
RestartSec=30

[Install]
WantedBy=multi-user.target
EOF

# 4. Enable + start
echo "[4/5] Enabling and starting service..."
sudo systemctl daemon-reload
sudo systemctl enable "$SERVICE"
sudo systemctl restart "$SERVICE"

# 5. Status
echo "[5/5] Status:"
sleep 2
sudo systemctl status "$SERVICE" --no-pager -l || true
echo
echo "Follow logs with:  sudo journalctl -u $SERVICE -f"
