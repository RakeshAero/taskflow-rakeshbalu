package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/middleware"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/models"
	"github.com/RakeshAero/taskflow-rakeshbalu/backend/internal/repository"
)

// AuthHandler handles POST /auth/register and POST /auth/login.
type AuthHandler struct {
	users     *repository.UserRepository
	jwtSecret string
}

func NewAuthHandler(users *repository.UserRepository, jwtSecret string) *AuthHandler {
	return &AuthHandler{users: users, jwtSecret: jwtSecret}
}

// Register handles POST /auth/register
//
// Flow:
//  1. Decode + validate JSON body
//  2. Hash password with bcrypt (cost 12)
//  3. Insert user row
//  4. Sign and return a JWT
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	// ── 1. Decode body ────────────────────────────────────────────────────────
	var input models.RegisterInput
	if !decodeJSON(w, r, &input) {
		return // decodeJSON already wrote 400
	}

	// ── 2. Validate ───────────────────────────────────────────────────────────
	if errs := input.Validate(); errs != nil {
		writeValidationError(w, errs)
		return
	}

	// ── 3. Hash password ──────────────────────────────────────────────────────
	// Cost 12 matches the spec requirement (cost ≥ 12).
	// bcrypt is intentionally slow — cost 12 takes ~250ms, which is fine for
	// auth endpoints but would be terrible for a hot loop.
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), 12)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not process password")
		return
	}

	// ── 4. Insert user ────────────────────────────────────────────────────────
	user, err := h.users.CreateUser(r.Context(), input.Name, input.Email, string(hash))
	if err != nil {
		// Email already registered → 400 with a clear message.
		// We use 400 (not 409) to match common API conventions and avoid
		// leaking whether an email exists (though the message itself does confirm
		// it — for a production app you'd make this vaguer).
		if errors.Is(err, repository.ErrEmailTaken) {
			writeValidationError(w, map[string]string{
				"email": "is already in use",
			})
			return
		}
		writeError(w, http.StatusInternalServerError, "could not create user")
		return
	}

	// ── 5. Sign JWT ───────────────────────────────────────────────────────────
	token, err := h.signToken(user.ID, user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate token")
		return
	}

	// 201 Created — new resource was created
	writeJSON(w, http.StatusCreated, models.AuthResponse{
		Token: token,
		User:  *user,
	})
}

// Login handles POST /auth/login
//
// Flow:
//  1. Decode + validate body
//  2. Look up user by email
//  3. Compare bcrypt hash
//  4. Sign and return a JWT
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	// ── 1. Decode + validate ──────────────────────────────────────────────────
	var input models.LoginInput
	if !decodeJSON(w, r, &input) {
		return
	}
	if errs := input.Validate(); errs != nil {
		writeValidationError(w, errs)
		return
	}

	// ── 2. Find user ──────────────────────────────────────────────────────────
	user, err := h.users.GetUserByEmail(r.Context(), input.Email)
	if err != nil {
		// Return 401 whether the email doesn't exist OR the password is wrong.
		// Never confirm which one failed — that would let attackers enumerate accounts.
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	// ── 3. Compare password ───────────────────────────────────────────────────
	// CompareHashAndPassword returns non-nil if the password doesn't match.
	// It's timing-safe — takes the same time regardless of where the mismatch is.
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	// ── 4. Sign JWT ───────────────────────────────────────────────────────────
	token, err := h.signToken(user.ID, user.Email)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not generate token")
		return
	}

	// 200 OK — existing resource accessed
	writeJSON(w, http.StatusOK, models.AuthResponse{
		Token: token,
		User:  *user,
	})
}

// signToken creates and signs a JWT with the user's ID and email as claims.
// Expiry is 24 hours from now as required by the spec.
func (h *AuthHandler) signToken(userID, email string) (string, error) {
	claims := middleware.Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	// HS256 is symmetric — same secret signs and verifies.
	// For a production app you'd use RS256 (asymmetric) so verification
	// can happen without sharing the signing key.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}