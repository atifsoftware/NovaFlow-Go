package core

import (
	"errors"
	"net/http"
	"os"
	"time"
)

var ErrInvalidCredentials = errors.New("invalid email or password")
var ErrEmailTaken = errors.New("email address is already registered")

// AuthService implements both session (cookie) and JWT (API) authentication
// against a "users" table with columns id, name, email, password.
type AuthService struct {
	db          *DB
	jwtSecret   string
	sessionTTL  time.Duration
	cookieName  string
}

func NewAuthService(db *DB, jwtSecret string) *AuthService {
	return &AuthService{
		db:         db,
		jwtSecret:  jwtSecret,
		sessionTTL: 24 * time.Hour,
		cookieName: "session_token",
	}
}

type AuthResult struct {
	Success bool
	UserID  int64
	Name    string
	Email   string
	Token   string // only populated for API/JWT login
}

// attemptLogin looks up the user by email and verifies the password hash.
func (a *AuthService) attemptLogin(email, password string) (map[string]interface{}, error) {
	user, err := a.db.Table("users").Where("email", "=", email).First()
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}
	hash, _ := user["password"].(string)
	if !VerifyPassword(password, hash) {
		return nil, ErrInvalidCredentials
	}
	return user, nil
}

// isSecureEnv returns true when running in production or staging,
// where cookies must be sent over HTTPS only.
func isSecureEnv() bool {
	env := os.Getenv("APP_ENV")
	return env == "production" || env == "staging"
}

// Login performs web (session cookie) authentication and sets the cookie
// directly on the response.
func (a *AuthService) Login(w http.ResponseWriter, email, password string) (*AuthResult, error) {
	user, err := a.attemptLogin(email, password)
	if err != nil {
		return nil, err
	}
	id := ToInt64(user["id"])
	token, err := GenerateJWT(Claims{"sub": id, "email": email}, a.jwtSecret, a.sessionTTL)
	if err != nil {
		return nil, err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     a.cookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecureEnv(), // BUG-02: set Secure=true in production/staging
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(a.sessionTTL),
	})
	return &AuthResult{Success: true, UserID: id, Email: email, Name: toString(user["name"])}, nil
}

// LoginAPI performs JWT authentication for API clients (no cookie set —
// the caller stores the token and sends it as a Bearer header).
func (a *AuthService) LoginAPI(email, password string) (*AuthResult, error) {
	user, err := a.attemptLogin(email, password)
	if err != nil {
		return nil, err
	}
	id := ToInt64(user["id"])
	token, err := GenerateJWT(Claims{"sub": id, "email": email}, a.jwtSecret, 7*24*time.Hour)
	if err != nil {
		return nil, err
	}
	return &AuthResult{Success: true, UserID: id, Email: email, Name: toString(user["name"]), Token: token}, nil
}

// Logout clears the session cookie.
func (a *AuthService) Logout(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

// CurrentSessionUser reads and validates the session cookie, returning the
// user ID claim if valid.
func (a *AuthService) CurrentSessionUser(r *http.Request) (interface{}, bool) {
	cookie, err := r.Cookie(a.cookieName)
	if err != nil {
		return nil, false
	}
	claims, err := ParseJWT(cookie.Value, a.jwtSecret)
	if err != nil {
		return nil, false
	}
	sub, ok := claims["sub"]
	return sub, ok
}

// Register creates a new user with a hashed password.
// Returns ErrEmailTaken if the email address is already in use.
func (a *AuthService) Register(name, email, password string) (int64, error) {
	// BUG-03: check for duplicate email before attempting INSERT
	existing, err := a.db.Table("users").Where("email", "=", email).First()
	if err != nil {
		return 0, err
	}
	if existing != nil {
		return 0, ErrEmailTaken
	}
	hashed, err := HashPassword(password)
	if err != nil {
		return 0, err
	}
	return a.db.Table("users").Insert(map[string]interface{}{
		"name":     name,
		"email":    email,
		"password": hashed,
	})
}
