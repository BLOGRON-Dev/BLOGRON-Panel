package api

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"blogron/util"
)

// Postfix virtual mailbox paths — adjust to your Postfix config
const (
	postfixVirtualMailboxDir  = "/etc/postfix/virtual_mailbox"
	postfixVirtualDomainsFile = "/etc/postfix/virtual_mailbox_domains"
	postfixVirtualMapsFile    = "/etc/postfix/virtual_mailbox_maps"
	dovecotPasswdFile         = "/etc/dovecot/users"
	mailStorageBase           = "/var/mail/vhosts"
)

type MailDomain struct {
	Domain    string `json:"domain"`
	Mailboxes int    `json:"mailboxes"`
	Active    bool   `json:"active"`
}

type Mailbox struct {
	Email  string `json:"email"`
	Domain string `json:"domain"`
	User   string `json:"user"`
	Quota  string `json:"quota"`
	Active bool   `json:"active"`
}

// ListMailDomains godoc
// GET /api/email/domains
func ListMailDomains(w http.ResponseWriter, r *http.Request) {
	domains := readMailDomains()
	util.WriteJSON(w, http.StatusOK, domains)
}

// AddMailDomain godoc
// POST /api/email/domains
func AddMailDomain(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	domain := util.Sanitize(body.Domain)
	if domain == "" {
		util.WriteError(w, http.StatusBadRequest, "invalid domain")
		return
	}

	// Add to virtual_mailbox_domains
	if err := appendLine(postfixVirtualDomainsFile, domain); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "failed to add domain: "+err.Error())
		return
	}

	// Create mail storage directory
	mailDir := filepath.Join(mailStorageBase, domain)
	util.RunCmd("mkdir", "-p", mailDir)
	util.RunCmd("chown", "-R", "vmail:vmail", mailDir)

	reloadPostfix()
	util.WriteJSON(w, http.StatusCreated, map[string]string{"status": "created", "domain": domain})
}

// DeleteMailDomain godoc
// DELETE /api/email/domains/{domain}
func DeleteMailDomain(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(chi_urlParam(r, "domain"))
	if domain == "" {
		util.WriteError(w, http.StatusBadRequest, "invalid domain")
		return
	}
	removeLine(postfixVirtualDomainsFile, domain)
	reloadPostfix()
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ListMailboxes godoc
// GET /api/email/mailboxes?domain=example.com
func ListMailboxes(w http.ResponseWriter, r *http.Request) {
	domain := util.Sanitize(r.URL.Query().Get("domain"))
	mailboxes := readMailboxes(domain)
	util.WriteJSON(w, http.StatusOK, mailboxes)
}

// CreateMailbox godoc
// POST /api/email/mailboxes
func CreateMailbox(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Quota    string `json:"quota"` // e.g. "1G", "500M"
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	if !strings.Contains(body.Email, "@") {
		util.WriteError(w, http.StatusBadRequest, "invalid email address")
		return
	}
	if len(body.Password) < 8 {
		util.WriteError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}

	parts := strings.SplitN(body.Email, "@", 2)
	user := util.Sanitize(parts[0])
	domain := util.Sanitize(parts[1])
	email := user + "@" + domain

	quota := body.Quota
	if quota == "" {
		quota = "1G"
	}

	// Add to virtual_mailbox_maps: email -> domain/user/
	mapLine := fmt.Sprintf("%s %s/%s/", email, domain, user)
	if err := appendLine(postfixVirtualMapsFile, mapLine); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "failed to add mailbox map")
		return
	}

	// Create mail directory
	mailDir := filepath.Join(mailStorageBase, domain, user)
	for _, sub := range []string{"", "/cur", "/new", "/tmp"} {
		util.RunCmd("mkdir", "-p", mailDir+sub)
	}
	util.RunCmd("chown", "-R", "vmail:vmail", filepath.Join(mailStorageBase, domain))

	// Add Dovecot passwd entry (SHA512-CRYPT hash)
	passwdLine := fmt.Sprintf("%s:{PLAIN}%s:::::userdb_quota_rule=*:storage=%s", email, body.Password, quota)
	if err := appendLine(dovecotPasswdFile, passwdLine); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "failed to add dovecot user")
		return
	}

	// Rebuild postfix maps
	util.RunCmd("postmap", postfixVirtualMapsFile)
	reloadPostfix()

	util.WriteJSON(w, http.StatusCreated, map[string]string{
		"status": "created",
		"email":  email,
		"quota":  quota,
	})
}

// DeleteMailbox godoc
// DELETE /api/email/mailboxes/{email}
func DeleteMailbox(w http.ResponseWriter, r *http.Request) {
	email := chi_urlParam(r, "email")
	if !strings.Contains(email, "@") {
		util.WriteError(w, http.StatusBadRequest, "invalid email")
		return
	}

	parts := strings.SplitN(email, "@", 2)
	user := util.Sanitize(parts[0])
	domain := util.Sanitize(parts[1])
	email = user + "@" + domain

	// Remove from virtual map
	removeLine(postfixVirtualMapsFile, email)
	// Remove from dovecot passwd
	removeLine(dovecotPasswdFile, email)

	util.RunCmd("postmap", postfixVirtualMapsFile)
	reloadPostfix()
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// GetMailQueue godoc
// GET /api/email/queue
func GetMailQueue(w http.ResponseWriter, r *http.Request) {
	out, err := util.RunCmd("postqueue", "-p")
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"queue": out})
}

// FlushMailQueue godoc
// POST /api/email/queue/flush
func FlushMailQueue(w http.ResponseWriter, r *http.Request) {
	util.RunCmd("postqueue", "-f")
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "flushed"})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func readMailDomains() []MailDomain {
	var domains []MailDomain
	f, err := os.Open(postfixVirtualDomainsFile)
	if err != nil {
		return domains
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		mailboxes := countMailboxesForDomain(line)
		domains = append(domains, MailDomain{
			Domain:    line,
			Mailboxes: mailboxes,
			Active:    true,
		})
	}
	return domains
}

func readMailboxes(filterDomain string) []Mailbox {
	var mailboxes []Mailbox
	f, err := os.Open(postfixVirtualMapsFile)
	if err != nil {
		return mailboxes
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 1 {
			continue
		}
		email := parts[0]
		if !strings.Contains(email, "@") {
			continue
		}
		ep := strings.SplitN(email, "@", 2)
		user, domain := ep[0], ep[1]
		if filterDomain != "" && domain != filterDomain {
			continue
		}
		mailboxes = append(mailboxes, Mailbox{
			Email:  email,
			User:   user,
			Domain: domain,
			Active: true,
		})
	}
	return mailboxes
}

func countMailboxesForDomain(domain string) int {
	count := 0
	f, err := os.Open(postfixVirtualMapsFile)
	if err != nil {
		return 0
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "@"+domain) {
			count++
		}
	}
	return count
}

func reloadPostfix() {
	util.RunCmd("systemctl", "reload", "postfix")
}

func appendLine(file, line string) error {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, line)
	return err
}

func removeLine(file, prefix string) {
	data, err := os.ReadFile(file)
	if err != nil {
		return
	}
	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, prefix) {
			kept = append(kept, line)
		}
	}
	os.WriteFile(file, []byte(strings.Join(kept, "\n")), 0644)
}
