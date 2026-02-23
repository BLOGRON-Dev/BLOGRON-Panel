# BLOGRON Panel

<p align="center">
  <img src="https://img.shields.io/badge/version-1.0.0-cyan?style=for-the-badge" />
  <img src="https://img.shields.io/badge/go-1.22-blue?style=for-the-badge&logo=go" />
  <img src="https://img.shields.io/badge/react-18-61DAFB?style=for-the-badge&logo=react" />
  <img src="https://img.shields.io/badge/license-MIT-green?style=for-the-badge" />
  <img src="https://img.shields.io/badge/platform-Ubuntu%20%7C%20Debian-orange?style=for-the-badge&logo=linux" />
</p>

<p align="center">
  A self-hosted, open-source Linux VPS control panel built with <strong>Go</strong> and <strong>React</strong>.<br/>
  Manage your entire server from a beautiful dark dashboard — no subscription, no vendor lock-in.
</p>

---

## Features

| Module | Technology | Capabilities |
|--------|-----------|--------------|
| **Dashboard** | /proc, systemd | Live CPU · RAM · Disk · Service status · Activity logs |
| **Users** | useradd / usermod | Create · Suspend · Delete Linux system users |
| **Web Server** | Nginx | Virtual hosts · PHP versions · SSL via Let's Encrypt |
| **Databases** | MySQL 8 | Create · Drop databases · Manage users and grants |
| **File Manager** | OS filesystem | Browse · Upload · Create · Delete · Rename files |
| **Email** | Postfix + Dovecot | Mail domains · Mailboxes · Queue management |
| **DNS** | BIND9 | Zone files · A · CNAME · MX · TXT · NS records |
| **Cron Jobs** | crontab | Schedule · Edit · Run · Delete cron tasks |
| **FTP** | vsftpd | FTP account management with chroot isolation |

---

## Quick Install

> Requires: Ubuntu 22.04 / 24.04 or Debian 11 / 12

```bash
wget https://github.com/BLOGRON-Dev/BLOGRON-Panel/releases/latest/download/blogron-panel-latest.tar.gz
tar -xzf blogron-panel-latest.tar.gz
cd release && sudo bash install.sh
```

The installer handles everything: Go, Node.js, Nginx, MySQL, BIND9, Postfix, Dovecot, vsftpd, fail2ban, UFW, and optional SSL via Let's Encrypt.

---

## Project Structure

```
BLOGRON-Panel/
├── install.sh              # One-command VPS installer
├── uninstall.sh            # Clean removal script
├── backend/                # Go API server
│   ├── main.go
│   ├── go.mod
│   ├── api/                # Route handlers (auth, system, users, vhosts, db, files, email, dns, cron, ftp)
│   ├── middleware/         # JWT auth middleware
│   ├── util/               # Command allowlist, sanitizer, helpers
│   ├── blogron.service     # systemd unit
│   └── blogron.sudoers     # Scoped sudo rules
└── frontend/               # React + Vite + Tailwind CSS
    └── src/App.jsx         # All 9 panel modules
```

---

## Security

- Non-root API execution via dedicated `blogron` system user
- Scoped sudo — only whitelisted commands can run
- Command allowlist prevents arbitrary shell execution
- Input sanitization and shell metacharacter rejection on all args
- Path traversal protection in file manager
- JWT auth with 8-hour expiry on all routes
- fail2ban + UFW configured automatically on install

---

## Development

```bash
git clone https://github.com/BLOGRON-Dev/BLOGRON-Panel.git
cd BLOGRON-Panel

# Backend
cd backend && go mod tidy
JWT_SECRET=dev ADMIN_USER=admin ADMIN_PASSWORD=changeme go run .

# Frontend (new terminal)
cd frontend && npm install && npm run dev
```

---

## Roadmap

- [ ] Multi-user panel accounts with RBAC
- [ ] Backup manager (local + S3)
- [ ] Docker container management
- [ ] Two-factor authentication (TOTP)
- [ ] Real-time web terminal (SSH in browser)
- [ ] Let's Encrypt auto-renewal dashboard

---

## Contributing

1. Fork the repo
2. Create a branch: `git checkout -b feature/your-feature`
3. Commit: `git commit -m 'feat: add your feature'`
4. Push and open a Pull Request

---

## License

MIT — see [LICENSE](LICENSE) for details.

---

<p align="center">Built by <a href="https://github.com/BLOGRON-Dev">BLOGRON-Dev</a></p>
