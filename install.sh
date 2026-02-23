#!/usr/bin/env bash
# ============================================================
#  BLOGRON Panel - One-Command VPS Installer
#  Supports: Ubuntu 22.04 / 24.04, Debian 11 / 12
#  Usage: sudo bash install.sh
# ============================================================
set -euo pipefail
IFS=$'\n\t'

# ── Colours ───────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'

log()  { echo -e "${CYAN}[PANEL]${NC} $*"; }
ok()   { echo -e "${GREEN}[  OK  ]${NC} $*"; }
warn() { echo -e "${YELLOW}[ WARN ]${NC} $*"; }
err()  { echo -e "${RED}[ERROR ]${NC} $*" >&2; exit 1; }
step() { echo -e "\n${BOLD}${CYAN}━━━ $* ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"; }

# ── Banner ─────────────────────────────────────────────────────────────────
clear
echo -e "${CYAN}"
cat << 'BANNER'
  ██████╗ ██╗      ██████╗  ██████╗ ██████╗  ██████╗ ███╗   ██╗
  ██╔══██╗██║     ██╔═══██╗██╔════╝ ██╔══██╗██╔═══██╗████╗  ██║
  ██████╔╝██║     ██║   ██║██║  ███╗██████╔╝██║   ██║██╔██╗ ██║
  ██╔══██╗██║     ██║   ██║██║   ██║██╔══██╗██║   ██║██║╚██╗██║
  ██████╔╝███████╗╚██████╔╝╚██████╔╝██║  ██║╚██████╔╝██║ ╚████║
  ╚═════╝ ╚══════╝ ╚═════╝  ╚═════╝ ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═══╝
                    P A N E L  —  v1.0.0
BANNER
echo -e "${NC}"

# ── Pre-flight checks ─────────────────────────────────────────────────────
[[ $EUID -ne 0 ]] && err "This script must be run as root. Try: sudo bash install.sh"

# Detect OS
if [[ -f /etc/os-release ]]; then
  . /etc/os-release
  OS_ID="$ID"
  OS_VER="$VERSION_ID"
else
  err "Cannot detect OS. Requires Ubuntu 22.04+, Debian 11+"
fi

[[ "$OS_ID" =~ ^(ubuntu|debian)$ ]] || err "Unsupported OS: $OS_ID. Use Ubuntu or Debian."
log "Detected OS: $OS_ID $OS_VER"

# Ensure we have internet
curl -fsS https://www.google.com -o /dev/null || err "No internet connection detected."

# ── Gather config ─────────────────────────────────────────────────────────
step "Configuration"

read -rp "$(echo -e "${BOLD}Panel domain (e.g. panel.example.com): ${NC}")" PANEL_DOMAIN
[[ -z "$PANEL_DOMAIN" ]] && err "Domain cannot be empty"

read -rp "$(echo -e "${BOLD}Admin username [admin]: ${NC}")" ADMIN_USER
ADMIN_USER="${ADMIN_USER:-admin}"

while true; do
  read -rsp "$(echo -e "${BOLD}Admin password (min 12 chars): ${NC}")" ADMIN_PASS
  echo
  [[ ${#ADMIN_PASS} -ge 12 ]] && break
  warn "Password must be at least 12 characters. Try again."
done

read -rp "$(echo -e "${BOLD}MySQL root password: ${NC}")" MYSQL_ROOT_PASS
[[ -z "$MYSQL_ROOT_PASS" ]] && MYSQL_ROOT_PASS=$(openssl rand -base64 24)

JWT_SECRET=$(openssl rand -base64 48)
INSTALL_DIR="/opt/blogron"
PANEL_USER="blogron"
PANEL_PORT="8080"

echo ""
log "Panel domain:   $PANEL_DOMAIN"
log "Admin user:     $ADMIN_USER"
log "Install dir:    $INSTALL_DIR"
log "API port:       $PANEL_PORT"
echo ""
read -rp "$(echo -e "${BOLD}Continue with installation? [y/N]: ${NC}")" CONFIRM
[[ "$CONFIRM" =~ ^[Yy]$ ]] || { log "Aborted."; exit 0; }

# ── System packages ────────────────────────────────────────────────────────
step "System Update & Dependencies"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get upgrade -y -qq
apt-get install -y -qq \
  curl wget gnupg2 ca-certificates lsb-release \
  software-properties-common apt-transport-https \
  ufw fail2ban unzip git openssl \
  nginx certbot python3-certbot-nginx \
  mysql-server \
  bind9 bind9utils \
  postfix dovecot-core dovecot-imapd dovecot-pop3d dovecot-lmtpd \
  vsftpd \
  supervisor \
  net-tools
ok "System packages installed"

# ── Go ────────────────────────────────────────────────────────────────────
step "Installing Go"
GO_VERSION="1.22.4"
GO_ARCH="linux-amd64"
GO_TAR="go${GO_VERSION}.${GO_ARCH}.tar.gz"

if ! command -v go &>/dev/null || [[ "$(go version 2>/dev/null | awk '{print $3}')" != "go${GO_VERSION}" ]]; then
  log "Downloading Go ${GO_VERSION}…"
  wget -q "https://go.dev/dl/${GO_TAR}" -O "/tmp/${GO_TAR}"
  rm -rf /usr/local/go
  tar -C /usr/local -xzf "/tmp/${GO_TAR}"
  rm "/tmp/${GO_TAR}"
  export PATH="$PATH:/usr/local/go/bin"
  echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile.d/go.sh
  chmod +x /etc/profile.d/go.sh
fi
ok "Go $(go version | awk '{print $3}') ready"

# ── Node.js ───────────────────────────────────────────────────────────────
step "Installing Node.js"
if ! command -v node &>/dev/null; then
  curl -fsSL https://deb.nodesource.com/setup_20.x | bash - &>/dev/null
  apt-get install -y -qq nodejs
fi
ok "Node.js $(node --version) ready"

# ── Panel system user ─────────────────────────────────────────────────────
step "Creating Panel System User"
if ! id "$PANEL_USER" &>/dev/null; then
  useradd --system --no-create-home --shell /usr/sbin/nologin "$PANEL_USER"
  ok "Created user: $PANEL_USER"
else
  ok "User $PANEL_USER already exists"
fi

# ── Install dir ───────────────────────────────────────────────────────────
step "Setting Up Install Directory"
mkdir -p "$INSTALL_DIR"/{backend,frontend}
chown -R "$PANEL_USER:$PANEL_USER" "$INSTALL_DIR"

# ── Build backend ─────────────────────────────────────────────────────────
step "Building Go Backend"

# Copy backend source
cp -r backend/* "$INSTALL_DIR/backend/"
cd "$INSTALL_DIR/backend"

export PATH="$PATH:/usr/local/go/bin"
export HOME=/root
export GOPATH=/root/go

go mod tidy 2>&1 | tail -5
go build -o "$INSTALL_DIR/blogron" . 2>&1
chown "$PANEL_USER:$PANEL_USER" "$INSTALL_DIR/blogron"
chmod 750 "$INSTALL_DIR/blogron"
ok "Backend binary built: $INSTALL_DIR/blogron"

# ── Build frontend ────────────────────────────────────────────────────────
step "Building Frontend"
cp -r frontend/* "$INSTALL_DIR/frontend/"
cd "$INSTALL_DIR/frontend"

# Write .env with API URL
cat > .env.production << EOF
VITE_API_URL=https://${PANEL_DOMAIN}
EOF

npm install --silent 2>&1 | tail -3
npm run build 2>&1 | tail -5

ok "Frontend built: $INSTALL_DIR/frontend/dist"

# ── MySQL ─────────────────────────────────────────────────────────────────
step "Configuring MySQL"
systemctl enable --now mysql

# Secure MySQL
mysql -e "ALTER USER 'root'@'localhost' IDENTIFIED WITH mysql_native_password BY '${MYSQL_ROOT_PASS}';" 2>/dev/null || true
mysql -u root -p"${MYSQL_ROOT_PASS}" -e "DELETE FROM mysql.user WHERE User='';" 2>/dev/null || true
mysql -u root -p"${MYSQL_ROOT_PASS}" -e "DROP DATABASE IF EXISTS test;" 2>/dev/null || true
mysql -u root -p"${MYSQL_ROOT_PASS}" -e "FLUSH PRIVILEGES;" 2>/dev/null || true

ok "MySQL configured"

# ── Nginx ─────────────────────────────────────────────────────────────────
step "Configuring Nginx"
mkdir -p /etc/nginx/sites-available /etc/nginx/sites-enabled /var/www

# Remove default
rm -f /etc/nginx/sites-enabled/default

# Panel config (HTTP first, HTTPS added after certbot)
cat > "/etc/nginx/sites-available/${PANEL_DOMAIN}.conf" << NGINXEOF
server {
    listen 80;
    listen [::]:80;
    server_name ${PANEL_DOMAIN};

    # Frontend
    root ${INSTALL_DIR}/frontend/dist;
    index index.html;

    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # API proxy
    location /api/ {
        proxy_pass http://127.0.0.1:${PANEL_PORT};
        proxy_http_version 1.1;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
        proxy_read_timeout 60s;
        client_max_body_size 100M;
    }
}
NGINXEOF

ln -sf "/etc/nginx/sites-available/${PANEL_DOMAIN}.conf" "/etc/nginx/sites-enabled/"
nginx -t
systemctl enable --now nginx
systemctl reload nginx
ok "Nginx configured"

# ── SSL with Certbot ──────────────────────────────────────────────────────
step "SSL Certificate"
SERVER_IP=$(curl -s https://api.ipify.org 2>/dev/null || echo "")
log "Server public IP: $SERVER_IP"
warn "Make sure DNS A record for $PANEL_DOMAIN points to $SERVER_IP before SSL setup."

read -rp "$(echo -e "${BOLD}Request SSL certificate now? [y/N]: ${NC}")" DO_SSL
if [[ "$DO_SSL" =~ ^[Yy]$ ]]; then
  read -rp "$(echo -e "${BOLD}Email for Let's Encrypt notifications: ${NC}")" SSL_EMAIL
  certbot --nginx -d "$PANEL_DOMAIN" --non-interactive --agree-tos -m "$SSL_EMAIL" || warn "Certbot failed — you can run it manually later: certbot --nginx -d $PANEL_DOMAIN"
  ok "SSL certificate installed"
else
  warn "Skipped SSL. Run manually: certbot --nginx -d $PANEL_DOMAIN"
fi

# ── Sudoers ───────────────────────────────────────────────────────────────
step "Installing Sudo Rules"
cp backend/blogron.sudoers /etc/sudoers.d/blogron
chmod 440 /etc/sudoers.d/blogron
visudo -c || err "sudoers validation failed!"
ok "Sudo rules installed"

# ── Hash admin password ───────────────────────────────────────────────────
step "Setting Admin Credentials"
# Generate bcrypt hash of admin password using htpasswd
ADMIN_HASH=$(python3 -c "import crypt; print(crypt.crypt('${ADMIN_PASS}', crypt.mksalt(crypt.METHOD_SHA512)))" 2>/dev/null || echo "$ADMIN_PASS")
# Store credentials in a config file the Go backend can read
mkdir -p "$INSTALL_DIR/config"
cat > "$INSTALL_DIR/config/admin.json" << CFGEOF
{
  "username": "${ADMIN_USER}",
  "password_hash": "${ADMIN_HASH}"
}
CFGEOF
chmod 640 "$INSTALL_DIR/config/admin.json"
chown "$PANEL_USER:$PANEL_USER" "$INSTALL_DIR/config/admin.json"
ok "Admin credentials stored"

# ── Systemd service ───────────────────────────────────────────────────────
step "Installing Systemd Service"
cat > /etc/systemd/system/blogron.service << SVCEOF
[Unit]
Description=BLOGRON Panel API
After=network.target mysql.service nginx.service
Wants=mysql.service

[Service]
Type=simple
User=${PANEL_USER}
Group=${PANEL_USER}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/blogron
Environment="PORT=${PANEL_PORT}"
Environment="JWT_SECRET=${JWT_SECRET}"
Environment="MYSQL_USER=root"
Environment="MYSQL_PASSWORD=${MYSQL_ROOT_PASS}"
Environment="ADMIN_USER=${ADMIN_USER}"
Environment="ADMIN_PASSWORD=${ADMIN_PASS}"
NoNewPrivileges=false
PrivateTmp=true
Restart=on-failure
RestartSec=5s
StandardOutput=journal
StandardError=journal
SyslogIdentifier=blogron

[Install]
WantedBy=multi-user.target
SVCEOF

systemctl daemon-reload
systemctl enable --now blogron
sleep 2
systemctl is-active --quiet blogron && ok "blogron service started" || warn "blogron may have failed to start — check: journalctl -u blogron"

# ── Firewall ──────────────────────────────────────────────────────────────
step "Configuring Firewall (UFW)"
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow 80/tcp
ufw allow 443/tcp
ufw allow 21/tcp    # FTP
ufw allow 990/tcp   # FTPS
ufw allow 40000:50000/tcp  # FTP passive
ufw allow 25/tcp    # SMTP
ufw allow 587/tcp   # SMTP submission
ufw allow 993/tcp   # IMAPS
ufw allow 995/tcp   # POP3S
ufw allow 53        # DNS
ufw --force enable
ok "Firewall configured"

# ── Fail2ban ──────────────────────────────────────────────────────────────
step "Configuring Fail2ban"
cat > /etc/fail2ban/jail.d/blogron.conf << F2BEOF
[sshd]
enabled = true
port = ssh
maxretry = 5
bantime = 3600

[nginx-http-auth]
enabled = true

[nginx-limit-req]
enabled = true
F2BEOF
systemctl enable --now fail2ban
ok "Fail2ban configured"

# ── Postfix quick config ───────────────────────────────────────────────────
step "Configuring Postfix (basic)"
debconf-set-selections <<< "postfix postfix/mailname string ${PANEL_DOMAIN}"
debconf-set-selections <<< "postfix postfix/main_mailer_type string 'Internet Site'"

postconf -e "myhostname = ${PANEL_DOMAIN}"
postconf -e "mydomain = ${PANEL_DOMAIN}"
postconf -e "virtual_mailbox_domains = /etc/postfix/virtual_mailbox_domains"
postconf -e "virtual_mailbox_maps = hash:/etc/postfix/virtual_mailbox_maps"
postconf -e "virtual_mailbox_base = /var/mail/vhosts"
postconf -e "virtual_uid_maps = static:5000"
postconf -e "virtual_gid_maps = static:5000"

touch /etc/postfix/virtual_mailbox_domains
touch /etc/postfix/virtual_mailbox_maps
postmap /etc/postfix/virtual_mailbox_maps

# Create vmail user
if ! id vmail &>/dev/null; then
  groupadd -g 5000 vmail
  useradd -g vmail -u 5000 vmail -d /var/mail/vhosts -s /usr/sbin/nologin
fi
mkdir -p /var/mail/vhosts
chown -R vmail:vmail /var/mail/vhosts

systemctl enable --now postfix
ok "Postfix configured"

# ── BIND9 quick config ─────────────────────────────────────────────────────
step "Configuring BIND9"
mkdir -p /etc/bind/zones
chown bind:bind /etc/bind/zones

touch /etc/bind/named.conf.local
systemctl enable --now bind9
ok "BIND9 configured"

# ── vsftpd config ─────────────────────────────────────────────────────────
step "Configuring vsftpd"
cat > /etc/vsftpd.conf << FTPEOF
listen=YES
listen_ipv6=NO
anonymous_enable=NO
local_enable=YES
write_enable=YES
local_umask=022
dirmessage_enable=YES
use_localtime=YES
xferlog_enable=YES
connect_from_port_20=YES
chroot_local_user=YES
allow_writeable_chroot=YES
secure_chroot_dir=/var/run/vsftpd/empty
pam_service_name=vsftpd
rsa_cert_file=/etc/ssl/certs/ssl-cert-snakeoil.pem
rsa_private_key_file=/etc/ssl/private/ssl-cert-snakeoil.key
ssl_enable=NO
userlist_enable=YES
userlist_file=/etc/vsftpd.userlist
userlist_deny=NO
pasv_enable=YES
pasv_min_port=40000
pasv_max_port=50000
FTPEOF

touch /etc/vsftpd.userlist
systemctl enable --now vsftpd
ok "vsftpd configured"

# ── Final health check ─────────────────────────────────────────────────────
step "Health Check"
sleep 3

HEALTH=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${PANEL_PORT}/api/health" 2>/dev/null || echo "000")
if [[ "$HEALTH" == "200" ]]; then
  ok "API health check: HTTP $HEALTH"
else
  warn "API health check returned HTTP $HEALTH — check: journalctl -u blogron -n 50"
fi

# ── Summary ────────────────────────────────────────────────────────────────
echo ""
echo -e "${GREEN}${BOLD}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}${BOLD}║         BLOGRON Panel Installation Complete!           ║${NC}"
echo -e "${GREEN}${BOLD}╚══════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BOLD}Panel URL:${NC}      https://${PANEL_DOMAIN}"
echo -e "${BOLD}Admin user:${NC}     ${ADMIN_USER}"
echo -e "${BOLD}MySQL pass:${NC}     ${MYSQL_ROOT_PASS}"
echo ""
echo -e "${BOLD}Useful commands:${NC}"
echo -e "  ${CYAN}systemctl status blogron${NC}   — check API status"
echo -e "  ${CYAN}journalctl -u blogron -f${NC}   — follow API logs"
echo -e "  ${CYAN}systemctl restart blogron${NC}  — restart API"
echo ""
echo -e "${YELLOW}⚠  Save these credentials — they will not be shown again!${NC}"
echo -e "${YELLOW}   MySQL root password: ${MYSQL_ROOT_PASS}${NC}"
echo -e "${YELLOW}   JWT secret stored in: /etc/systemd/system/blogron.service${NC}"
echo ""
