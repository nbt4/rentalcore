package handlers

import (
	"net/http"

	"go-barcode-webapp/internal/models"

	"github.com/gin-gonic/gin"
)

func extractionProcessingResponse(upload models.PDFUpload) (int, gin.H, bool) {
	switch upload.ProcessingStatus {
	case "pending", "processing":
		return http.StatusAccepted, gin.H{
			"processing_status": upload.ProcessingStatus,
		}, true
	case "failed":
		return http.StatusUnprocessableEntity, gin.H{
			"error":             "PDF-Verarbeitung fehlgeschlagen. Bitte die Datei erneut hochladen.",
			"processing_status": upload.ProcessingStatus,
		}, true
	default:
		return 0, nil, false
	}
}
