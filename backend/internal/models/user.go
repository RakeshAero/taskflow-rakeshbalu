package models

import "time"

// User represents a row in the users table.
// The struct tags tell two things:
//   - `db:"..."` → sqlx maps this column name when scanning query results
//   - `json:"..."` → encoding/json uses this key when writing API responses
//
// password is intentionally OMITTED from json output (json:"-").
// You never want to send a bcrypt hash to the client — not even accidentally.
type User struct {
	ID        string    `db:"id"         json:"id"`
	Name      string    `db:"name"       json:"name"`
	Email     string    `db:"email"      json:"email"`
	Password  string    `db:"password"   json:"-"`          // never serialised to JSON
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// RegisterInput is what the client sends to POST /auth/register.
// Keeping input structs separate from the model means you control exactly
// what fields are accepted — the User struct is for DB reads, not writes.
type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Validate checks all required fields and returns a map of field → error message.
// Returns nil if input is valid.
// The map shape matches the spec: { "fields": { "email": "is required" } }
func (i *RegisterInput) Validate() map[string]string {
	errs := map[string]string{}

	if i.Name == "" {
		errs["name"] = "is required"
	}
	if i.Email == "" {
		errs["email"] = "is required"
	}
	if i.Password == "" {
		errs["password"] = "is required"
	} else if len(i.Password) < 8 {
		errs["password"] = "must be at least 8 characters"
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// LoginInput is what the client sends to POST /auth/login.
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (i *LoginInput) Validate() map[string]string {
	errs := map[string]string{}

	if i.Email == "" {
		errs["email"] = "is required"
	}
	if i.Password == "" {
		errs["password"] = "is required"
	}

	if len(errs) > 0 {
		return errs
	}
	return nil
}

// AuthResponse is returned by both /auth/register and /auth/login.
// The client stores the token in localStorage and sends it as
// "Authorization: Bearer <token>" on every subsequent request.
type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}