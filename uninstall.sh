#!/usr/bin/env bash
# BLOGRON Panel Uninstaller
set -euo pipefail

RED='\033[0;31m'; YELLOW='\033[1;33m'; NC='\033[0m'; BOLD='\033[1m'

[[ $EUID -ne 0 ]] && echo -e "${RED}Run as root${NC}" && exit 1

echo -e "${YELLOW}${BOLD}WARNING: This will remove BLOGRON Panel and all its configuration.${NC}"
read -rp "Type 'yes' to confirm: " CONFIRM
[[ "$CONFIRM" != "yes" ]] && echo "Aborted." && exit 0

echo "Stopping and disabling service..."
systemctl stop blogron 2>/dev/null || true
systemctl disable blogron 2>/dev/null || true
rm -f /etc/systemd/system/blogron.service
systemctl daemon-reload

echo "Removing install directory..."
rm -rf /opt/blogron

echo "Removing sudo rules..."
rm -f /etc/sudoers.d/blogron

echo "Removing system user..."
userdel blogron 2>/dev/null || true

echo "Removing fail2ban config..."
rm -f /etc/fail2ban/jail.d/blogron.conf
systemctl reload fail2ban 2>/dev/null || true

echo -e "${YELLOW}Note: Nginx, MySQL, BIND9, Postfix, vsftpd packages were NOT removed.${NC}"
echo -e "${YELLOW}Remove manually if needed: apt remove nginx mysql-server bind9 postfix vsftpd dovecot-core${NC}"
echo "BLOGRON Panel uninstalled."
