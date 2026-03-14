package usecase

import (
	"context"
	"fmt"
	"log"

	"contract-scanner/internal/usecase/providers"
	"contract-scanner/internal/usecase/repositories"

	"github.com/google/uuid"
)

type IDeleteAnalysis interface {
	Execute(ctx context.Context, id uuid.UUID, userID string) error
}

type DeleteAnalysis struct {
	AnalyseRepo repositories.AnalyseRepo
	Storage     providers.StorageProvider
}

func NewDeleteAnalysis(analyseRepo repositories.AnalyseRepo, storage providers.StorageProvider) IDeleteAnalysis {
	return &DeleteAnalysis{AnalyseRepo: analyseRepo, Storage: storage}
}

func (uc *DeleteAnalysis) Execute(ctx context.Context, id uuid.UUID, userID string) error {
	analyse, err := uc.AnalyseRepo.Get(id)
	if err != nil {
		return fmt.Errorf("not found")
	}
	if analyse.ClerkUserID != userID {
		return fmt.Errorf("not found")
	}

	// Delete S3 objects before removing the DB record so that a failure here
	// aborts the operation and leaves the DB record intact (retryable).
	// S3 DeleteObject returns 204 even for missing objects, so a retry after
	// a partial failure is always safe.
	if analyse.S3Key != "" {
		if err := uc.Storage.DeleteObject(ctx, analyse.S3Key); err != nil {
			log.Printf("[delete] failed to delete S3 object key=%s: %v", analyse.S3Key, err)
			return fmt.Errorf("failed to delete file from storage: %w", err)
		}
	}
	if analyse.ExtractedTextS3Key != nil && *analyse.ExtractedTextS3Key != "" {
		if err := uc.Storage.DeleteObject(ctx, *analyse.ExtractedTextS3Key); err != nil {
			log.Printf("[delete] failed to delete S3 extracted text key=%s: %v", *analyse.ExtractedTextS3Key, err)
			return fmt.Errorf("failed to delete cached text from storage: %w", err)
		}
	}

	return uc.AnalyseRepo.Delete(id)
}
