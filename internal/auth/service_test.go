package auth

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

func newTestDB(t *testing.T) *Service {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "auth_test.db"))
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatalf("migrate test db: %v", err)
	}
	return NewService(db)
}

func TestNeedsSetup(t *testing.T) {
	service := newTestDB(t)
	ctx := t.Context()

	needed, err := service.NeedsSetup(ctx)
	if err != nil {
		t.Fatalf("NeedsSetup error: %v", err)
	}
	if !needed {
		t.Fatal("expected NeedsSetup=true for fresh DB")
	}

	_, err = service.Setup(ctx, "admin", "adminpassword123")
	if err != nil {
		t.Fatalf("Setup error: %v", err)
	}

	needed, err = service.NeedsSetup(ctx)
	if err != nil {
		t.Fatalf("NeedsSetup error: %v", err)
	}
	if needed {
		t.Fatal("expected NeedsSetup=false after setup")
	}
}

func TestSetupValidation(t *testing.T) {
	service := newTestDB(t)
	ctx := t.Context()

	tests := []struct {
		name    string
		user    string
		pass    string
		wantErr string
	}{
		{"username too short", "ab", "adminpassword123", "username must be 3 to 64 characters"},
		{"username too long", strings.Repeat("a", 65), "adminpassword123", "username must be 3 to 64 characters"},
		{"password too short", "admin", "short", "password must be 12 to 256 characters"},
		{"password too long", "admin", strings.Repeat("a", 257), "password must be 12 to 256 characters"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := service.Setup(ctx, tc.user, tc.pass)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("expected error containing %q, got %v", tc.wantErr, err)
			}
		})
	}
}

func TestSetupCannotBeRepeated(t *testing.T) {
	service := newTestDB(t)
	ctx := t.Context()

	session, err := service.Setup(ctx, "admin", "adminpassword123")
	if err != nil {
		t.Fatalf("first setup failed: %v", err)
	}
	if session == nil || session.User.Username != "admin" || session.CSRFToken == "" || session.Token == "" {
		t.Fatalf("invalid session returned from setup: %+v", session)
	}

	_, err = service.Setup(ctx, "admin2", "adminpassword123")
	if err == nil || !strings.Contains(err.Error(), "already been completed") {
		t.Fatalf("expected already completed error, got %v", err)
	}
}

func TestLoginAndAuthenticate(t *testing.T) {
	service := newTestDB(t)
	ctx := t.Context()

	_, err := service.Setup(ctx, "testuser", "securepassword123")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Login with correct credentials
	session, err := service.Login(ctx, "testuser", "securepassword123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if session.User.Username != "testuser" || session.Token == "" || session.CSRFToken == "" {
		t.Fatalf("unexpected session after login: %+v", session)
	}

	// Authenticate with valid token
	authSession, err := service.Authenticate(ctx, session.Token)
	if err != nil {
		t.Fatalf("authenticate failed: %v", err)
	}
	if authSession.User.Username != "testuser" || authSession.CSRFToken != session.CSRFToken {
		t.Fatalf("authenticated session mismatch: got %+v, want %+v", authSession, session)
	}

	// Login with wrong password
	_, err = service.Login(ctx, "testuser", "wrongpassword123")
	if err == nil {
		t.Fatal("expected error on wrong password, got nil")
	}

	// Login with nonexistent user
	_, err = service.Login(ctx, "nonexistent", "securepassword123")
	if err == nil {
		t.Fatal("expected error on nonexistent user, got nil")
	}
}

func TestAuthenticateInvalidTokens(t *testing.T) {
	service := newTestDB(t)
	ctx := t.Context()

	// Empty token
	_, err := service.Authenticate(ctx, "")
	if err == nil || !strings.Contains(err.Error(), "missing session") {
		t.Fatalf("expected missing session error, got %v", err)
	}

	// Nonexistent token
	_, err = service.Authenticate(ctx, "invalid-token-string")
	if err == nil || !strings.Contains(err.Error(), "invalid session") {
		t.Fatalf("expected invalid session error, got %v", err)
	}
}

func TestLogout(t *testing.T) {
	service := newTestDB(t)
	ctx := t.Context()

	session, err := service.Setup(ctx, "logoutuser", "securepassword123")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Verify authenticate works before logout
	_, err = service.Authenticate(ctx, session.Token)
	if err != nil {
		t.Fatalf("authenticate failed before logout: %v", err)
	}

	// Logout
	if err := service.Logout(ctx, session.Token); err != nil {
		t.Fatalf("logout failed: %v", err)
	}

	// Verify authenticate fails after logout
	_, err = service.Authenticate(ctx, session.Token)
	if err == nil {
		t.Fatal("expected authenticate to fail after logout, got nil")
	}
}

func TestExpiredSession(t *testing.T) {
	service := newTestDB(t)
	ctx := t.Context()

	session, err := service.Setup(ctx, "expireuser", "securepassword123")
	if err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	// Manually set expires_at in DB to past
	_, err = service.db.ExecContext(ctx, `UPDATE sessions SET expires_at = ? WHERE id = ?`,
		time.Now().UTC().Add(-1*time.Hour).Format("2006-01-02 15:04:05"), session.ID)
	if err != nil {
		t.Fatalf("update expires_at: %v", err)
	}

	_, err = service.Authenticate(ctx, session.Token)
	if err == nil {
		t.Fatal("expected expired session to fail authenticate, got nil")
	}
}
