package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/skiphead/go-letopis/internal/domain/entity"
	"github.com/skiphead/go-letopis/internal/domain/repository"
)

// MeetingUseCase defines the interface for meeting-related operations.
type MeetingUseCase interface {
	List(ctx context.Context, userID int64) ([]entity.Meeting, error)
	Get(ctx context.Context, messageID, telegramID int64) (*entity.Meeting, error)
	SearchByKeywords(ctx context.Context, req entity.SearchRequest) ([]entity.TranscriptionRecord, error)
}

// meetingUseCase implements MeetingUseCase.
type meetingUseCase struct {
	meetingRepo repository.MeetingRepository
	logger      *slog.Logger
}

// NewMeetingUseCase creates a new MeetingUseCase instance.
func NewMeetingUseCase(meetingRepo repository.MeetingRepository, logger *slog.Logger) MeetingUseCase {
	return &meetingUseCase{
		meetingRepo: meetingRepo,
		logger:      logger,
	}
}

// List returns all meetings for a given user.
func (uc *meetingUseCase) List(ctx context.Context, userID int64) ([]entity.Meeting, error) {
	if userID == 0 {
		return nil, fmt.Errorf("user_id is required")
	}

	list, err := uc.meetingRepo.List(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return list, nil
		}
		return nil, err
	}

	return list, nil
}

// Get returns a specific meeting by ID and Telegram ID.
func (uc *meetingUseCase) Get(ctx context.Context, messageID, telegramID int64) (*entity.Meeting, error) {
	if messageID == 0 {
		return nil, fmt.Errorf("message_id is required")
	}

	meeting, err := uc.meetingRepo.Get(ctx, messageID, telegramID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return meeting, nil
		}
		return nil, err
	}

	return meeting, nil
}

// SearchByKeywords performs a keyword search on meetings.
func (uc *meetingUseCase) SearchByKeywords(ctx context.Context, req entity.SearchRequest) ([]entity.TranscriptionRecord, error) {
	return uc.meetingRepo.SearchByKeywords(ctx, req)
}
