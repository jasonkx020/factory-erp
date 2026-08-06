package biz

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"erp/internal/api"
)

func (s *Services) handleUploads(c *gin.Context, method string) bool {
	if method != "POST" {
		return false
	}
	fh, err := c.FormFile("file")
	if err != nil {
		fh, err = c.FormFile("image")
	}
	if err != nil {
		api.FailJSON(c, "FILE_REQUIRED")
		return true
	}
	src, err := fh.Open()
	if err != nil {
		api.FailJSON(c, "FILE_OPEN_FAILED")
		return true
	}
	defer src.Close()

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif", ".heic":
	default:
		ext = ".jpg"
	}
	day := time.Now().Format("20060102")
	dir := filepath.Join("data", "uploads", day)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		api.FailJSON(c, "MKDIR_FAILED:"+err.Error())
		return true
	}
	name := strings.ReplaceAll(uuid.NewString(), "-", "") + ext
	dstPath := filepath.Join(dir, name)
	dst, err := os.Create(dstPath)
	if err != nil {
		api.FailJSON(c, "CREATE_FAILED:"+err.Error())
		return true
	}
	defer dst.Close()
	if _, err := io.Copy(dst, src); err != nil {
		api.FailJSON(c, "WRITE_FAILED:"+err.Error())
		return true
	}
	url := fmt.Sprintf("/files/uploads/%s/%s", day, name)
	api.OK(c, gin.H{"url": url, "file_url": url, "size": fh.Size, "filename": fh.Filename})
	return true
}
