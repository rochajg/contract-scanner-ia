package handler

import (
	"net/http"
	"os"

	"contract-scanner/internal/middleware"
	"contract-scanner/internal/usecase"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct {
	uploadPDF usecase.IUploadPDF
}

func NewUploadHandler(uploadPDF usecase.IUploadPDF) *UploadHandler {
	return &UploadHandler{uploadPDF: uploadPDF}
}

// Upload receives a PDF via multipart/form-data (field "file"), uploads it to S3
// and creates an Analyse record. Returns the analysis_id to be used in /process.
func (h *UploadHandler) Upload(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "field 'file' is required (multipart/form-data)"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file"})
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/pdf"
	}

	output, err := h.uploadPDF.Execute(c.Request.Context(), usecase.UploadPDFInput{
		Filename:    fileHeader.Filename,
		ContentType: contentType,
		SizeBytes:   fileHeader.Size,
		FileBody:    file,
		ClerkUserID: middleware.GetUserID(c),
		Model:       os.Getenv("LLM_MODEL"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, output)
}
