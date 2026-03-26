package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skiphead/go-letopis/internal/domain/entity"
)

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

	err := row.Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.TelegramID,
		&user.UserName,
		&user.FirstName,
		&user.LastName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return &user, nil
}

// Create inserts a new user record into the database.
func (r *userRepository) Create(ctx context.Context, user *entity.User) (*entity.User, error) {
	if user == nil {
		return nil, errors.New("user is nil")
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

	row := r.pool.QueryRow(ctx, sqlQuery, user.TelegramID, user.UserName, user.FirstName, user.LastName)

	return r.scanUser(row)
}

// Get retrieves a user by Telegram ID.
func (r *userRepository) Get(ctx context.Context, telegramID int64) (*entity.User, error) {
	if telegramID == 0 {
		return nil, fmt.Errorf("telegram id is required")
	}

	query := `SELECT id, created_at, updated_at, telegram_id, username, first_name, last_name 
		FROM users 
		WHERE telegram_id = $1`

	row := r.pool.QueryRow(ctx, query, telegramID)

	return r.scanUser(row)
}
