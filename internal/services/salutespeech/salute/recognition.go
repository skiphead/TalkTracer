package salute

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/recognition/async"
	"github.com/skiphead/salutespeech/types"
	"github.com/skiphead/salutespeech/upload"
	"github.com/skiphead/salutespeech/utils"
)

// Client defines the interface for Salute Speech operations.
type Client interface {
	Upload(ctx context.Context, pathAudioFile string) (*async.Request, error)
	CreateTask(ctx context.Context, req *async.Request) (*async.Response, error)
	WaitForResult(ctx context.Context, responseResultID string) (*async.TaskResult, error)
	ExtractText(ctx context.Context, fileID string) (string, error)
}

// clientImpl implements Client using the Salute Speech API.
type clientImpl struct {
	clientAsync  *async.Client
	clientUpload upload.Client
	logger       *slog.Logger
}

// NewClient creates a new Salute Speech client instance.
func NewClient(clientID, clientSecret string, logger *slog.Logger) (Client, error) {
	basicAuth := client.GenerateBasicAuthKey(clientID, clientSecret)

	// Create OAuth client
	oauthClient, err := client.NewOAuthClient(client.Config{
		AuthKey: basicAuth,
		Scope:   types.ScopeSaluteSpeechPers,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth client: %w", err)
	}

	// Create token manager
	tokenMgr := client.NewTokenManager(oauthClient, client.TokenManagerConfig{})

	clientUpload, err := upload.NewClient(tokenMgr, upload.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create upload client: %w", err)
	}

	clientAsync, err := async.NewClient(tokenMgr, async.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create async client: %w", err)
	}

	return &clientImpl{
		clientUpload: clientUpload,
		clientAsync:  clientAsync,
		logger:       logger,
	}, nil
}

// CreateTask creates a new recognition task.
func (s *clientImpl) CreateTask(ctx context.Context, req *async.Request) (*async.Response, error) {
	resp, err := s.clientAsync.CreateTask(ctx, req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// WaitForResult waits for the recognition task to complete.
func (s *clientImpl) WaitForResult(ctx context.Context, responseResultID string) (*async.TaskResult, error) {
	result, err := s.clientAsync.WaitForResult(ctx, responseResultID, 2*time.Second, 5*time.Minute)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ExtractText extracts text from the recognition result.
func (s *clientImpl) ExtractText(ctx context.Context, responseFileID string) (string, error) {
	byteData, err := s.clientAsync.DownloadTaskResult(ctx, responseFileID)
	if err != nil {
		return "", err
	}
	return ExtractTextFromResults(byteData)
}

// Upload uploads an audio file for recognition.
func (s *clientImpl) Upload(ctx context.Context, pathAudioFile string) (*async.Request, error) {
	audioType, detectErr := utils.DetectAudioContentType(pathAudioFile)
	if detectErr != nil {
		s.logger.Error("Failed to detect audio type", slog.String("error", detectErr.Error()))
	}

	uploadResp, err := s.clientUpload.UploadFromFile(ctx, pathAudioFile, audioType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload audio: %w", err)
	}

	return &async.Request{
		RequestFileID: uploadResp.Result.RequestFileID,
		Options: &async.Options{
			AudioEncoding: async.EncodingOGG_OPUS,
			SampleRate:    16000,
			Model:         async.ModelGeneral,
			Language:      "ru-RU",
		},
	}, nil
}
