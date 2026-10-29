package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/Maluslock/comic/server/internal/repository"
)

// Credit level thresholds. >=90 good, >=70 fair, otherwise poor. Kept here (not
// in the handler) so the C-end card, admin console and future exports agree.
const (
	creditLevelGoodThreshold = 90
	creditLevelFairThreshold = 70

	CreditLevelGood = "good"
	CreditLevelFair = "fair"
	CreditLevelPoor = "poor"

	// A user with no credit_scores row starts at the default score; returning
	// this instead of 404 keeps the C-end card usable for brand new accounts.
	creditDefaultScore int32 = 100
)

// CreditSummary is the payload behind GET /v1/me/credit.
type CreditSummary struct {
	Score            int32  `json:"score"`
	Level            string `json:"level"`
	SuccessCount     int32  `json:"successCount"`
	CancelCount      int32  `json:"cancelCount"`
	NoShowCount      int32  `json:"noShowCount"`
	DisputeLostCount int32  `json:"disputeLostCount"`
	UpdatedAt        string `json:"updatedAt"`
}

// CreditEventItem is one row of the credit timeline (camelCase for the app).
type CreditEventItem struct {
	ID          int64   `json:"id"`
	Delta       int32   `json:"delta"`
	Reason      string  `json:"reason"`
	RefType     *string `json:"refType"`
	RefID       *int64  `json:"refId"`
	BookingID   *int64  `json:"bookingId"`
	Description *string `json:"description"`
	CreatedAt   string  `json:"createdAt"`
}

// CreditEventList is the paginated timeline payload.
type CreditEventList struct {
	List  []CreditEventItem `json:"list"`
	Total int64             `json:"total"`
}

// CreditService serves read-only credit data for the C-end (and reuses the same
// scoring helpers the booking lifecycle writes with).
type CreditService struct {
	queries *repository.Queries
}

func NewCreditService(queries *repository.Queries) *CreditService {
	return &CreditService{queries: queries}
}

// CreditLevel maps a score to its bucket. Unknown/zero scores fall to poor only
// when they are genuinely low; the default 100 is good.
func CreditLevel(score int32) string {
	switch {
	case score >= creditLevelGoodThreshold:
		return CreditLevelGood
	case score >= creditLevelFairThreshold:
		return CreditLevelFair
	default:
		return CreditLevelPoor
	}
}

// Summary returns the caller's score plus the tallies derived from credit_events.
// Missing rows are not an error: a fresh account reads 100/good/0/0/0/0.
func (s *CreditService) Summary(ctx context.Context, userID int64) (CreditSummary, error) {
	out := CreditSummary{Score: creditDefaultScore, Level: CreditLevelGood}

	score, err := s.queries.GetCreditScoreByUser(ctx, userID)
	switch {
	case err == nil:
		out.Score = score.Score
		out.Level = CreditLevel(score.Score)
		out.UpdatedAt = score.UpdatedAt.Format("2006-01-02T15:04:05Z07:00")
	case errors.Is(err, pgx.ErrNoRows):
		// keep defaults
	default:
		return out, err
	}

	counts, err := s.queries.GetCreditCounts(ctx, userID)
	if err != nil {
		return out, err
	}
	out.SuccessCount = counts.SuccessCount
	out.CancelCount = counts.CancelCount
	out.NoShowCount = counts.NoShowCount
	out.DisputeLostCount = counts.DisputeLostCount
	return out, nil
}

// Events returns one page of the caller's credit history, newest first.
func (s *CreditService) Events(ctx context.Context, userID int64, limit, offset int32) (CreditEventList, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := s.queries.ListCreditEventsByUser(ctx, userID, limit, offset)
	if err != nil {
		return CreditEventList{}, err
	}
	total, err := s.queries.CountCreditEventsByUser(ctx, userID)
	if err != nil {
		return CreditEventList{}, err
	}

	items := make([]CreditEventItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, CreditEventItem{
			ID:          r.ID,
			Delta:       r.Delta,
			Reason:      r.Reason,
			RefType:     r.RefType,
			RefID:       r.RefID,
			BookingID:   r.BookingID,
			Description: r.Description,
			CreatedAt:   r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return CreditEventList{List: items, Total: total}, nil
}
