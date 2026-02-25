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

# Tag and push to trigger the GitHub Actions release workflow
echo ""
read -rp "Tag version for release (e.g. v1.0.0): " RELEASE_TAG
[[ -z "$RELEASE_TAG" ]] && RELEASE_TAG="v1.0.0"

git tag "$RELEASE_TAG"
git push origin "$RELEASE_TAG"

echo ""
echo "Done! GitHub Actions will now build and publish the release automatically."
echo "View your repo at:     https://github.com/BLOGRON-Dev/BLOGRON-Panel"
echo "View the release at:   https://github.com/BLOGRON-Dev/BLOGRON-Panel/releases"
echo ""
echo "Once the workflow completes (~2-3 min), install with:"
echo "  wget https://github.com/BLOGRON-Dev/BLOGRON-Panel/releases/download/${RELEASE_TAG}/blogron-panel-latest.tar.gz"
echo "  tar -xzf blogron-panel-latest.tar.gz"
echo "  cd release && sudo bash install.sh"
