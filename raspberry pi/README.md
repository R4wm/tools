# Raspberry Pi DMZ firewall

`secure-dmz-firewall.sh` configures a Raspberry Pi as a DMZ web host.

It permits only TCP ports 22 (SSH), 80 (HTTP), and 443 (HTTPS) from the
Internet. All other inbound traffic is denied. The configured trusted LANs can
still access all services, including NFS and Samba.

The script also protects Docker-published ports. Docker normally bypasses UFW
for forwarded ports, so the script installs and enables
`docker-user-firewall.service`, which blocks WAN access to Docker containers
except TCP 80 and 443.

## Current network values

- WAN interface: `eth0`
- IPv4 LAN: `10.0.0.0/24`
- IPv6 LAN: `fdac:3746:2b2e:a8e9::/64`

## Apply

Run on the Pi as root:

```bash
sudo ./secure-dmz-firewall.sh
```

Override the detected deployment values if the network changes:

```bash
LAN_V4_CIDR=192.168.1.0/24 WAN_IF=eth0 sudo ./secure-dmz-firewall.sh
```

The script resets existing UFW user rules before applying its declared policy.
Review custom firewall rules before rerunning it.
