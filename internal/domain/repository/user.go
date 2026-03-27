package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skiphead/go-letopis/internal/domain/entity"
)

// ErrUserNotFound is returned when a user is not found in the database.
var ErrUserNotFound = errors.New("user not found")

// UserRepository defines the interface for user data operations.
type UserRepository interface {
	Create(ctx context.Context, user *entity.User) (*entity.User, error)
	Get(ctx context.Context, telegramID int64) (*entity.User, error)
}

// userRepository implements UserRepository using PostgreSQL.
type userRepository struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(db *pgxpool.Pool, logger *slog.Logger) UserRepository {
	return &userRepository{
		pool:   db,
		logger: logger,
	}
}

// scanUser scans a single user row from the database.
func (r *userRepository) scanUser(row pgx.Row) (*entity.User, error) {
	var user entity.User
	var userName, firstName, lastName sql.NullString

	err := row.Scan(
		&user.ID, &user.CreatedAt, &user.UpdatedAt, &user.TelegramID,
		&userName, &firstName, &lastName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to scan user: %w", err)
	}

	if userName.Valid {
		user.UserName = userName.String
	}
	if firstName.Valid {
		user.FirstName = firstName.String
	}
	if lastName.Valid {
		user.LastName = lastName.String
	}

	return &user, nil
}

// Create inserts a new user record into the database.
func (r *userRepository) Create(ctx context.Context, user *entity.User) (*entity.User, error) {
	if user == nil {
		return nil, errors.New("user is nil")
	}

	// Validate required fields
	if user.TelegramID == 0 {
		return nil, errors.New("telegram_id is required and cannot be 0")
	}

	sqlQuery := `INSERT INTO users (
		created_at,
		updated_at,
		telegram_id,
		username,
		first_name,
		last_name
	) VALUES (
		now(), now(), $1, $2, $3, $4
	) RETURNING id, created_at, updated_at, telegram_id, username, first_name, last_name`

	// Handle nil values for optional fields
	var userName, firstName, lastName any
	if user.UserName == "" {
		userName = nil
	} else {
		userName = user.UserName
	}
	if user.FirstName == "" {
		firstName = nil
	} else {
		firstName = user.FirstName
	}
	if user.LastName == "" {
		lastName = nil
	} else {
		lastName = user.LastName
	}

	row := r.pool.QueryRow(ctx, sqlQuery, user.TelegramID, userName, firstName, lastName)

	createdUser, err := r.scanUser(row)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return createdUser, nil
}

// Get retrieves a user by Telegram ID.
func (r *userRepository) Get(ctx context.Context, telegramID int64) (*entity.User, error) {
	if telegramID == 0 {
		return nil, errors.New("telegram_id is required and cannot be 0")
	}

	query := `SELECT id, created_at, updated_at, telegram_id, username, first_name, last_name 
		FROM users 
		WHERE telegram_id = $1`

	row := r.pool.QueryRow(ctx, query, telegramID)

	user, err := r.scanUser(row)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}
