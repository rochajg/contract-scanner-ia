package usecase

import (
	"context"
	"fmt"
	"io"
	"time"

	"contract-scanner/internal/infra/database/postgres/models"
	"contract-scanner/internal/usecase/providers"
	"contract-scanner/internal/usecase/repositories"

	"github.com/google/uuid"
)

type IUploadPDF interface {
	Execute(ctx context.Context, input UploadPDFInput) (*UploadPDFOutput, error)
}

type UploadPDFInput struct {
	Filename    string
	ContentType string
	SizeBytes   int64
	FileBody    io.Reader
	ClerkUserID string
	Model       string
}

type UploadPDFOutput struct {
	AnalysisID string `json:"analysis_id"`
	S3Key      string `json:"s3_key"`
}

type UploadPDF struct {
	AnalyseRepo     repositories.AnalyseRepo
	StorageProvider providers.StorageProvider
}

func NewUploadPDF(analyseRepo repositories.AnalyseRepo, storageProvider providers.StorageProvider) IUploadPDF {
	return &UploadPDF{
		AnalyseRepo:     analyseRepo,
		StorageProvider: storageProvider,
	}
}

func (uc *UploadPDF) Execute(ctx context.Context, input UploadPDFInput) (*UploadPDFOutput, error) {
	analyseID := uuid.New()
	s3Key := fmt.Sprintf("contracts/%s.pdf", analyseID)

	contentType := input.ContentType
	if contentType == "" {
		contentType = "application/pdf"
	}

	if err := uc.StorageProvider.PutObject(ctx, s3Key, input.FileBody, contentType, input.SizeBytes); err != nil {
		return nil, fmt.Errorf("error uploading pdf to s3: %w", err)
	}

	model := input.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	analyse := models.Analyse{
		ID:          analyseID,
		ClerkUserID: input.ClerkUserID,
		Status:      "UPLOADED",
		Filename:    &input.Filename,
		ContentType: &contentType,
		SizeBytes:   &input.SizeBytes,
		S3Key:       s3Key,
		Model:       model,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := uc.AnalyseRepo.Create(analyse); err != nil {
		return nil, fmt.Errorf("error creating analyse record: %w", err)
	}

	return &UploadPDFOutput{
		AnalysisID: analyseID.String(),
		S3Key:      s3Key,
	}, nil
}
