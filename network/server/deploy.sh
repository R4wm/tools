#!/bin/bash
set -e

echo "========================================"
echo "Speed Test Server Deployment Script"
echo "========================================"
echo ""

# Check if http_receiver.go exists
if [ ! -f "http_receiver.go" ]; then
    echo "ERROR: http_receiver.go not found in current directory"
    echo "Please run this script from the directory containing http_receiver.go"
    exit 1
fi

# Build the server
echo "[1/6] Building Go server..."
go build http_receiver.go
echo "✓ Built http_receiver"
echo ""

# Create systemd service
echo "[2/6] Creating systemd service..."
sudo tee /etc/systemd/system/speedtest.service > /dev/null <<EOF
[Unit]
Description=Speed Test HTTP Server
After=network.target

[Service]
Type=simple
User=$USER
WorkingDirectory=$HOME
ExecStart=$HOME/http_receiver -port 8080
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF
echo "✓ Created /etc/systemd/system/speedtest.service"
echo ""

# Create nginx config snippet
echo "[3/6] Creating nginx configuration..."
sudo tee /etc/nginx/conf.d/speedtest.conf > /dev/null <<'EOF'
# Speed test endpoints configuration
# Include this in your server {} block or it will apply to default server

location ~ ^/(upload_test|download_test|ping|health)$ {
    client_max_body_size 2G;
    client_body_buffer_size 128k;
    proxy_pass http://127.0.0.1:8080;
    proxy_request_buffering off;
    proxy_http_version 1.1;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_connect_timeout 300s;
    proxy_send_timeout 300s;
    proxy_read_timeout 300s;
}
EOF
echo "✓ Created /etc/nginx/conf.d/speedtest.conf"
echo ""

# Test nginx configuration
echo "[4/6] Testing nginx configuration..."
sudo nginx -t
echo "✓ Nginx configuration is valid"
echo ""

# Enable and start systemd service
echo "[5/6] Enabling and starting speedtest service..."
sudo systemctl daemon-reload
sudo systemctl enable speedtest
sudo systemctl restart speedtest
sleep 2
echo "✓ Service started"
echo ""

# Reload nginx
echo "[6/6] Reloading nginx..."
sudo systemctl reload nginx
echo "✓ Nginx reloaded"
echo ""

# Check service status
echo "========================================"
echo "Deployment Complete!"
echo "========================================"
echo ""
echo "Service status:"
sudo systemctl status speedtest --no-pager -l
echo ""
echo "Testing endpoints locally..."
echo ""

# Test endpoints
echo "Testing /ping:"
curl -s http://localhost:8080/ping || echo "Failed"
echo ""

echo "Testing /health:"
curl -s http://localhost:8080/health || echo "Failed"
echo ""

echo "Testing /upload_test:"
echo "test data" | curl -s -X POST -d @- http://localhost:8080/upload_test || echo "Failed"
echo ""

echo "========================================"
echo "Server is ready!"
echo "========================================"
echo ""
echo "Test from client with:"
echo "  uptest --server prsmusa.com --protocol http --test-upload --upload-size 1048576 --remote-config=false --progress -o inline"
echo ""
echo "View logs with:"
echo "  sudo journalctl -u speedtest -f"
echo ""
