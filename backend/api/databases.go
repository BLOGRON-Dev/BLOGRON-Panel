package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"blogron/util"
)

// MySQL credentials are read from environment variables.
// Set MYSQL_USER and MYSQL_PASSWORD in your .env or systemd service file.
func mysqlArgs(extraArgs ...string) []string {
	user := os.Getenv("MYSQL_USER")
	if user == "" {
		user = "root"
	}
	pass := os.Getenv("MYSQL_PASSWORD")

	base := []string{"-u" + user}
	if pass != "" {
		base = append(base, "-p"+pass)
	}
	return append(base, extraArgs...)
}

type Database struct {
	Name   string `json:"name"`
	Size   string `json:"size"`
	Tables int    `json:"tables"`
}

type createDatabaseRequest struct {
	Name     string `json:"name"`
	DBUser   string `json:"db_user"`
	Password string `json:"password"`
	Host     string `json:"host"`
}

// ListDatabases godoc
// GET /api/databases
func ListDatabases(w http.ResponseWriter, r *http.Request) {
	out, err := util.RunCmd("mysql", append(mysqlArgs("-e", "SHOW DATABASES;", "--skip-column-names", "-s"))...)
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, "mysql query failed: "+err.Error())
		return
	}

	skipSystem := map[string]bool{
		"information_schema": true,
		"performance_schema": true,
		"mysql":              true,
		"sys":                true,
	}

	var dbs []Database
	for _, line := range strings.Split(out, "\n") {
		name := strings.TrimSpace(line)
		if name == "" || skipSystem[name] {
			continue
		}
		db := Database{Name: name}
		db.Size = getDatabaseSize(name)
		db.Tables = getDatabaseTableCount(name)
		dbs = append(dbs, db)
	}
	util.WriteJSON(w, http.StatusOK, dbs)
}

// CreateDatabase godoc
// POST /api/databases
func CreateDatabase(w http.ResponseWriter, r *http.Request) {
	var req createDatabaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	dbName := util.Sanitize(req.Name)
	dbUser := util.Sanitize(req.DBUser)
	if dbName == "" {
		util.WriteError(w, http.StatusBadRequest, "invalid database name")
		return
	}

	host := req.Host
	if host == "" {
		host = "localhost"
	}
	host = util.Sanitize(host)

	// Create database
	if _, err := util.RunCmd("mysql", append(mysqlArgs("-e", "CREATE DATABASE `"+dbName+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"))...); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "failed to create database: "+err.Error())
		return
	}

	// Create user and grant privileges if requested
	if dbUser != "" && req.Password != "" {
		grantSQL := "CREATE USER '" + dbUser + "'@'" + host + "' IDENTIFIED BY '" + req.Password + "'; " +
			"GRANT ALL PRIVILEGES ON `" + dbName + "`.* TO '" + dbUser + "'@'" + host + "'; " +
			"FLUSH PRIVILEGES;"
		util.RunCmd("mysql", append(mysqlArgs("-e", grantSQL))...)
	}

	util.WriteJSON(w, http.StatusCreated, map[string]string{
		"status":   "created",
		"database": dbName,
		"user":     dbUser,
	})
}

// DropDatabase godoc
// DELETE /api/databases/{name}
func DropDatabase(w http.ResponseWriter, r *http.Request) {
	name := util.Sanitize(chi_urlParam(r, "name"))
	if name == "" {
		util.WriteError(w, http.StatusBadRequest, "invalid database name")
		return
	}

	// Safety: refuse to drop system databases
	protected := map[string]bool{"mysql": true, "information_schema": true, "performance_schema": true, "sys": true}
	if protected[name] {
		util.WriteError(w, http.StatusForbidden, "cannot drop system database")
		return
	}

	if _, err := util.RunCmd("mysql", append(mysqlArgs("-e", "DROP DATABASE `"+name+"`;"))...); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "failed to drop database: "+err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "dropped", "database": name})
}

// ListTables godoc
// GET /api/databases/{name}/tables
func ListTables(w http.ResponseWriter, r *http.Request) {
	name := util.Sanitize(chi_urlParam(r, "name"))
	if name == "" {
		util.WriteError(w, http.StatusBadRequest, "invalid database name")
		return
	}

	out, err := util.RunCmd("mysql", append(mysqlArgs(name, "-e", "SHOW TABLES;", "--skip-column-names", "-s"))...)
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var tables []string
	for _, line := range strings.Split(out, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			tables = append(tables, t)
		}
	}
	util.WriteJSON(w, http.StatusOK, map[string]interface{}{"database": name, "tables": tables})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func getDatabaseSize(name string) string {
	query := `SELECT ROUND(SUM(data_length + index_length) / 1024 / 1024, 1) AS 'MB'
              FROM information_schema.tables WHERE table_schema = '` + name + `';`
	out, err := util.RunCmd("mysql", append(mysqlArgs("-e", query, "--skip-column-names", "-s"))...)
	if err != nil || strings.TrimSpace(out) == "NULL" {
		return "0 MB"
	}
	return strings.TrimSpace(out) + " MB"
}

func getDatabaseTableCount(name string) int {
	out, err := util.RunCmd("mysql", append(mysqlArgs(name, "-e", "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '"+name+"';", "--skip-column-names", "-s"))...)
	if err != nil {
		return 0
	}
	count := 0
	for _, c := range strings.TrimSpace(out) {
		count = count*10 + int(c-'0')
	}
	return count
}
