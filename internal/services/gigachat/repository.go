package gigachatservice

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/skiphead/go-gigachat"
	"github.com/skiphead/salutespeech/client"
	"github.com/skiphead/salutespeech/types"
)

// Client defines the interface for GigaChat operations.
type Client interface {
	Completion(ctx context.Context, systemContent, userContent string) (*gigachat.ChatResponse, error)
}

// clientImpl implements Client using the GigaChat API.
type clientImpl struct {
	clientChat *gigachat.Client
	logger     *slog.Logger
}

// NewClient creates a new GigaChat client instance.
func NewClient(clientID, clientSecret string, logger *slog.Logger) (Client, error) {
	basicAuth := client.GenerateBasicAuthKey(clientID, clientSecret)

	// Create OAuth client
	oauthClient, err := client.NewOAuthClient(client.Config{
		AuthKey: basicAuth,
		Scope:   types.ScopeGigaChatAPIPers,
		Timeout: 30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create OAuth client: %w", err)
	}

	// Create token manager
	tokenMgr := client.NewTokenManager(oauthClient, client.TokenManagerConfig{})

	clientChat, err := gigachat.NewClient(tokenMgr, gigachat.Config{
		BaseURL:       "https://gigachat.devices.sberbank.ru/api/v1",
		AllowInsecure: false,
		Timeout:       30 * time.Second,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create GigaChat client: %w", err)
	}

	return &clientImpl{
		clientChat: clientChat,
		logger:     logger,
	}, nil
}

// Completion sends a chat completion request to GigaChat.
func (s *clientImpl) Completion(ctx context.Context, systemContent, userContent string) (*gigachat.ChatResponse, error) {
	chatReq := &gigachat.ChatRequest{
		Model: gigachat.ModelGigaChatProPreview.String(),
		Messages: []gigachat.Message{
			{
				Role:    gigachat.RoleSystem,
				Content: systemContent,
			},
			{
				Role:    gigachat.RoleUser,
				Content: userContent,
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	response, err := s.clientChat.Completion(ctx, chatReq)
	if err != nil {
		return nil, err
	}

	return response, nil
}
