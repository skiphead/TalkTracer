package bot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"gopkg.in/telebot.v3"
)

// updateMetadata contains extracted update data for logging.
type updateMetadata struct {
	updateID string
	chatID   int64
	userID   int64
	username string
	handler  string
}

// loggingMiddleware logs handler execution: timing, user, and result.
func (b *Bot) loggingMiddleware(next telebot.HandlerFunc) telebot.HandlerFunc {
	return func(c telebot.Context) error {
		start := time.Now()
		meta := extractUpdateMetadata(c)

		err := next(c)
		b.logHandlerResult(meta, time.Since(start), err)

		return err
	}
}

// extractUpdateMetadata extracts metadata from the update context.
func extractUpdateMetadata(c telebot.Context) updateMetadata {
	meta := updateMetadata{
		updateID: fmt.Sprintf("%d", c.Update().ID),
		username: "anonymous",
		handler:  "unknown",
	}

	if msg := c.Message(); msg != nil {
		meta.handler = determineMessageHandler(msg)
	} else if cb := c.Callback(); cb != nil {
		meta.handler = "callback"
	}

	if sender := c.Sender(); sender != nil {
		meta.userID = sender.ID
		meta.username = resolveUsername(sender)
	}

	if chat := c.Chat(); chat != nil {
		meta.chatID = chat.ID
	}

	return meta
}

// determineMessageHandler determines the handler type based on message content.
func determineMessageHandler(msg *telebot.Message) string {
	switch {
	case msg.Text != "":
		return msg.Text
	case msg.Audio != nil:
		return "audio"
	case msg.Voice != nil:
		return "voice"
	default:
		return "unknown"
	}
}

// resolveUsername returns the username or a generated fallback.
func resolveUsername(user *telebot.User) string {
	if user.Username != "" {
		return user.Username
	}
	return fmt.Sprintf("user_%d", user.ID)
}

// logHandlerResult logs the handler execution result.
func (b *Bot) logHandlerResult(meta updateMetadata, duration time.Duration, err error) {
	attrs := []slog.Attr{
		slog.String("update_id", meta.updateID),
		slog.Int64("chat_id", meta.chatID),
		slog.Int64("user_id", meta.userID),
		slog.String("username", meta.username),
		slog.String("handler", meta.handler),
		slog.Duration("duration", duration),
	}

	level := slog.LevelInfo
	message := "Handler completed"
	if err != nil {
		level = slog.LevelError
		message = "Handler error"
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	b.logger.LogAttrs(context.Background(), level, message, attrs...)
}
