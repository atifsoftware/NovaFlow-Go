package core

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// UploadHandler validates and stores a single multipart file upload,
// mirroring NovaFlow PHP's UploadHandler (allowedTypes/maxSize/upload()).
type UploadHandler struct {
	allowedTypes map[string]bool
	maxSizeBytes int64
	destDir      string
}

func NewUploadHandler(destDir string) *UploadHandler {
	return &UploadHandler{
		allowedTypes: map[string]bool{},
		maxSizeBytes: 5 << 20, // 5MB default
		destDir:      destDir,
	}
}

func (u *UploadHandler) AllowedTypes(mimeTypes ...string) *UploadHandler {
	for _, t := range mimeTypes {
		u.allowedTypes[t] = true
	}
	return u
}

func (u *UploadHandler) MaxSize(bytes int64) *UploadHandler {
	u.maxSizeBytes = bytes
	return u
}

type UploadResult struct {
	FileName string
	Path     string
	Size     int64
	MimeType string
}

// Upload reads the named form file from the request, validates it, and
// saves it under destDir with a collision-safe timestamped name.
func (u *UploadHandler) Upload(r *http.Request, fieldName string) (*UploadResult, error) {
	if err := r.ParseMultipartForm(u.maxSizeBytes + (1 << 20)); err != nil {
		return nil, fmt.Errorf("parse form: %w", err)
	}
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return nil, fmt.Errorf("no file provided: %w", err)
	}
	defer file.Close()

	if header.Size > u.maxSizeBytes {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", header.Size, u.maxSizeBytes)
	}

	mimeType, err := detectMime(file)
	if err != nil {
		return nil, err
	}
	if len(u.allowedTypes) > 0 && !u.allowedTypes[mimeType] {
		return nil, fmt.Errorf("file type %s not allowed", mimeType)
	}

	if err := os.MkdirAll(u.destDir, 0o755); err != nil {
		return nil, err
	}
	safeName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), sanitizeFilename(header.Filename))
	destPath := filepath.Join(u.destDir, safeName)

	out, err := os.Create(destPath)
	if err != nil {
		return nil, err
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		return nil, err
	}

	return &UploadResult{FileName: safeName, Path: destPath, Size: header.Size, MimeType: mimeType}, nil
}

func detectMime(f multipart.File) (string, error) {
	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return "", err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return "", err
	}
	return http.DetectContentType(buf[:n]), nil
}

func sanitizeFilename(name string) string {
	name = strings.ReplaceAll(name, " ", "_")
	replacer := strings.NewReplacer("/", "_", "\\", "_", "..", "_")
	return replacer.Replace(name)
}
