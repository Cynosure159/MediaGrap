package tokens

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrInvalid = errors.New("invalid token configuration")
var ErrUnauthorized = errors.New("invalid token")

type Token struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	OwnerID    int64    `json:"ownerId"`
	Prefix     string   `json:"prefix"`
	Scopes     []string `json:"scopes"`
	SourceIDs  []int64  `json:"sourceIds"`
	ExpiresAt  int64    `json:"expiresAt"`
	RevokedAt  *int64   `json:"revokedAt"`
	LastUsedAt *int64   `json:"lastUsedAt"`
	CreatedAt  int64    `json:"createdAt"`
}

func (t Token) Has(scope string) bool    { return slices.Contains(t.Scopes, scope) }
func (t Token) Allows(source int64) bool { return slices.Contains(t.SourceIDs, source) }

type Input struct {
	Name      string   `json:"name"`
	Scopes    []string `json:"scopes"`
	SourceIDs []int64  `json:"sourceIds"`
	ExpiresAt int64    `json:"expiresAt"`
}
type Created struct {
	Token  Token  `json:"token"`
	Secret string `json:"secret"`
}
type Service struct{ db *sql.DB }

func New(db *sql.DB) *Service { return &Service{db: db} }
func (s *Service) Create(ctx context.Context, owner int64, in Input) (Created, error) {
	now := time.Now().UnixMilli()
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 100 || len(in.SourceIDs) == 0 || len(in.SourceIDs) > 100 || in.ExpiresAt <= now || in.ExpiresAt > time.Now().AddDate(1, 0, 0).UnixMilli() {
		return Created{}, ErrInvalid
	}
	if len(in.Scopes) == 0 {
		in.Scopes = []string{"media:read", "metadata:read", "jobs:read"}
	}
	for _, scope := range in.Scopes {
		if !slices.Contains([]string{"media:read", "metadata:read", "jobs:read", "jobs:write", "metadata:write", "plans:preview", "plans:apply"}, scope) {
			return Created{}, ErrInvalid
		}
	}
	for _, id := range in.SourceIDs {
		var exists bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sources WHERE id=?)`, id).Scan(&exists); err != nil {
			return Created{}, err
		}
		if !exists {
			return Created{}, ErrInvalid
		}
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return Created{}, err
	}
	secret := "mgp_" + base64.RawURLEncoding.EncodeToString(b)
	hash := sha256.Sum256([]byte(secret))
	id := "tok_" + uuid.NewString()
	scopes, _ := json.Marshal(in.Scopes)
	sources, _ := json.Marshal(in.SourceIDs)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Created{}, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO api_tokens(id,name,owner_user_id,token_prefix,token_hash,scopes_json,source_ids_json,expires_at,created_at) SELECT ?,?,?,?,?,?,?,?,? WHERE (SELECT COUNT(*) FROM api_tokens WHERE owner_user_id=? AND revoked_at IS NULL AND expires_at>?)<100`, id, in.Name, owner, secret[:12], hash[:], string(scopes), string(sources), in.ExpiresAt, now, owner, now)
	if err != nil {
		return Created{}, err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return Created{}, ErrInvalid
	}
	for _, source := range in.SourceIDs {
		if _, err = tx.ExecContext(ctx, `INSERT INTO api_token_sources(token_id,source_id) VALUES(?,?) ON CONFLICT DO NOTHING`, id, source); err != nil {
			return Created{}, err
		}
	}
	if err = tx.Commit(); err != nil {
		return Created{}, err
	}
	return Created{Token{ID: id, Name: in.Name, OwnerID: owner, Prefix: secret[:12], Scopes: in.Scopes, SourceIDs: in.SourceIDs, ExpiresAt: in.ExpiresAt, CreatedAt: now}, secret}, nil
}
func scan(row interface{ Scan(...any) error }) (Token, error) {
	var t Token
	var scopes, sources string
	err := row.Scan(&t.ID, &t.Name, &t.OwnerID, &t.Prefix, &scopes, &sources, &t.ExpiresAt, &t.RevokedAt, &t.LastUsedAt, &t.CreatedAt)
	if err != nil {
		return t, err
	}
	if json.Unmarshal([]byte(scopes), &t.Scopes) != nil || json.Unmarshal([]byte(sources), &t.SourceIDs) != nil {
		return Token{}, ErrUnauthorized
	}
	return t, nil
}

const columns = `t.id,t.name,t.owner_user_id,t.token_prefix,t.scopes_json,(SELECT json_group_array(source_id) FROM api_token_sources WHERE token_id=t.id),t.expires_at,t.revoked_at,t.last_used_at,t.created_at`

func (s *Service) Authenticate(ctx context.Context, secret string) (Token, error) {
	if len(secret) != 47 || !strings.HasPrefix(secret, "mgp_") {
		return Token{}, ErrUnauthorized
	}
	hash := sha256.Sum256([]byte(secret))
	now := time.Now().UnixMilli()
	t, err := scan(s.db.QueryRowContext(ctx, `SELECT `+columns+` FROM api_tokens t JOIN users u ON u.id=t.owner_user_id WHERE t.token_hash=? AND t.audience='mcp' AND t.revoked_at IS NULL AND t.expires_at>?`, hash[:], now))
	if err != nil {
		return Token{}, ErrUnauthorized
	}
	// Single-user installation: every existing user is the administrator. Future
	// roles must be intersected here before allowing any additional capabilities.
	if _, err = s.db.ExecContext(ctx, `UPDATE api_tokens SET last_used_at=? WHERE id=? AND (last_used_at IS NULL OR last_used_at<?)`, now, t.ID, now-60000); err != nil {
		return Token{}, err
	}
	return t, nil
}
func (s *Service) List(ctx context.Context, owner int64) ([]Token, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+columns+` FROM api_tokens t WHERE owner_user_id=? ORDER BY created_at DESC LIMIT 100`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Token{}
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	return items, rows.Err()
}
func (s *Service) Revoke(ctx context.Context, owner int64, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE api_tokens SET revoked_at=? WHERE id=? AND owner_user_id=? AND revoked_at IS NULL`, time.Now().UnixMilli(), id, owner)
	return err
}
func (s *Service) Audit(ctx context.Context, t Token, method, code, requestID string, duration time.Duration) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO mcp_audit(token_id,owner_user_id,method,result_code,request_id,duration_ms,created_at) VALUES(?,?,?,?,?,?,?)`, t.ID, t.OwnerID, method, code, requestID, duration.Milliseconds(), time.Now().UnixMilli())
	return err
}
func (s *Service) Cleanup(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM mcp_audit WHERE id IN (SELECT id FROM mcp_audit WHERE created_at<? ORDER BY id LIMIT 500)`, time.Now().Add(-7*24*time.Hour).UnixMilli())
	return err
}

// Current revalidates queued automation without retaining the bearer secret.
func (s *Service) Current(ctx context.Context, id string) (Token, error) {
	t, err := scan(s.db.QueryRowContext(ctx, `SELECT `+columns+` FROM api_tokens t JOIN users u ON u.id=t.owner_user_id WHERE t.id=? AND t.audience='mcp' AND t.revoked_at IS NULL AND t.expires_at>?`, id, time.Now().UnixMilli()))
	if err != nil {
		return Token{}, ErrUnauthorized
	}
	return t, nil
}
