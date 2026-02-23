#!/usr/bin/env bash
# ============================================================
# BLOGRON Panel — GitHub Push Script
# Run this once after extracting the package on your machine
# ============================================================
set -e

REPO="https://github.com/BLOGRON-Dev/BLOGRON-Panel.git"

echo "BLOGRON Panel — GitHub Upload"
echo "=============================="
echo ""
echo "This will push all files to: $REPO"
echo ""
read -rp "Enter your GitHub username: " GH_USER
read -rsp "Enter your GitHub Personal Access Token (PAT): " GH_TOKEN
echo ""

# Set remote with credentials
git remote remove origin 2>/dev/null || true
git remote add origin "https://${GH_USER}:${GH_TOKEN}@github.com/BLOGRON-Dev/BLOGRON-Panel.git"

# Stage everything
git add -A

# Initial commit
git commit -m "feat: initial release of BLOGRON Panel v1.0.0

- Go backend API with JWT authentication
- React + Tailwind CSS dark dashboard frontend
- 9 management modules: Dashboard, Users, Web Server, Databases,
  File Manager, Email, DNS, Cron Jobs, FTP
- One-command VPS installer (Ubuntu 22.04/24.04, Debian 11/12)
- GitHub Actions CI/CD workflows
- Security: scoped sudo, command allowlist, path traversal guard"

# Rename branch to main
git branch -M main

# Push
echo ""
echo "Pushing to GitHub..."
git push -u origin main

echo ""
echo "Done! View your repo at: https://github.com/BLOGRON-Dev/BLOGRON-Panel"
echo ""
echo "Next steps:"
echo "  1. Create a release tag: git tag v1.0.0 && git push origin v1.0.0"
echo "  2. GitHub Actions will automatically build and attach release assets"
