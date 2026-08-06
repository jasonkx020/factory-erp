package biz

import (
	"strings"

	"github.com/gin-gonic/gin"

	"erp/internal/api"
)

// handleIdCardOCR is a pluggable stub. When OCR is disabled, returns OCR_NOT_CONFIGURED
// so clients can fall back to manual entry.
func (s *Services) handleIdCardOCR(c *gin.Context) bool {
	if !s.OCREnabled {
		api.FailJSON(c, "OCR_NOT_CONFIGURED")
		return true
	}
	// Accept multipart for future providers; currently no real SDK.
	fh, err := c.FormFile("file")
	if err != nil {
		fh, err = c.FormFile("image")
	}
	_ = fh
	if err != nil && c.ContentType() != "" && !strings.Contains(c.ContentType(), "multipart") {
		api.FailJSON(c, "INVALID_REQUEST")
		return true
	}
	provider := s.OCRProvider
	if provider == "" {
		provider = "stub"
	}
	api.FailJSON(c, "OCR_PROVIDER_NOT_IMPLEMENTED:"+provider)
	return true
}
