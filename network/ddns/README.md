# cloudflare-ddns

Keeps a Cloudflare DNS **A record** pointed at your home connection's current
public IPv4 — so you can self-host behind a dynamic IP and stop paying for a VPS.

Source: [`src/cloudflare_ddns.go`](../../src/cloudflare_ddns.go) → builds to `bin/cloudflare-ddns`.

## How it finds your public IP (behind NAT)

A machine behind a router can't see its own public IP locally — it has to ask
something outside the NAT. This tool does it natively, in order:

1. **OpenDNS over DNS** — resolves the special name `myip.opendns.com` against
   OpenDNS's resolvers, which answer with your public IP (the native-Go version
   of `dig myip.opendns.com @resolver1.opendns.com`). Forced over IPv4.
2. **HTTPS echo fallbacks** — Cloudflare `cdn-cgi/trace`, then ipify/icanhazip,
   also pinned to IPv4.

Cloudflare's existing record is the source of truth, so the daemon **self-heals**:
if the record drifts or is deleted, the next tick fixes/recreates it. It only
calls the API when something actually changed.

## Build

```bash
cd ~/github/tools
make cloudflare-ddns      # -> bin/cloudflare-ddns
make install              # -> ~/bin/cloudflare-ddns
```

## Configure

1. Create a scoped API token at <https://dash.cloudflare.com/profile/api-tokens>
   using the **"Edit zone DNS"** template, limited to your zone.
2. Copy and fill the config:

   ```bash
   sudo mkdir -p /etc/cloudflare-ddns
   sudo cp network/ddns/config.example.env /etc/cloudflare-ddns/config.env
   sudo chmod 600 /etc/cloudflare-ddns/config.env
   sudo $EDITOR /etc/cloudflare-ddns/config.env
   ```

See [`config.example.env`](config.example.env) for every option.

## Run

```bash
# one-shot (test, or for a cron/timer if you prefer)
cloudflare-ddns -config /etc/cloudflare-ddns/config.env -once -v

# self-running daemon (no cron)
cloudflare-ddns -config /etc/cloudflare-ddns/config.env -interval 5m
```

## Run on its own (systemd, recommended)

```bash
./network/ddns/deploy.sh         # installs + starts cloudflare-ddns.service
sudo journalctl -u cloudflare-ddns -f
```

The service uses `Restart=always`, so it survives crashes and reboots — same
pattern as the other long-running tools in this repo. No cron job needed.

## Notes

- Updates **IPv4 A records** only. AAAA/IPv6 is not handled yet.
- For port-forwarding to a home box, keep `CF_PROXIED=false` (DNS-only) so
  traffic hits your IP directly instead of Cloudflare's HTTP proxy.
- Never commit your filled-in `config.env` — it holds your API token. The repo
  `.gitignore` excludes `*.env` config copies.
