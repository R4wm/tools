#!/usr/bin/env bash
# Secure a Raspberry Pi used as a DMZ web host.
#
# Internet: TCP 22 (SSH), 80 (HTTP), and 443 (HTTPS) only.
# Internal networks retain access to all host services, including NFS/Samba.
# Docker-published ports need extra filtering because Docker bypasses UFW's
# normal INPUT rules; this script blocks all Docker forwarding from the WAN
# except TCP 80 and 443.
#
# Run as root. Override these only if this host's network changes:
#   LAN_V4_CIDR=10.0.0.0/24 LAN_V6_CIDR=fdac:3746:2b2e:a8e9::/64 \
#     WAN_IF=eth0 sudo ./secure-dmz-firewall.sh

set -euo pipefail

LAN_V4_CIDR="${LAN_V4_CIDR:-10.0.0.0/24}"
LAN_V6_CIDR="${LAN_V6_CIDR:-fdac:3746:2b2e:a8e9::/64}"
WAN_IF="${WAN_IF:-eth0}"
INSTALL_PATH=/usr/local/sbin/secure-dmz-firewall
UNIT_PATH=/etc/systemd/system/docker-user-firewall.service
CHAIN=DMZ_ONLY_PUBLIC

if [[ ${EUID} -ne 0 ]]; then
  echo "Run with sudo." >&2
  exit 1
fi

if [[ ${1:-} == --docker-only ]]; then
  for ipt in iptables ip6tables; do
    command -v "$ipt" >/dev/null || continue
    "$ipt" -N "$CHAIN" 2>/dev/null || true
    "$ipt" -F "$CHAIN"
    "$ipt" -C DOCKER-USER -j "$CHAIN" 2>/dev/null || "$ipt" -I DOCKER-USER 1 -j "$CHAIN"
    "$ipt" -A "$CHAIN" -m conntrack --ctstate RELATED,ESTABLISHED -j RETURN
    if [[ $ipt == iptables ]]; then
      "$ipt" -A "$CHAIN" -i "$WAN_IF" -s "$LAN_V4_CIDR" -j RETURN
    else
      "$ipt" -A "$CHAIN" -i "$WAN_IF" -s "$LAN_V6_CIDR" -j RETURN
    fi
    "$ipt" -A "$CHAIN" -i "$WAN_IF" -p tcp --dport 80 -j RETURN
    "$ipt" -A "$CHAIN" -i "$WAN_IF" -p tcp --dport 443 -j RETURN
    "$ipt" -A "$CHAIN" -i "$WAN_IF" -j DROP
    "$ipt" -A "$CHAIN" -j RETURN
  done
  exit 0
fi

if [[ $# -ne 0 ]]; then
  echo "Usage: $0 [--docker-only]" >&2
  exit 2
fi

command -v ufw >/dev/null || { echo "ufw is required." >&2; exit 1; }
ip link show "$WAN_IF" >/dev/null || { echo "Interface $WAN_IF does not exist." >&2; exit 1; }

# This is intentionally declarative: remove any previous UFW allowances, then
# install only the DMZ policy described above. Existing SSH sessions survive.
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw default deny routed
ufw logging low

ufw allow 22/tcp comment 'SSH from Internet'
ufw allow 80/tcp comment 'HTTP from Internet'
ufw allow 443/tcp comment 'HTTPS from Internet'
ufw allow from "$LAN_V4_CIDR" comment 'trusted IPv4 LAN'
ufw allow from "$LAN_V6_CIDR" comment 'trusted IPv6 LAN'
ufw --force enable

install -m 0755 "$0" "$INSTALL_PATH"
cat >"$UNIT_PATH" <<UNIT
[Unit]
Description=Restrict Docker-published ports on the DMZ host
After=docker.service network-online.target
Wants=network-online.target
Requires=docker.service

[Service]
Type=oneshot
ExecStart=$INSTALL_PATH --docker-only
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable --now docker-user-firewall.service
"$INSTALL_PATH" --docker-only

echo "Firewall active: public TCP ports 22, 80, and 443 only."
echo "Trusted networks: $LAN_V4_CIDR and $LAN_V6_CIDR."
ufw status verbose
