# Speed test server

`speedtest-server` provides a small HTTP service for network measurements.

- `GET /download_test` streams arbitrary bytes for exactly the configured duration (10 seconds by default). It ignores query parameters and intentionally does not send `Content-Length`.
- `POST /upload_test` reads and discards a request body, then returns the observed upload rate as JSON.
- `GET` or `HEAD /ping` returns `pong`.
- `GET` or `HEAD /health` returns `OK`.

The download handler reuses one random 64 KiB buffer. It does not read a file or generate random bytes in the hot path, so CPU and disk I/O do not become the limiting factor. It caps concurrent streams (20 by default); excess requests receive `503`.

## Build and run

From the repository root:

```bash
make speedtest-server
./bin/speedtest-server -port 8080 -download-duration 10s -max-downloads 20
```

For a quick local check:

```bash
curl --http1.1 --no-buffer -o /dev/null -w '%{size_download} bytes in %{time_total}s\n' http://127.0.0.1:8080/download_test
curl -fsS http://127.0.0.1:8080/health
```

## Server deployment

The supplied templates assume a dedicated, unprivileged `speedtest` account and a binary at `/usr/local/bin/speedtest-server`. The service binds to `127.0.0.1` by default; use `-listen-address 0.0.0.0` only if you deliberately need direct public access:

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin speedtest
sudo install -m 0755 bin/speedtest-server /usr/local/bin/speedtest-server
sudo install -m 0644 network/server/speedtest.service /etc/systemd/system/speedtest.service
sudo systemctl daemon-reload
sudo systemctl enable --now speedtest
```

Copy the contents of [`nginx-speedtest.conf`](nginx-speedtest.conf) into the appropriate HTTPS `server` block, then validate and reload Nginx:

```bash
sudo nginx -t && sudo systemctl reload nginx
```

`proxy_buffering off` and `gzip off` on `/download_test` are required: buffering changes the timing and compression changes the byte count. Keep the Go listener private (for example, use Nginx as the only public listener or bind it to `127.0.0.1`).

For a public endpoint, add Nginx rate/concurrency limits appropriate for the server's egress capacity. The application limit is a final safeguard, not a substitute for perimeter protection.
