package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	maxUploadSize = 10 << 20 // 10MB
	uploadDir     = ".acpone-uploads"
)

// FileInfo represents a file in the workspace
type FileInfo struct {
	Path   string `json:"path"`   // Relative path from workspace root
	Name   string `json:"name"`   // File name
	IsDir  bool   `json:"isDir"`  // Is directory
}

// handleWorkspaceFiles returns files in the current workspace
func (s *Server) handleWorkspaceFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	workspaceID := r.URL.Query().Get("workspaceId")
	query := r.URL.Query().Get("q")
	limitStr := r.URL.Query().Get("limit")

	limit := 50 // Default limit
	if limitStr != "" {
		if n, err := parseInt(limitStr); err == nil && n > 0 {
			limit = n
		}
	}

	workspacePath := s.resolveWorkspacePath(workspaceID)
	if workspacePath == "" || workspacePath == "." {
		writeJSON(w, map[string]any{"files": []FileInfo{}})
		return
	}

	files := listWorkspaceFiles(workspacePath, query, limit)
	writeJSON(w, map[string]any{"files": files})
}

// listWorkspaceFiles walks the workspace and returns matching files
func listWorkspaceFiles(root, query string, limit int) []FileInfo {
	var files []FileInfo
	query = strings.ToLower(query)

	// Skip these directories
	skipDirs := map[string]bool{
		".git":         true,
		"node_modules": true,
		".idea":        true,
		".vscode":      true,
		"vendor":       true,
		"dist":         true,
		"build":        true,
		"__pycache__":  true,
		".next":        true,
		".nuxt":        true,
		"coverage":     true,
		".cache":       true,
	}

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Get relative path
		relPath, err := filepath.Rel(root, path)
		if err != nil || relPath == "." {
			return nil
		}

		// Skip hidden files/dirs (except query matches)
		name := info.Name()
		if strings.HasPrefix(name, ".") && !strings.Contains(strings.ToLower(name), query) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip certain directories
		if info.IsDir() {
			if skipDirs[name] {
				return filepath.SkipDir
			}
			return nil // Don't add directories to results
		}

		// Check limit
		if len(files) >= limit {
			return filepath.SkipAll
		}

		// Match query (case insensitive)
		if query != "" {
			lowerPath := strings.ToLower(relPath)
			lowerName := strings.ToLower(name)
			if !strings.Contains(lowerPath, query) && !strings.Contains(lowerName, query) {
				return nil
			}
		}

		// Use forward slashes for consistency
		relPath = filepath.ToSlash(relPath)

		files = append(files, FileInfo{
			Path:  relPath,
			Name:  name,
			IsDir: info.IsDir(),
		})

		return nil
	})

	return files
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, &pathError{msg: "invalid number"}
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// UploadedFile represents an uploaded file
type UploadedFile struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

func (s *Server) handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit request size
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		writeError(w, "File too large (max 10MB)", http.StatusBadRequest)
		return
	}

	// Get workspace ID from form
	workspaceID := r.FormValue("workspaceId")
	workspacePath := s.resolveWorkspacePath(workspaceID)

	// Create upload directory
	uploadPath := filepath.Join(workspacePath, uploadDir)
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		writeError(w, "Failed to create upload directory", http.StatusInternalServerError)
		return
	}

	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		writeError(w, "No files uploaded", http.StatusBadRequest)
		return
	}

	uploadedFiles := make([]UploadedFile, 0, len(files))

	for _, fileHeader := range files {
		// Open uploaded file
		file, err := fileHeader.Open()
		if err != nil {
			writeError(w, "Failed to read uploaded file", http.StatusInternalServerError)
			return
		}
		defer file.Close()

		// Generate unique filename to avoid conflicts
		ext := filepath.Ext(fileHeader.Filename)
		baseName := strings.TrimSuffix(fileHeader.Filename, ext)
		uniqueName := fmt.Sprintf("%s_%d%s", baseName, time.Now().UnixNano(), ext)
		destPath := filepath.Join(uploadPath, uniqueName)

		// Create destination file
		dst, err := os.Create(destPath)
		if err != nil {
			writeError(w, "Failed to save file", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		// Copy file content
		size, err := io.Copy(dst, file)
		if err != nil {
			writeError(w, "Failed to save file", http.StatusInternalServerError)
			return
		}

		uploadedFiles = append(uploadedFiles, UploadedFile{
			Name: fileHeader.Filename,
			Path: destPath,
			Size: size,
		})
	}

	writeJSON(w, map[string]any{
		"success": true,
		"files":   uploadedFiles,
	})
}

func (s *Server) handleFileCleanup(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		WorkspaceID string `json:"workspaceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		writeError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	workspacePath := s.resolveWorkspacePath(data.WorkspaceID)
	uploadPath := filepath.Join(workspacePath, uploadDir)

	// Remove upload directory and all contents
	if err := os.RemoveAll(uploadPath); err != nil && !os.IsNotExist(err) {
		writeError(w, "Failed to cleanup files", http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]any{"success": true})
}

// CleanupUploads removes the upload directory for a workspace
func (s *Server) CleanupUploads(workspaceID string) error {
	workspacePath := s.resolveWorkspacePath(workspaceID)
	uploadPath := filepath.Join(workspacePath, uploadDir)
	return os.RemoveAll(uploadPath)
}
