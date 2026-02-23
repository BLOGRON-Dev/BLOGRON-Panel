package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"blogron/util"
)

const (
	nginxSitesAvailable = "/etc/nginx/sites-available"
	nginxSitesEnabled   = "/etc/nginx/sites-enabled"
	webRoot             = "/var/www"
)

type Vhost struct {
	Domain  string `json:"domain"`
	DocRoot string `json:"docroot"`
	SSL     bool   `json:"ssl"`
	Enabled bool   `json:"enabled"`
	PHP     string `json:"php"`
	IP      string `json:"ip"`
}

type createVhostRequest struct {
	Domain  string `json:"domain"`
	DocRoot string `json:"docroot"`
	PHP     string `json:"php"`
	SSL     bool   `json:"ssl"`
}

// ListVhosts godoc
// GET /api/vhosts
func ListVhosts(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(nginxSitesAvailable)
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, "cannot read nginx sites-available")
		return
	}

	var vhosts []Vhost
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".conf") {
			continue
		}
		domain := strings.TrimSuffix(e.Name(), ".conf")
		vh := parseVhostConf(domain)
		_, errStat := os.Stat(filepath.Join(nginxSitesEnabled, e.Name()))
		vh.Enabled = errStat == nil
		vhosts = append(vhosts, vh)
	}
	util.WriteJSON(w, http.StatusOK, vhosts)
}

// CreateVhost godoc
// POST /api/vhosts
func CreateVhost(w http.ResponseWriter, r *http.Request) {
	var req createVhostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	domain := util.Sanitize(req.Domain)
	if domain == "" {
		util.WriteError(w, http.StatusBadRequest, "invalid domain")
		return
	}

	phpVersion := req.PHP
	if phpVersion == "" {
		phpVersion = "8.2"
	}

	docroot := req.DocRoot
	if docroot == "" {
		docroot = fmt.Sprintf("%s/%s/public_html", webRoot, domain)
	}

	// Write nginx config
	confPath := filepath.Join(nginxSitesAvailable, domain+".conf")
	conf := buildNginxConfig(domain, docroot, phpVersion, req.SSL)

	if err := os.WriteFile(confPath, []byte(conf), 0644); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "failed to write nginx config: "+err.Error())
		return
	}

	// Create document root
	util.RunCmd("mkdir", "-p", docroot)
	util.RunCmd("chown", "www-data:www-data", docroot)

	// Enable by default
	symlink := filepath.Join(nginxSitesEnabled, domain+".conf")
	os.Symlink(confPath, symlink) // nolint: ignore if already exists

	// Test and reload nginx
	if _, err := util.RunCmd("nginx", "-t"); err != nil {
		os.Remove(confPath)
		os.Remove(symlink)
		util.WriteError(w, http.StatusInternalServerError, "nginx config test failed: "+err.Error())
		return
	}
	util.RunCmd("systemctl", "reload", "nginx")

	util.WriteJSON(w, http.StatusCreated, map[string]string{"status": "created", "domain": domain})
}

// DeleteVhost godoc
// DELETE /api/vhosts/{domain}
func DeleteVhost(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	if domain == "" {
		util.WriteError(w, http.StatusBadRequest, "invalid domain")
		return
	}

	confFile := domain + ".conf"
	os.Remove(filepath.Join(nginxSitesEnabled, confFile))
	os.Remove(filepath.Join(nginxSitesAvailable, confFile))

	util.RunCmd("systemctl", "reload", "nginx")
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// EnableVhost godoc
// POST /api/vhosts/{domain}/enable
func EnableVhost(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	confFile := domain + ".conf"
	src := filepath.Join(nginxSitesAvailable, confFile)
	dst := filepath.Join(nginxSitesEnabled, confFile)

	if _, err := os.Stat(src); os.IsNotExist(err) {
		util.WriteError(w, http.StatusNotFound, "vhost not found")
		return
	}

	os.Symlink(src, dst)
	util.RunCmd("systemctl", "reload", "nginx")
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "enabled"})
}

// DisableVhost godoc
// POST /api/vhosts/{domain}/disable
func DisableVhost(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	symlink := filepath.Join(nginxSitesEnabled, domain+".conf")
	os.Remove(symlink)
	util.RunCmd("systemctl", "reload", "nginx")
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "disabled"})
}

// EnableSSL godoc
// POST /api/vhosts/{domain}/ssl
func EnableSSL(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	if domain == "" {
		util.WriteError(w, http.StatusBadRequest, "invalid domain")
		return
	}

	var body struct {
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	email := util.Sanitize(body.Email)

	args := []string{"--nginx", "-d", domain, "--non-interactive", "--agree-tos"}
	if email != "" {
		args = append(args, "--email", email)
	} else {
		args = append(args, "--register-unsafely-without-email")
	}

	if _, err := util.RunCmd("certbot", args...); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "certbot failed: "+err.Error())
		return
	}

	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "ssl_enabled", "domain": domain})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func buildNginxConfig(domain, docroot, phpVersion, ssl interface{}) string {
	d := fmt.Sprintf("%v", domain)
	dr := fmt.Sprintf("%v", docroot)
	php := fmt.Sprintf("%v", phpVersion)

	phpSocket := fmt.Sprintf("/run/php/php%s-fpm.sock", php)

	return fmt.Sprintf(`server {
    listen 80;
    listen [::]:80;

    server_name %s www.%s;
    root %s;
    index index.php index.html index.htm;

    access_log /var/log/nginx/%s.access.log;
    error_log  /var/log/nginx/%s.error.log;

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:%s;
    }

    location ~ /\.ht {
        deny all;
    }

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
}
`, d, d, dr, d, d, phpSocket)
}

func parseVhostConf(domain string) Vhost {
	vh := Vhost{Domain: domain}
	confPath := filepath.Join(nginxSitesAvailable, domain+".conf")
	data, err := os.ReadFile(confPath)
	if err != nil {
		return vh
	}
	content := string(data)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "root ") {
			vh.DocRoot = strings.TrimSuffix(strings.TrimPrefix(line, "root "), ";")
		}
		if strings.Contains(line, "ssl_certificate") {
			vh.SSL = true
		}
		if strings.Contains(line, "php") && strings.Contains(line, "fpm") {
			// Extract PHP version from socket path e.g. php8.2-fpm
			for _, v := range []string{"8.3", "8.2", "8.1", "8.0", "7.4"} {
				if strings.Contains(line, "php"+v) {
					vh.PHP = v
					break
				}
			}
		}
	}
	return vh
}
