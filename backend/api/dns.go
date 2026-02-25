package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"blogron/util"
)

const (
	bindZonesDir   = "/etc/bind/zones"
	bindNamedLocal = "/etc/bind/named.conf.local"
)

// bindService returns the correct systemd unit name for BIND9.
// Ubuntu uses 'named'; Debian may use 'bind9'. The installer writes
// the detected name into the BIND_SERVICE environment variable.
func bindService() string {
	if svc := os.Getenv("BIND_SERVICE"); svc != "" {
		return svc
	}
	return "named" // safe default — works on Ubuntu 22.04/24.04
}

type DNSZone struct {
	Domain  string      `json:"domain"`
	Serial  int64       `json:"serial"`
	Records []DNSRecord `json:"records"`
}

type DNSRecord struct {
	Name  string `json:"name"`
	TTL   string `json:"ttl"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// ListDNSZones godoc
// GET /api/dns
func ListDNSZones(w http.ResponseWriter, r *http.Request) {
	entries, err := os.ReadDir(bindZonesDir)
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, "cannot read zones directory")
		return
	}

	var zones []map[string]string
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".db") {
			continue
		}
		domain := strings.TrimSuffix(e.Name(), ".db")
		zones = append(zones, map[string]string{"domain": domain})
	}
	util.WriteJSON(w, http.StatusOK, zones)
}

// GetDNSZone godoc
// GET /api/dns/{domain}
func GetDNSZone(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	if domain == "" {
		util.WriteError(w, http.StatusBadRequest, "invalid domain")
		return
	}

	zone, err := parseZoneFile(domain)
	if err != nil {
		util.WriteError(w, http.StatusNotFound, "zone not found: "+err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, zone)
}

// CreateDNSZone godoc
// POST /api/dns
func CreateDNSZone(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Domain    string `json:"domain"`
		IPAddress string `json:"ip"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	domain := util.Sanitize(body.Domain)
	ip := body.IPAddress
	if domain == "" || ip == "" {
		util.WriteError(w, http.StatusBadRequest, "domain and ip are required")
		return
	}

	serial := time.Now().Format("2006010215")
	zoneContent := buildZoneFile(domain, ip, serial)
	zoneFile := filepath.Join(bindZonesDir, domain+".db")

	if err := os.MkdirAll(bindZonesDir, 0755); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "cannot create zones dir")
		return
	}

	if err := os.WriteFile(zoneFile, []byte(zoneContent), 0644); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "failed to write zone file: "+err.Error())
		return
	}

	// Add zone to named.conf.local
	if err := addZoneToNamedConf(domain, zoneFile); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "failed to update named.conf.local: "+err.Error())
		return
	}

	// Reload BIND
	if _, err := util.RunCmd("systemctl", "reload", bindService()); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "DNS service reload failed: "+err.Error())
		return
	}

	util.WriteJSON(w, http.StatusCreated, map[string]string{"status": "created", "domain": domain})
}

// DeleteDNSZone godoc
// DELETE /api/dns/{domain}
func DeleteDNSZone(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	if domain == "" {
		util.WriteError(w, http.StatusBadRequest, "invalid domain")
		return
	}

	zoneFile := filepath.Join(bindZonesDir, domain+".db")
	os.Remove(zoneFile)
	removeZoneFromNamedConf(domain)
	util.RunCmd("systemctl", "reload", bindService())
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// AddDNSRecord godoc
// POST /api/dns/{domain}/records
func AddDNSRecord(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	var rec DNSRecord
	if err := json.NewDecoder(r.Body).Decode(&rec); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	// Validate type
	allowedTypes := map[string]bool{"A": true, "AAAA": true, "CNAME": true, "MX": true, "TXT": true, "NS": true, "PTR": true, "SRV": true}
	rec.Type = strings.ToUpper(rec.Type)
	if !allowedTypes[rec.Type] {
		util.WriteError(w, http.StatusBadRequest, "unsupported record type")
		return
	}
	if rec.TTL == "" {
		rec.TTL = "3600"
	}

	zoneFile := filepath.Join(bindZonesDir, domain+".db")
	data, err := os.ReadFile(zoneFile)
	if err != nil {
		util.WriteError(w, http.StatusNotFound, "zone not found")
		return
	}

	// Append record
	newLine := fmt.Sprintf("%s\t%s\tIN\t%s\t%s", rec.Name, rec.TTL, rec.Type, rec.Value)
	updated := string(data) + "\n" + newLine + "\n"

	// Bump serial
	updated = bumpSerial(updated)

	if err := os.WriteFile(zoneFile, []byte(updated), 0644); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "failed to write zone file")
		return
	}

	util.RunCmd("systemctl", "reload", bindService())
	util.WriteJSON(w, http.StatusCreated, map[string]string{"status": "added"})
}

// DeleteDNSRecord godoc
// DELETE /api/dns/{domain}/records
// Body: { "name": "www", "type": "A" }
func DeleteDNSRecord(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	var body struct {
		Name  string `json:"name"`
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	zoneFile := filepath.Join(bindZonesDir, domain+".db")
	data, err := os.ReadFile(zoneFile)
	if err != nil {
		util.WriteError(w, http.StatusNotFound, "zone not found")
		return
	}

	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		if strings.Contains(line, body.Name) && strings.Contains(line, body.Type) {
			continue // remove matching record
		}
		kept = append(kept, line)
	}
	updated := bumpSerial(strings.Join(kept, "\n"))
	os.WriteFile(zoneFile, []byte(updated), 0644)
	util.RunCmd("systemctl", "reload", bindService())
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func buildZoneFile(domain, ip, serial string) string {
	return fmt.Sprintf(`$TTL 3600
@   IN  SOA ns1.%s. admin.%s. (
            %s  ; Serial
            3600        ; Refresh
            1800        ; Retry
            604800      ; Expire
            300 )       ; Minimum TTL

; Name Servers
@       IN  NS  ns1.%s.
@       IN  NS  ns2.%s.

; A Records
@       IN  A   %s
www     IN  A   %s
ns1     IN  A   %s
ns2     IN  A   %s

; Mail
@       IN  MX  10 mail.%s.
mail    IN  A   %s

; SPF
@       IN  TXT "v=spf1 mx a ip4:%s -all"
`, domain, domain, serial, domain, domain, ip, ip, ip, ip, domain, ip, ip)
}

func parseZoneFile(domain string) (DNSZone, error) {
	zoneFile := filepath.Join(bindZonesDir, domain+".db")
	data, err := os.ReadFile(zoneFile)
	if err != nil {
		return DNSZone{}, err
	}

	zone := DNSZone{Domain: domain}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "$") {
			continue
		}
		// Extract serial from SOA
		if strings.Contains(line, "Serial") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				zone.Serial, _ = strconv.ParseInt(parts[0], 10, 64)
			}
			continue
		}
		// Parse record lines: name ttl IN type value
		parts := strings.Fields(line)
		if len(parts) >= 4 && parts[2] == "IN" {
			rec := DNSRecord{
				Name:  parts[0],
				Type:  parts[3],
				Value: strings.Join(parts[4:], " "),
			}
			// TTL may or may not be present
			if _, err := strconv.Atoi(parts[1]); err == nil {
				rec.TTL = parts[1]
			}
			zone.Records = append(zone.Records, rec)
		}
	}
	return zone, nil
}

func addZoneToNamedConf(domain, zoneFile string) error {
	entry := fmt.Sprintf(`
zone "%s" {
    type master;
    file "%s";
};
`, domain, zoneFile)

	f, err := os.OpenFile(bindNamedLocal, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprint(f, entry)
	return err
}

func removeZoneFromNamedConf(domain string) {
	data, err := os.ReadFile(bindNamedLocal)
	if err != nil {
		return
	}
	// Remove the zone block
	content := string(data)
	marker := fmt.Sprintf(`zone "%s"`, domain)
	lines := strings.Split(content, "\n")
	var kept []string
	skip := false
	depth := 0
	for _, line := range lines {
		if strings.Contains(line, marker) {
			skip = true
		}
		if skip {
			depth += strings.Count(line, "{") - strings.Count(line, "}")
			if depth <= 0 && strings.Contains(line, "}") {
				skip = false
				depth = 0
				continue
			}
			continue
		}
		kept = append(kept, line)
	}
	os.WriteFile(bindNamedLocal, []byte(strings.Join(kept, "\n")), 0644)
}

func bumpSerial(zoneContent string) string {
	newSerial := time.Now().Format("2006010215")
	lines := strings.Split(zoneContent, "\n")
	for i, line := range lines {
		if strings.Contains(line, "Serial") || strings.Contains(line, "serial") {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				lines[i] = strings.Replace(line, parts[0], newSerial, 1)
			}
			break
		}
	}
	return strings.Join(lines, "\n")
}
