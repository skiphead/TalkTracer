package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skiphead/go-letopis/internal/domain/entity"
)

// MeetingRepository defines the interface for meeting data operations.
type MeetingRepository interface {
	Create(ctx context.Context, meeting *entity.Meeting) error
	List(ctx context.Context, userID int64) ([]entity.Meeting, error)
	Get(ctx context.Context, id, telegramID int64) (*entity.Meeting, error)
	SearchByKeywords(ctx context.Context, req entity.SearchRequest) ([]entity.TranscriptionRecord, error)
}

// meetingRepository implements MeetingRepository using PostgreSQL.
type meetingRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewMeetingRepository creates a new MeetingRepository instance.
func NewMeetingRepository(db *pgxpool.Pool, logger *slog.Logger) MeetingRepository {
	return &meetingRepository{
		pool:   db,
		logger: logger,
	}
}

// Create inserts a new meeting record into the database.
func (r *meetingRepository) Create(ctx context.Context, meeting *entity.Meeting) error {
	sqlQuery := `INSERT INTO meetings (created_at, updated_at, user_id, title, transcription, summary, audio_file_id, duration_seconds) 
		VALUES (now(), now(), $1, $2, $3, $4, $5, $6)`

	_, err := r.pool.Exec(ctx, sqlQuery,
		meeting.UserID, meeting.Title,
		meeting.Transcription, meeting.Summary,
		meeting.AudioFileID, meeting.DurationSeconds)
	if err != nil {
		return err
	}

	return nil
}

// Get retrieves a meeting by ID and user Telegram ID.
func (r *meetingRepository) Get(ctx context.Context, id, telegramID int64) (*entity.Meeting, error) {
	if id <= 0 {
		return nil, fmt.Errorf("invalid id")
	}

	sqlQuery := `SELECT m.id, 
		m.created_at,
		m.updated_at,
		m.user_id, 
		m.title, 
		m.transcription,
		m.summary,
		m.audio_file_id,
		m.duration_seconds
	FROM meetings m
	INNER JOIN users u ON m.user_id = u.id
	WHERE m.id = $1 AND u.telegram_id = $2`

	var meeting entity.Meeting
	var summary sql.NullString
	var transcription sql.NullString

	err := r.pool.QueryRow(ctx, sqlQuery, id, telegramID).Scan(
		&meeting.ID,
		&meeting.CreatedAt,
		&meeting.UpdatedAt,
		&meeting.UserID,
		&meeting.Title,
		&transcription,
		&summary,
		&meeting.AudioFileID,
		&meeting.DurationSeconds)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("meeting not found")
		}
		return nil, err
	}

	// Handle NULL values
	if transcription.Valid {
		meeting.Transcription = transcription.String
	} else {
		meeting.Transcription = ""
	}

	if summary.Valid {
		meeting.Summary = summary.String
	} else {
		meeting.Summary = ""
	}

	return &meeting, nil
}

// List retrieves all meetings for a given user.
func (r *meetingRepository) List(ctx context.Context, userID int64) ([]entity.Meeting, error) {
	sqlQuery := `SELECT m.id,
		m.created_at,
		m.updated_at,
		m.user_id,
		m.title,
		m.transcription,
		m.summary,
		m.audio_file_id,
		m.duration_seconds
	FROM meetings m
	INNER JOIN users u ON m.user_id = u.id
	WHERE u.telegram_id = $1
	ORDER BY m.created_at, m.updated_at`

	rows, err := r.pool.Query(ctx, sqlQuery, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var meetings []entity.Meeting
	for rows.Next() {
		var meeting entity.Meeting
		var summary sql.NullString
		var transcription sql.NullString

		err = rows.Scan(
			&meeting.ID,
			&meeting.CreatedAt,
			&meeting.UpdatedAt,
			&meeting.UserID,
			&meeting.Title,
			&transcription,
			&summary,
			&meeting.AudioFileID,
			&meeting.DurationSeconds)
		if err != nil {
			return nil, err
		}

		// Handle NULL values
		if transcription.Valid {
			meeting.Transcription = transcription.String
		} else {
			meeting.Transcription = ""
		}

		if summary.Valid {
			meeting.Summary = summary.String
		} else {
			meeting.Summary = ""
		}

		meetings = append(meetings, meeting)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return meetings, nil
}

// SearchByKeywords performs a full-text search on meetings by keywords.
func (r *meetingRepository) SearchByKeywords(ctx context.Context, req entity.SearchRequest) ([]entity.TranscriptionRecord, error) {
	if len(req.Keywords) == 0 {
		return []entity.TranscriptionRecord{}, nil
	}

	// Build tsQuery from keywords using plainto_tsquery
	tsQuery := strings.Join(req.Keywords, " ")

	query := `
		SELECT m.id, m.user_id, m.transcription
		FROM meetings m
		INNER JOIN users u ON m.user_id = u.id
		WHERE u.telegram_id = $1
		AND to_tsvector('russian', COALESCE(m.transcription, '')) @@ plainto_tsquery('russian', $2)
		ORDER BY ts_rank(to_tsvector('russian', COALESCE(m.transcription, '')), plainto_tsquery('russian', $2)) DESC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, req.UserID, tsQuery, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	var records []entity.TranscriptionRecord
	for rows.Next() {
		var record entity.TranscriptionRecord
		var transcription sql.NullString

		err = rows.Scan(
			&record.ID,
			&record.UserID,
			&transcription,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Handle NULL value for transcription
		if transcription.Valid {
			record.Transcription = transcription.String
		} else {
			record.Transcription = ""
		}

		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return records, nil
}
