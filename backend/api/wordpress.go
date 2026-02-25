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

const wpRoot = "/var/www"
const wpCliPath = "/usr/local/bin/wp"

// ── Types ────────────────────────────────────────────────────────────────────

type WPSite struct {
	Domain    string `json:"domain"`
	Path      string `json:"path"`
	Title     string `json:"title"`
	WPVersion string `json:"wp_version"`
	DBName    string `json:"db_name"`
	DBUser    string `json:"db_user"`
	Active    bool   `json:"active"`
	SSL       bool   `json:"ssl"`
}

type WPPlugin struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version"`
	Title   string `json:"title"`
}

type WPTheme struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Version string `json:"version"`
	Title   string `json:"title"`
}

type createWPRequest struct {
	Domain    string `json:"domain"`
	SiteTitle string `json:"site_title"`
	AdminUser string `json:"admin_user"`
	AdminPass string `json:"admin_pass"`
	AdminEmail string `json:"admin_email"`
	DBName    string `json:"db_name"`
	DBUser    string `json:"db_user"`
	DBPass    string `json:"db_pass"`
	PHP       string `json:"php"`
}

// ── Routes ────────────────────────────────────────────────────────────────────

// ListWPSites godoc
// GET /api/wordpress
func ListWPSites(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(wpRoot)
	if err != nil {
		util.WriteJSON(w, http.StatusOK, []WPSite{})
		return
	}

	var sites []WPSite
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		domain := e.Name()
		wpConfigPath := filepath.Join(wpRoot, domain, "public_html", "wp-config.php")
		if _, err := os.Stat(wpConfigPath); os.IsNotExist(err) {
			// Also check root level (some installs don't use public_html)
			wpConfigPath = filepath.Join(wpRoot, domain, "wp-config.php")
			if _, err := os.Stat(wpConfigPath); os.IsNotExist(err) {
				continue
			}
		}
		site := probeWPSite(domain, wpConfigPath)
		sites = append(sites, site)
	}
	util.WriteJSON(w, http.StatusOK, sites)
}

// CreateWPSite godoc
// POST /api/wordpress
func CreateWPSite(w http.ResponseWriter, r *http.Request) {
	var req createWPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	domain := util.Sanitize(req.Domain)
	if domain == "" {
		util.WriteError(w, http.StatusBadRequest, "domain is required")
		return
	}

	// Defaults
	if req.SiteTitle == "" {
		req.SiteTitle = domain
	}
	if req.AdminUser == "" {
		req.AdminUser = "admin"
	}
	if req.AdminEmail == "" {
		req.AdminEmail = "admin@" + domain
	}
	dbName := util.Sanitize(req.DBName)
	if dbName == "" {
		dbName = "wp_" + strings.ReplaceAll(domain, ".", "_")
	}
	dbUser := util.Sanitize(req.DBUser)
	if dbUser == "" {
		// Truncate to 16 chars (MySQL user limit)
		raw := "wp_" + strings.ReplaceAll(domain, ".", "_")
		if len(raw) > 16 {
			raw = raw[:16]
		}
		dbUser = raw
	}
	dbPass := req.DBPass
	if dbPass == "" {
		dbPass = randomPass(20)
	}
	phpVersion := req.PHP
	if phpVersion == "" {
		phpVersion = os.Getenv("PHP_VERSION")
	}
	if phpVersion == "" {
		phpVersion = "8.2"
	}

	docroot := filepath.Join(wpRoot, domain, "public_html")

	// 1. Create MariaDB database + user
	setupSQL := fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"+
			"CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s';"+
			"GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost';"+
			"FLUSH PRIVILEGES;",
		dbName, dbUser, dbPass, dbName, dbUser,
	)
	if _, err := util.RunCmd("mysql", append(mysqlArgs("-e", setupSQL))...); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "failed to create database: "+err.Error())
		return
	}

	// 2. Create docroot
	if _, err := util.RunCmd("mkdir", "-p", docroot); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "failed to create docroot: "+err.Error())
		return
	}
	util.RunCmd("chown", "-R", "www-data:www-data", filepath.Join(wpRoot, domain))

	// 3. Download WordPress core via WP-CLI
	if _, err := wpCmd(docroot, "core", "download", "--locale=en_US"); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "wp core download failed: "+err.Error())
		return
	}

	// 4. Create wp-config.php
	if _, err := wpCmd(docroot,
		"config", "create",
		"--dbname="+dbName,
		"--dbuser="+dbUser,
		"--dbpass="+dbPass,
		"--dbhost=localhost",
		"--dbcharset=utf8mb4",
	); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "wp config create failed: "+err.Error())
		return
	}

	// 5. Run WP install
	siteURL := "http://" + domain
	if _, err := wpCmd(docroot,
		"core", "install",
		"--url="+siteURL,
		"--title="+req.SiteTitle,
		"--admin_user="+req.AdminUser,
		"--admin_password="+req.AdminPass,
		"--admin_email="+req.AdminEmail,
	); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "wp core install failed: "+err.Error())
		return
	}

	// 6. Set file ownership back to www-data
	util.RunCmd("chown", "-R", "www-data:www-data", filepath.Join(wpRoot, domain))

	// 7. Create nginx vhost for this WP site
	confPath := filepath.Join(nginxSitesAvailable, domain+".conf")
	conf := buildWPNginxConfig(domain, docroot, phpVersion)
	if err := os.WriteFile(confPath, []byte(conf), 0644); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "failed to write nginx config: "+err.Error())
		return
	}
	symlink := filepath.Join(nginxSitesEnabled, domain+".conf")
	os.Symlink(confPath, symlink)
	if _, err := util.RunCmd("nginx", "-t"); err != nil {
		os.Remove(confPath)
		os.Remove(symlink)
		util.WriteError(w, http.StatusInternalServerError, "nginx config test failed: "+err.Error())
		return
	}
	util.RunCmd("systemctl", "reload", "nginx")

	util.WriteJSON(w, http.StatusCreated, map[string]string{
		"status":   "created",
		"domain":   domain,
		"db_name":  dbName,
		"db_user":  dbUser,
		"db_pass":  dbPass,
		"site_url": siteURL,
		"wp_admin": siteURL + "/wp-admin",
	})
}

// DeleteWPSite godoc
// DELETE /api/wordpress/{domain}
func DeleteWPSite(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	if domain == "" {
		util.WriteError(w, http.StatusBadRequest, "invalid domain")
		return
	}

	var body struct {
		DeleteDB bool `json:"delete_db"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	// Remove files
	siteDir := filepath.Join(wpRoot, domain)
	util.RunCmd("rm", "-rf", siteDir)

	// Remove nginx config
	util.RunCmd("rm", "-f", filepath.Join(nginxSitesEnabled, domain+".conf"))
	util.RunCmd("rm", "-f", filepath.Join(nginxSitesAvailable, domain+".conf"))
	util.RunCmd("systemctl", "reload", "nginx")

	// Optionally drop DB
	if body.DeleteDB {
		dbName := "wp_" + strings.ReplaceAll(domain, ".", "_")
		util.RunCmd("mysql", append(mysqlArgs("-e", "DROP DATABASE IF EXISTS `"+dbName+"`;"))...)
	}

	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ListWPPlugins godoc
// GET /api/wordpress/{domain}/plugins
func ListWPPlugins(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	docroot := resolveWPDocroot(domain)
	if docroot == "" {
		util.WriteError(w, http.StatusNotFound, "wordpress site not found")
		return
	}

	out, err := wpCmd(docroot, "plugin", "list", "--format=json")
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, "wp plugin list failed: "+err.Error())
		return
	}

	var plugins []WPPlugin
	if err := json.Unmarshal([]byte(out), &plugins); err != nil {
		// Return raw if JSON parse fails
		util.WriteJSON(w, http.StatusOK, []WPPlugin{})
		return
	}
	util.WriteJSON(w, http.StatusOK, plugins)
}

// InstallWPPlugin godoc
// POST /api/wordpress/{domain}/plugins
// Body: { "name": "woocommerce", "activate": true }
func InstallWPPlugin(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	docroot := resolveWPDocroot(domain)
	if docroot == "" {
		util.WriteError(w, http.StatusNotFound, "wordpress site not found")
		return
	}

	var body struct {
		Name     string `json:"name"`
		Activate bool   `json:"activate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		util.WriteError(w, http.StatusBadRequest, "plugin name is required")
		return
	}

	name := util.Sanitize(body.Name)
	args := []string{"plugin", "install", name}
	if body.Activate {
		args = append(args, "--activate")
	}
	if _, err := wpCmd(docroot, args...); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "plugin install failed: "+err.Error())
		return
	}
	util.RunCmd("chown", "-R", "www-data:www-data", filepath.Join(wpRoot, domain))
	util.WriteJSON(w, http.StatusCreated, map[string]string{"status": "installed", "plugin": name})
}

// ToggleWPPlugin godoc
// PUT /api/wordpress/{domain}/plugins/{plugin}
// Body: { "action": "activate" | "deactivate" | "delete" | "update" }
func ToggleWPPlugin(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	plugin := util.Sanitize(chi_urlParam(r, "plugin"))
	docroot := resolveWPDocroot(domain)
	if docroot == "" {
		util.WriteError(w, http.StatusNotFound, "wordpress site not found")
		return
	}

	var body struct {
		Action string `json:"action"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	allowed := map[string]bool{"activate": true, "deactivate": true, "delete": true, "update": true}
	if !allowed[body.Action] {
		util.WriteError(w, http.StatusBadRequest, "action must be activate, deactivate, delete, or update")
		return
	}

	if _, err := wpCmd(docroot, "plugin", body.Action, plugin); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "plugin "+body.Action+" failed: "+err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": body.Action + "d", "plugin": plugin})
}

// ListWPThemes godoc
// GET /api/wordpress/{domain}/themes
func ListWPThemes(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	docroot := resolveWPDocroot(domain)
	if docroot == "" {
		util.WriteError(w, http.StatusNotFound, "wordpress site not found")
		return
	}

	out, err := wpCmd(docroot, "theme", "list", "--format=json")
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, "wp theme list failed: "+err.Error())
		return
	}

	var themes []WPTheme
	json.Unmarshal([]byte(out), &themes)
	util.WriteJSON(w, http.StatusOK, themes)
}

// InstallWPTheme godoc
// POST /api/wordpress/{domain}/themes
// Body: { "name": "astra", "activate": true }
func InstallWPTheme(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	docroot := resolveWPDocroot(domain)
	if docroot == "" {
		util.WriteError(w, http.StatusNotFound, "wordpress site not found")
		return
	}

	var body struct {
		Name     string `json:"name"`
		Activate bool   `json:"activate"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		util.WriteError(w, http.StatusBadRequest, "theme name is required")
		return
	}

	name := util.Sanitize(body.Name)
	args := []string{"theme", "install", name}
	if body.Activate {
		args = append(args, "--activate")
	}
	if _, err := wpCmd(docroot, args...); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "theme install failed: "+err.Error())
		return
	}
	util.RunCmd("chown", "-R", "www-data:www-data", filepath.Join(wpRoot, domain))
	util.WriteJSON(w, http.StatusCreated, map[string]string{"status": "installed", "theme": name})
}

// ToggleWPTheme godoc
// PUT /api/wordpress/{domain}/themes/{theme}
func ToggleWPTheme(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	theme := util.Sanitize(chi_urlParam(r, "theme"))
	docroot := resolveWPDocroot(domain)
	if docroot == "" {
		util.WriteError(w, http.StatusNotFound, "wordpress site not found")
		return
	}

	var body struct {
		Action string `json:"action"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	allowed := map[string]bool{"activate": true, "delete": true, "update": true}
	if !allowed[body.Action] {
		util.WriteError(w, http.StatusBadRequest, "action must be activate, delete, or update")
		return
	}

	if _, err := wpCmd(docroot, "theme", body.Action, theme); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "theme "+body.Action+" failed: "+err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": body.Action + "d", "theme": theme})
}

// WPUpdateCore godoc
// POST /api/wordpress/{domain}/update
func WPUpdateCore(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	docroot := resolveWPDocroot(domain)
	if docroot == "" {
		util.WriteError(w, http.StatusNotFound, "wordpress site not found")
		return
	}

	out, err := wpCmd(docroot, "core", "update")
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, "wp core update failed: "+err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated", "output": out})
}

// WPMaintenanceMode godoc
// POST /api/wordpress/{domain}/maintenance
// Body: { "enable": true }
func WPMaintenanceMode(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	docroot := resolveWPDocroot(domain)
	if docroot == "" {
		util.WriteError(w, http.StatusNotFound, "wordpress site not found")
		return
	}

	var body struct {
		Enable bool `json:"enable"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	mode := "deactivate"
	if body.Enable {
		mode = "activate"
	}
	wpCmd(docroot, "maintenance-mode", mode)
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": map[bool]string{true: "enabled", false: "disabled"}[body.Enable]})
}

// WPSearchReplace godoc
// POST /api/wordpress/{domain}/search-replace
// Body: { "search": "http://old.com", "replace": "https://new.com" }
func WPSearchReplace(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	docroot := resolveWPDocroot(domain)
	if docroot == "" {
		util.WriteError(w, http.StatusNotFound, "wordpress site not found")
		return
	}

	var body struct {
		Search  string `json:"search"`
		Replace string `json:"replace"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Search == "" {
		util.WriteError(w, http.StatusBadRequest, "search and replace are required")
		return
	}

	out, err := wpCmd(docroot, "search-replace", body.Search, body.Replace, "--all-tables")
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, "search-replace failed: "+err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "done", "output": out})
}

// WPCacheFlush godoc
// POST /api/wordpress/{domain}/cache-flush
func WPCacheFlush(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	docroot := resolveWPDocroot(domain)
	if docroot == "" {
		util.WriteError(w, http.StatusNotFound, "wordpress site not found")
		return
	}

	wpCmd(docroot, "cache", "flush")
	wpCmd(docroot, "rewrite", "flush")
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "flushed"})
}

// ── helpers ───────────────────────────────────────────────────────────────────

// wpCmd runs a WP-CLI command as www-data in the given docroot.
func wpCmd(docroot string, args ...string) (string, error) {
	// Build: sudo -u www-data wp --path=<docroot> --allow-root <args...>
	cmdArgs := append([]string{"-u", "www-data", wpCliPath, "--path=" + docroot, "--allow-root"}, args...)
	return util.RunCmd("sudo", cmdArgs...)
}

func resolveWPDocroot(domain string) string {
	candidates := []string{
		filepath.Join(wpRoot, domain, "public_html"),
		filepath.Join(wpRoot, domain),
	}
	for _, c := range candidates {
		if _, err := os.Stat(filepath.Join(c, "wp-config.php")); err == nil {
			return c
		}
	}
	return ""
}

func probeWPSite(domain, wpConfigPath string) WPSite {
	docroot := filepath.Dir(wpConfigPath)
	site := WPSite{
		Domain: domain,
		Path:   docroot,
		Active: true,
	}

	// Read DB name from wp-config.php
	data, err := os.ReadFile(wpConfigPath)
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, "DB_NAME") {
				parts := strings.Split(line, "'")
				if len(parts) >= 4 {
					site.DBName = parts[3]
				}
			}
			if strings.Contains(line, "DB_USER") {
				parts := strings.Split(line, "'")
				if len(parts) >= 4 {
					site.DBUser = parts[3]
				}
			}
		}
	}

	// Check SSL via nginx config
	nginxConf := filepath.Join(nginxSitesEnabled, domain+".conf")
	if confData, err := os.ReadFile(nginxConf); err == nil {
		site.SSL = strings.Contains(string(confData), "ssl_certificate")
	}

	// Get WP version from wp-includes/version.php
	versionFile := filepath.Join(docroot, "wp-includes", "version.php")
	if vData, err := os.ReadFile(versionFile); err == nil {
		for _, line := range strings.Split(string(vData), "\n") {
			if strings.Contains(line, "$wp_version") && strings.Contains(line, "=") {
				parts := strings.Split(line, "'")
				if len(parts) >= 2 {
					site.WPVersion = parts[1]
				}
				break
			}
		}
	}

	return site
}

func buildWPNginxConfig(domain, docroot, phpVersion string) string {
	phpSocket := fmt.Sprintf("/run/php/php%s-fpm.sock", phpVersion)
	return fmt.Sprintf(`server {
    listen 80;
    listen [::]:80;

    server_name %s www.%s;
    root %s;
    index index.php index.html;

    access_log /var/log/nginx/%s.access.log;
    error_log  /var/log/nginx/%s.error.log;

    client_max_body_size 64M;

    location / {
        try_files $uri $uri/ /index.php?$args;
    }

    location ~ \.php$ {
        include snippets/fastcgi-php.conf;
        fastcgi_pass unix:%s;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        fastcgi_read_timeout 300;
    }

    # WordPress security rules
    location ~* /(?:uploads|files)/.*\.php$ { deny all; }
    location ~ /\. { deny all; }
    location = /xmlrpc.php { deny all; }
    location ~* /wp-config.php { deny all; }

    # Cache static assets
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 30d;
        add_header Cache-Control "public, no-transform";
    }

    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
}
`, domain, domain, docroot, domain, domain, phpSocket)
}

func randomPass(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[i%len(chars)]
	}
	return string(b)
}
