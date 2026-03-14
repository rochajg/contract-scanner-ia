package usecase

import (
	"context"
	"encoding/json"
	"time"

	"contract-scanner/internal/usecase/repositories"
)

type IListAnalyses interface {
	Execute(ctx context.Context, userID string) ([]AnalyseSummary, error)
}

type AnalyseSummary struct {
	ID          string          `json:"id"`
	Filename    *string         `json:"filename"`
	Status      string          `json:"status"`
	CreatedAt   time.Time       `json:"created_at"`
	CompletedAt *time.Time      `json:"completed_at,omitempty"`
	Result      json.RawMessage `json:"result,omitempty"`
}

type ListAnalyses struct {
	AnalyseRepo repositories.AnalyseRepo
}

func NewListAnalyses(analyseRepo repositories.AnalyseRepo) IListAnalyses {
	return &ListAnalyses{AnalyseRepo: analyseRepo}
}

func (uc *ListAnalyses) Execute(_ context.Context, userID string) ([]AnalyseSummary, error) {
	analyses, err := uc.AnalyseRepo.ListByUser(userID, 50)
	if err != nil {
		return nil, err
	}

	result := make([]AnalyseSummary, len(analyses))
	for i, a := range analyses {
		result[i] = AnalyseSummary{
			ID:          a.ID.String(),
			Filename:    a.Filename,
			Status:      a.Status,
			CreatedAt:   a.CreatedAt,
			CompletedAt: a.CompletedAt,
			Result:      json.RawMessage(a.ResultJSON),
		}
	}
	return result, nil
}
