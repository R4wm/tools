# Network Utilities - Upload Speed Testing

Network testing and monitoring tools for measuring upload bandwidth.

## uptest - Upload Speed Testing Utility

A comprehensive tool for testing upload bandwidth to remote servers with remote configuration management.

### Features

- **Machine Fingerprint ID**: Each client has unique deterministic ID based on IP + hostname + OS
- **Remote Configuration**: Fetch per-client configuration from central server
- **HTTP Upload Testing**: 1GB in-memory data generation (no disk I/O)
- **Multiple Output Formats**: JSON, inline text
- **Real-time Progress**: Optional progress bar during upload
- **Redis Integration**: Time-series data storage (coming soon)
- **GitHub Integration**: Auto-upload results to repository (coming soon)
- **Daemon Mode**: Run tests on schedule
- **Remote Uninstall**: Decommission clients remotely
- **Docker Support**: Easy deployment (coming soon)

### Quick Start

```bash
# Build
cd /home/r4wm/github/tools
make uptest

# Install
make install

# Run a test
uptest --progress

# Run with custom server
uptest --server test.example.com --progress

# Daemon mode (run every hour)
uptest --daemon --interval 1h
```

### Configuration

uptest can be configured via:
1. Command-line flags (highest priority)
2. Remote configuration from central.prsmusa.com
3. Local config file (fallback)

See `config.example.yaml` for configuration options.

### Remote Configuration

The tool automatically fetches configuration from `https://central.prsmusa.com/uptest/config?client_id={id}` where the client ID is generated deterministically from your machine's fingerprint (IP + hostname + OS).

Benefits:
- Change test parameters remotely without SSH
- Enable/disable features per-client
- Remote uninstall capability
- Emergency kill switch
- Client autonomy (works offline with local config)

### Client ID

Your client ID is automatically generated from:
- Current outbound IP address
- Hostname
- Operating system

Example: `3e7b4f8a1c2d5e6f` (SHA256 hash of IP|hostname|OS)

This ensures:
- Same machine always gets same ID
- Stable across reboots (unless IP/hostname/OS changes)
- No manual configuration needed

### Output Formats

**JSON (default):**
```bash
uptest -o json
```

Output includes: test_id, timestamp, upload speed (Mbps and bytes/sec), duration, latency, status, metadata.

**Inline:**
```bash
uptest -o inline
```

Human-readable format with key metrics.

### Examples

```bash
# Silent mode (for cron)
uptest --silent -o json

# Verbose logging
uptest -v --progress

# Custom data size (100MB)
uptest --size 104857600 --progress

# Short timeout
uptest --timeout 2m --progress
```

### Server Setup

The Go server in [`server/`](server/) provides the upload receiver plus a fixed-duration
download stream. See [`server/README.md`](server/README.md) for build, systemd, and Nginx
deployment instructions.

### Docker Deployment

Docker support with host networking mode for zero performance impact (coming soon).

```bash
# Build image
make docker-build

# Run with docker-compose
make docker-run

# View logs
make docker-logs

# Stop
make docker-stop
```

### Development Status

- ✅ Core HTTP upload testing
- ✅ Machine fingerprint-based client ID
- ✅ Remote configuration with metadata headers
- ✅ Progress bar and multiple output formats
- ✅ Daemon mode
- ✅ Remote uninstall
- 🚧 Redis integration (in progress)
- 🚧 GitHub integration (in progress)
- 🚧 Docker deployment (in progress)
- 🚧 Server-side components (in progress)

### Architecture

```
uptest (client)
├── Generates machine fingerprint ID
├── Fetches remote config from central server
├── Runs HTTP upload test
├── Saves results to Redis (optional)
└── Uploads to GitHub repo (optional)

central.prsmusa.com (server)
├── /uptest/config - Config API (per-client)
├── /upload_test - Upload endpoint
├── /download_test - Fixed-duration download endpoint
└── Client registry with metadata
```

### License

Same as parent repository (MIT).

### Contributing

This tool is part of a larger personal utilities collection. See the main repository README for details.
