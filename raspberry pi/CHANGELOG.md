# Changelog

## 2026-07-26

- Added `secure-dmz-firewall.sh` for the Raspberry Pi DMZ host.
- Restricted Internet access to TCP 22, 80, and 443.
- Kept NFS, Samba, and other services accessible only from trusted LANs.
- Added Docker `DOCKER-USER` filtering and a persistent systemd service so
  Docker-published ports cannot bypass the firewall.
