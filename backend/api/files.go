package api

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"blogron/util"
)

// File manager is restricted to this base path for safety.
// Adjust for your environment — typical values: "/var/www", "/home"
const fileManagerRoot = "/var/www"

type FileEntry struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	IsDir       bool      `json:"is_dir"`
	Size        int64     `json:"size"`
	Permissions string    `json:"permissions"`
	Modified    time.Time `json:"modified"`
}

// safePath resolves the requested path within fileManagerRoot and ensures
// it doesn't escape via path traversal (e.g., "../../etc/passwd").
func safePath(requested string) (string, error) {
	if requested == "" {
		requested = "/"
	}
	// Resolve relative to root
	full := filepath.Join(fileManagerRoot, filepath.Clean("/"+requested))
	// Ensure the result is still within fileManagerRoot
	if !strings.HasPrefix(full, fileManagerRoot) {
		return "", os.ErrPermission
	}
	return full, nil
}

// ListFiles godoc
// GET /api/files?path=/some/dir
func ListFiles(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Query().Get("path")
	absPath, err := safePath(reqPath)
	if err != nil {
		util.WriteError(w, http.StatusForbidden, "path traversal detected")
		return
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, "cannot read directory: "+err.Error())
		return
	}

	var files []FileEntry
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		relPath := strings.TrimPrefix(absPath+"/"+e.Name(), fileManagerRoot)
		files = append(files, FileEntry{
			Name:        e.Name(),
			Path:        relPath,
			IsDir:       e.IsDir(),
			Size:        info.Size(),
			Permissions: info.Mode().String(),
			Modified:    info.ModTime(),
		})
	}
	util.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"path":  strings.TrimPrefix(absPath, fileManagerRoot),
		"files": files,
	})
}

// MakeDirectory godoc
// POST /api/files/mkdir
// Body: { "path": "/new-folder" }
func MakeDirectory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	absPath, err := safePath(body.Path)
	if err != nil {
		util.WriteError(w, http.StatusForbidden, "invalid path")
		return
	}

	if err := os.MkdirAll(absPath, 0755); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "mkdir failed: "+err.Error())
		return
	}
	util.WriteJSON(w, http.StatusCreated, map[string]string{"status": "created", "path": body.Path})
}

// DeleteFile godoc
// DELETE /api/files?path=/file-or-dir
func DeleteFile(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Query().Get("path")
	absPath, err := safePath(reqPath)
	if err != nil {
		util.WriteError(w, http.StatusForbidden, "invalid path")
		return
	}

	// Refuse to delete the root itself
	if absPath == fileManagerRoot {
		util.WriteError(w, http.StatusForbidden, "cannot delete root directory")
		return
	}

	if err := os.RemoveAll(absPath); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "delete failed: "+err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// RenameFile godoc
// POST /api/files/rename
// Body: { "from": "/old-name", "to": "/new-name" }
func RenameFile(w http.ResponseWriter, r *http.Request) {
	var body struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	src, err := safePath(body.From)
	if err != nil {
		util.WriteError(w, http.StatusForbidden, "invalid source path")
		return
	}
	dst, err := safePath(body.To)
	if err != nil {
		util.WriteError(w, http.StatusForbidden, "invalid destination path")
		return
	}

	if err := os.Rename(src, dst); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "rename failed: "+err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "renamed"})
}

// ReadFile godoc
// GET /api/files/read?path=/some/file.conf
func ReadFile(w http.ResponseWriter, r *http.Request) {
	reqPath := r.URL.Query().Get("path")
	absPath, err := safePath(reqPath)
	if err != nil {
		util.WriteError(w, http.StatusForbidden, "invalid path")
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		util.WriteError(w, http.StatusNotFound, "file not found")
		return
	}
	if info.IsDir() {
		util.WriteError(w, http.StatusBadRequest, "path is a directory")
		return
	}
	// Limit reads to 2 MB to prevent accidental large file reads
	if info.Size() > 2*1024*1024 {
		util.WriteError(w, http.StatusRequestEntityTooLarge, "file too large to read via API (max 2 MB)")
		return
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, "read failed: "+err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"path": reqPath, "content": string(data)})
}

// WriteFile godoc
// POST /api/files/write
// Body: { "path": "/some/file.conf", "content": "..." }
func WriteFile(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		util.WriteError(w, http.StatusBadRequest, "invalid body")
		return
	}

	absPath, err := safePath(body.Path)
	if err != nil {
		util.WriteError(w, http.StatusForbidden, "invalid path")
		return
	}

	if err := os.WriteFile(absPath, []byte(body.Content), 0644); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "write failed: "+err.Error())
		return
	}
	util.WriteJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

// UploadFile godoc
// POST /api/files/upload (multipart/form-data)
// Form fields: path (destination dir), file (the upload)
func UploadFile(w http.ResponseWriter, r *http.Request) {
	// Limit uploads to 100 MB
	r.Body = http.MaxBytesReader(w, r.Body, 100*1024*1024)

	if err := r.ParseMultipartForm(100 << 20); err != nil {
		util.WriteError(w, http.StatusBadRequest, "could not parse multipart form")
		return
	}

	destDir := r.FormValue("path")
	absDir, err := safePath(destDir)
	if err != nil {
		util.WriteError(w, http.StatusForbidden, "invalid destination path")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		util.WriteError(w, http.StatusBadRequest, "missing file field")
		return
	}
	defer file.Close()

	// Sanitize filename
	filename := filepath.Base(header.Filename)
	destPath := filepath.Join(absDir, filename)

	// Ensure dest is still within root
	if !strings.HasPrefix(destPath, fileManagerRoot) {
		util.WriteError(w, http.StatusForbidden, "invalid destination")
		return
	}

	out, err := os.Create(destPath)
	if err != nil {
		util.WriteError(w, http.StatusInternalServerError, "cannot create file: "+err.Error())
		return
	}
	defer out.Close()

	if _, err = io.Copy(out, file); err != nil {
		util.WriteError(w, http.StatusInternalServerError, "upload failed: "+err.Error())
		return
	}

	util.WriteJSON(w, http.StatusCreated, map[string]string{
		"status":   "uploaded",
		"filename": filename,
		"path":     strings.TrimPrefix(destPath, fileManagerRoot),
	})
}
