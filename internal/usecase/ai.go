package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/skiphead/go-letopis/internal/domain/entity"
	"github.com/skiphead/go-letopis/internal/domain/repository"
	gigachatservice "github.com/skiphead/go-letopis/internal/services/gigachat"
	"github.com/skiphead/go-letopis/internal/services/salutespeech/salute"
)

// AIUseCase defines the interface for AI-related operations.
type AIUseCase interface {
	Recognition(ctx context.Context, media *entity.Media) (string, error)
	Chat(ctx context.Context, text string) (string, error)
}

// aiUseCase implements AIUseCase using Salute Speech and GigaChat services.
type aiUseCase struct {
	userRepo           repository.UserRepository
	meetingRepo        repository.MeetingRepository
	logger             *slog.Logger
	saluteSpeechClient salute.Client
	gigaChatClient     gigachatservice.Client
}

// NewAIUseCase creates a new AIUseCase instance.
func NewAIUseCase(
	userRepo repository.UserRepository,
	meetingRepo repository.MeetingRepository,
	saluteSpeechClient salute.Client,
	gigaChatClient gigachatservice.Client,
	logger *slog.Logger,
) AIUseCase {
	return &aiUseCase{
		userRepo:           userRepo,
		meetingRepo:        meetingRepo,
		logger:             logger,
		saluteSpeechClient: saluteSpeechClient,
		gigaChatClient:     gigaChatClient,
	}
}

// Recognition processes audio recognition and generates a summary.
func (uc *aiUseCase) Recognition(ctx context.Context, media *entity.Media) (string, error) {
	asyncReq, err := uc.saluteSpeechClient.Upload(ctx, media.FilePath)
	if err != nil {
		return "", fmt.Errorf("upload speech video: %w", err)
	}

	resp, err := uc.saluteSpeechClient.CreateTask(ctx, asyncReq)
	if err != nil {
		return "", fmt.Errorf("create task: %w", err)
	}

	waitResult, err := uc.saluteSpeechClient.WaitForResult(ctx, resp.Result.ID)
	if err != nil {
		return "", fmt.Errorf("wait result: %w", err)
	}

	textExtract, err := uc.saluteSpeechClient.ExtractText(ctx, waitResult.Result.ResponseFileId)
	if err != nil {
		return "", fmt.Errorf("extract text: %w", err)
	}

	year, month, day := time.Now().Date()
	sysContent := fmt.Sprintf("сегодня дата %d-%d-%d, %s", year, month, day, chat)

	summary, err := uc.gigaChatClient.Completion(ctx, sysContent, textExtract)
	if err != nil {
		return "", fmt.Errorf("completion: %w", err)
	}

	var summaries []string
	for _, choice := range summary.Choices {
		summaries = append(summaries, choice.Message.Content)
	}

	user, err := uc.userRepo.Get(ctx, media.UserID)
	if err != nil {
		return "", fmt.Errorf("get user: %w", err)
	}

	err = uc.meetingRepo.Create(ctx, &entity.Meeting{
		UserID:          user.ID,
		Title:           media.FileName,
		Transcription:   textExtract,
		Summary:         strings.Join(summaries, " "),
		AudioFileID:     media.FileID,
		DurationSeconds: media.Duration,
	})
	if err != nil {
		return "", err
	}

	return strings.Join(summaries, " "), nil
}

// Chat handles chat interactions with the AI.
func (uc *aiUseCase) Chat(ctx context.Context, text string) (string, error) {
	response, err := uc.gigaChatClient.Completion(ctx, chat, text)
	if err != nil {
		return "", fmt.Errorf("completion: %w", err)
	}

	var responses []string
	for _, choice := range response.Choices {
		responses = append(responses, choice.Message.Content)
	}

	return strings.Join(responses, " "), nil
}
