package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound   = errors.New("user not found")
	ErrEmailTaken = errors.New("email already taken")
)

type User struct {
	ID           uuid.UUID
	Name         string
	Email        string
	PasswordHash string
	AvatarURL    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const userColumns = "id::text, name, email, password_hash, avatar_url, created_at, updated_at"

func scanUser(row pgx.Row) (*User, error) {
	var (
		u     User
		rawID string
	)

	if err := row.Scan(&rawID, &u.Name, &u.Email, &u.PasswordHash, &u.AvatarURL, &u.CreatedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}

	id, err := uuid.Parse(rawID)
	if err != nil {
		return nil, fmt.Errorf("parse user id: %w", err)
	}
	u.ID = id

	return &u, nil
}

func (r *Repository) Create(ctx context.Context, name, email, passwordHash string) (*User, error) {
	row := r.pool.QueryRow(ctx,
		`INSERT INTO users (name, email, password_hash)
		 VALUES ($1, $2, $3)
		 RETURNING `+userColumns,
		name, email, passwordHash)

	u, err := scanUser(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	return u, nil
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE email = $1`, email)

	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by email: %w", err)
	}

	return u, nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+userColumns+` FROM users WHERE id = $1`, id.String())

	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return u, nil
}

func (r *Repository) FindByOAuthAccount(ctx context.Context, provider, providerAccountID string) (*User, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT u.id::text, u.name, u.email, u.password_hash, u.avatar_url, u.created_at, u.updated_at
		 FROM users u
		 JOIN oauth_accounts oa ON oa.user_id = u.id
		 WHERE oa.provider = $1 AND oa.provider_account_id = $2`,
		provider, providerAccountID)

	u, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find user by oauth account: %w", err)
	}

	return u, nil
}

func (r *Repository) CreateWithOAuth(ctx context.Context, name, email string, avatarURL *string, provider, providerAccountID string) (*User, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin oauth user tx: %w", err)
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx,
		`INSERT INTO users (name, email, avatar_url)
		 VALUES ($1, $2, $3)
		 RETURNING `+userColumns,
		name, email, avatarURL)

	u, err := scanUser(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("insert oauth user: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO oauth_accounts (user_id, provider, provider_account_id)
		 VALUES ($1, $2, $3)`,
		u.ID.String(), provider, providerAccountID); err != nil {
		return nil, fmt.Errorf("insert oauth account: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit oauth user tx: %w", err)
	}

	return u, nil
}

func (r *Repository) LinkOAuthAccount(ctx context.Context, userID uuid.UUID, provider, providerAccountID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO oauth_accounts (user_id, provider, provider_account_id)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (provider, provider_account_id) DO NOTHING`,
		userID.String(), provider, providerAccountID)
	if err != nil {
		return fmt.Errorf("link oauth account: %w", err)
	}

	return nil
}
