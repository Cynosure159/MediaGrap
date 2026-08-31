package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const SessionCookieName = "mediagrap_session"

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

type Session struct {
	ID        string
	Token     string
	CSRFToken string
	User      User
	ExpiresAt time.Time
}

type Service struct{ db *sql.DB }

func NewService(db *sql.DB) *Service { return &Service{db: db} }

func (s *Service) NeedsSetup(ctx context.Context) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return false, err
	}
	return count == 0, nil
}

func (s *Service) Setup(ctx context.Context, username, password string) (*Session, error) {
	if err := validateCredentials(username, password); err != nil {
		return nil, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var existing int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&existing); err != nil {
		return nil, err
	}
	if existing > 0 {
		return nil, errors.New("setup has already been completed")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO users(username, password_hash) VALUES(?, ?)`, strings.TrimSpace(username), string(hash))
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	session, err := newSession(ctx, tx, User{ID: mustLastID(result), Username: strings.TrimSpace(username)})
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) Login(ctx context.Context, username, password string) (*Session, error) {
	var user User
	var passwordHash string
	err := s.db.QueryRowContext(ctx, `SELECT id, username, password_hash FROM users WHERE username = ?`, strings.TrimSpace(username)).Scan(&user.ID, &user.Username, &passwordHash)
	if err != nil || bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return nil, errors.New("invalid username or password")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	session, err := newSession(ctx, tx, user)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return session, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, errors.New("missing session")
	}
	hash := sha256.Sum256([]byte(token))
	session := &Session{Token: token}
	var expires string
	err := s.db.QueryRowContext(ctx, `SELECT s.id, s.csrf_token, s.expires_at, u.id, u.username FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.token_hash = ? AND s.expires_at > datetime('now')`, hash[:]).Scan(&session.ID, &session.CSRFToken, &expires, &session.User.ID, &session.User.Username)
	if err != nil {
		return nil, errors.New("invalid session")
	}
	session.ExpiresAt, _ = time.Parse("2006-01-02 15:04:05", expires)
	return session, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	hash := sha256.Sum256([]byte(token))
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hash[:])
	return err
}

func newSession(ctx context.Context, tx *sql.Tx, user User) (*Session, error) {
	id, err := randomToken(24)
	if err != nil {
		return nil, err
	}
	token, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	csrf, err := randomToken(24)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256([]byte(token))
	expires := time.Now().UTC().Add(14 * 24 * time.Hour)
	_, err = tx.ExecContext(ctx, `INSERT INTO sessions(id, user_id, token_hash, csrf_token, expires_at) VALUES(?, ?, ?, ?, ?)`, id, user.ID, hash[:], csrf, expires.Format("2006-01-02 15:04:05"))
	if err != nil {
		return nil, err
	}
	return &Session{ID: id, Token: token, CSRFToken: csrf, User: user, ExpiresAt: expires}, nil
}

func randomToken(bytes int) (string, error) {
	buffer := make([]byte, bytes)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buffer), nil
}

func validateCredentials(username, password string) error {
	if len(strings.TrimSpace(username)) < 3 || len(strings.TrimSpace(username)) > 64 {
		return errors.New("username must be 3 to 64 characters")
	}
	if len(password) < 12 || len(password) > 256 {
		return errors.New("password must be 12 to 256 characters")
	}
	return nil
}

func mustLastID(result sql.Result) int64 { id, _ := result.LastInsertId(); return id }
