package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/models"
)

//Decleration
type UserRepository struct {
	db *sqlx.DB
}

//constructor
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Using named errors (instead of strings) means the compiler catches typos.
var (
	ErrNotFound      = errors.New("not found")
	ErrEmailTaken    = errors.New("email already in use")
)


// CreateUser inserts a new user row and returns the full created record.
func (r *UserRepository) CreateUser(ctx context.Context, name, email, hashedPassword string) (*models.User, error) {
	
	query := `
		INSERT INTO users (name, email, password)
		VALUES ($1, $2, $3)
		RETURNING *`

	var user models.User

	// => (context,destination,query,params....)
	err := r.db.GetContext(ctx, &user, query, name, email, hashedPassword)
	if err != nil {
		// Error code 23505 = unique_violation.
		if isUniqueViolation(err) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("CreateUser: %w", err)
	}

	return &user, nil
}

// GetUserByEmail finds a user by email address.
// Returns ErrNotFound if no row matches — handler maps this to 401
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	query := `SELECT * FROM users WHERE email = $1 LIMIT 1`

	var user models.User
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		// sql.ErrNoRows is the standard "no matching row" error from database/sql.
		// sqlx wraps it so we check with errors.Is().
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetUserByEmail: %w", err)
	}

	return &user, nil
}

// GetUserByID fetches a user by their UUID primary key.
// Used by the JWT middleware to verify the token's user_id still exists.
func (r *UserRepository) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	query := `SELECT * FROM users WHERE id = $1 LIMIT 1`

	var user models.User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetUserByID: %w", err)
	}

	return &user, nil
}