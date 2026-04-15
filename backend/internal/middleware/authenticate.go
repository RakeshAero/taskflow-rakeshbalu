package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey is a private type for context keys.
// Using a custom type (not a plain string) prevents collisions with
// other packages that also store values in the request context.
type contextKey string

const (
	// ContextUserID is the key under which the authenticated user's UUID
	// is stored in the request context after the JWT is validated.
	ContextUserID contextKey = "user_id"
)

// Claims defines the payload we embed inside each JWT.
// Embedding jwt.RegisteredClaims gives us standard fields like
// ExpiresAt and IssuedAt for free.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// Authenticate returns a chi middleware that:
//  1. Reads the "Authorization: Bearer <token>" header
//  2. Parses and validates the JWT signature + expiry
//  3. Injects the user_id into the request context
//  4. Returns 401 if anything is wrong
//
// Downstream handlers retrieve the user ID with GetUserID(r.Context()).
func Authenticate(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			// ── 1. Extract the token string ───────────────────────────────
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				writeError(w, http.StatusUnauthorized, "authorization header required")
				return
			}

			// Header must be exactly "Bearer <token>" — two parts, space-separated.
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				writeError(w, http.StatusUnauthorized, "authorization header format must be: Bearer <token>")
				return
			}
			tokenString := parts[1]

			// ── 2. Parse and validate the JWT ─────────────────────────────
			// jwt.ParseWithClaims verifies:
			//   - the signature (using our secret key)
			//   - the expiry (ExpiresAt claim)
			//   - the algorithm (we lock it to HS256 via the keyFunc)
			token, err := jwt.ParseWithClaims(
				tokenString,
				&Claims{},
				func(token *jwt.Token) (interface{}, error) {
					// Guard against the "alg: none" attack — an attacker can
					// craft a token with algorithm "none" and no signature.
					// We explicitly reject anything that isn't HMAC.
					if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
						return nil, jwt.ErrSignatureInvalid
					}
					return []byte(jwtSecret), nil
				},
			)
			if err != nil || !token.Valid {
				writeError(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			// ── 3. Extract claims ─────────────────────────────────────────
			claims, ok := token.Claims.(*Claims)
			if !ok || claims.UserID == "" {
				writeError(w, http.StatusUnauthorized, "invalid token claims")
				return
			}

			// ── 4. Inject user_id into context ────────────────────────────
			// context.WithValue creates a new context that carries the user ID.
			// Handlers read it back with GetUserID(r.Context()).
			ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID retrieves the authenticated user's UUID from the request context.
// Returns an empty string if the middleware didn't run (should never happen
// on protected routes, but safe to check in handlers).
func GetUserID(ctx context.Context) string {
	id, _ := ctx.Value(ContextUserID).(string)
	return id
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
