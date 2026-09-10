package webhooks

import (
	"context"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type Config struct {
	KeyFile string
	Policy  NetworkPolicy
	Paused  bool
}
type Service struct {
	heartbeat atomic.Int64
	db        *sql.DB
	cipher    cipher.AEAD
	policy    NetworkPolicy
	logger    *slog.Logger
	paused    bool
	newClient func(context.Context, string) (*http.Client, error)
}

func New(db *sql.DB, config Config, logger *slog.Logger) *Service {
	a, keyErr := loadCipher(config.KeyFile)
	if logger == nil {
		logger = slog.Default()
	}
	if keyErr != nil && config.KeyFile != "" {
		logger.Warn("webhook signing unavailable", "error_code", "KEY_UNAVAILABLE")
	}
	return &Service{db: db, cipher: a, policy: config.Policy, logger: logger, paused: config.Paused, newClient: config.Policy.client}
}

type Endpoint struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	URL        string   `json:"url"`
	Enabled    bool     `json:"enabled"`
	EventTypes []string `json:"eventTypes"`
	SourceIDs  []int64  `json:"sourceIds"`
	Version    int64    `json:"version"`
	KeyID      string   `json:"keyId"`
	CreatedAt  int64    `json:"createdAt"`
}
type Input struct {
	Name       string   `json:"name"`
	URL        string   `json:"url"`
	Enabled    bool     `json:"enabled"`
	EventTypes []string `json:"eventTypes"`
	SourceIDs  []int64  `json:"sourceIds"`
	Version    int64    `json:"version"`
}
type Created struct {
	Endpoint Endpoint `json:"endpoint"`
	Secret   string   `json:"secret"`
}

func (s *Service) validate(ctx context.Context, in Input) error {
	if len(strings.TrimSpace(in.Name)) == 0 || len(in.Name) > 100 || len(in.EventTypes) == 0 || len(in.SourceIDs) == 0 || len(in.SourceIDs) > 100 {
		return ErrInvalid
	}
	for _, t := range in.EventTypes {
		if t != "job.succeeded" && t != "job.failed" && t != "write_plan.applied" && t != "artwork_plan.applied" && t != "rename_plan.applied" {
			return ErrInvalid
		}
	}
	for _, id := range in.SourceIDs {
		var exists bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM sources WHERE id=?)`, id).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrInvalid
		}
	}
	if in.URL != "" {
		_, _, err := s.policy.validate(ctx, in.URL)
		return err
	}
	return nil
}
func (s *Service) Create(ctx context.Context, in Input) (Created, error) {
	if s.cipher == nil {
		return Created{}, ErrUnavailable
	}
	if in.URL == "" {
		return Created{}, ErrInvalid
	}
	if err := s.validate(ctx, in); err != nil {
		return Created{}, err
	}
	id, keyID := "wh_"+uuid.NewString(), "key_"+uuid.NewString()
	secret := newSecret()
	now := time.Now().UnixMilli()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Created{}, err
	}
	defer tx.Rollback()
	types, _ := json.Marshal(in.EventTypes)
	sources, _ := json.Marshal(in.SourceIDs)
	result, err := tx.ExecContext(ctx, `INSERT INTO webhook_endpoints(id,name,url,enabled,event_types_json,source_ids_json,active_key_id,created_at,updated_at) SELECT ?,?,?,?,?,?,?,?,? WHERE (SELECT COUNT(*) FROM webhook_endpoints WHERE deleted_at IS NULL)<20`, id, in.Name, in.URL, in.Enabled, string(types), string(sources), keyID, now, now)
	if err != nil {
		return Created{}, err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return Created{}, ErrConflict
	}
	for _, source := range in.SourceIDs {
		if _, err = tx.ExecContext(ctx, `INSERT INTO webhook_endpoint_sources(endpoint_id,source_id) VALUES(?,?) ON CONFLICT DO NOTHING`, id, source); err != nil {
			return Created{}, err
		}
	}
	if err = s.insertKey(ctx, tx, id, keyID, secret, now); err != nil {
		return Created{}, err
	}
	if err = tx.Commit(); err != nil {
		return Created{}, err
	}
	endpoint, err := s.Get(ctx, id)
	return Created{endpoint, secret}, err
}
func newSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}
func (s *Service) insertKey(ctx context.Context, tx *sql.Tx, id, keyID, secret string, now int64) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO webhook_signing_keys(id,endpoint_id,encrypted_secret,created_at) VALUES(?,?,?,?)`, keyID, id, encrypt(s.cipher, keyID, []byte(secret)), now)
	return err
}
func (s *Service) List(ctx context.Context) ([]Endpoint, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id,name,enabled,event_types_json,(SELECT json_group_array(source_id) FROM webhook_endpoint_sources WHERE endpoint_id=webhook_endpoints.id),config_version,active_key_id,created_at FROM webhook_endpoints WHERE deleted_at IS NULL ORDER BY created_at,id LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Endpoint{}
	for rows.Next() {
		var e Endpoint
		var types, sources string
		if err = rows.Scan(&e.ID, &e.Name, &e.Enabled, &types, &sources, &e.Version, &e.KeyID, &e.CreatedAt); err != nil {
			return nil, err
		}
		if json.Unmarshal([]byte(types), &e.EventTypes) != nil || json.Unmarshal([]byte(sources), &e.SourceIDs) != nil {
			return nil, ErrInvalid
		}
		e.URL = "[configured]"
		items = append(items, e)
	}
	return items, rows.Err()
}
func (s *Service) Get(ctx context.Context, id string) (Endpoint, error) {
	items, err := s.List(ctx)
	if err != nil {
		return Endpoint{}, err
	}
	for _, e := range items {
		if e.ID == id {
			return e, nil
		}
	}
	return Endpoint{}, sql.ErrNoRows
}
func (s *Service) Update(ctx context.Context, id string, in Input) error {
	if err := s.validate(ctx, in); err != nil {
		return err
	}
	types, _ := json.Marshal(in.EventTypes)
	sources, _ := json.Marshal(in.SourceIDs)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UnixMilli()
	r, err := tx.ExecContext(ctx, `UPDATE webhook_endpoints SET name=?,url=CASE WHEN ?='' THEN url ELSE ? END,enabled=?,event_types_json=?,source_ids_json=?,config_version=config_version+1,updated_at=? WHERE id=? AND config_version=? AND deleted_at IS NULL`, in.Name, in.URL, in.URL, in.Enabled, string(types), string(sources), now, id, in.Version)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return ErrConflict
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM webhook_endpoint_sources WHERE endpoint_id=?`, id); err != nil {
		return err
	}
	for _, source := range in.SourceIDs {
		if _, err = tx.ExecContext(ctx, `INSERT INTO webhook_endpoint_sources(endpoint_id,source_id) VALUES(?,?) ON CONFLICT DO NOTHING`, id, source); err != nil {
			return err
		}
	}
	if !in.Enabled {
		if _, err = tx.ExecContext(ctx, `UPDATE webhook_deliveries SET state='cancelled',updated_at=? WHERE endpoint_id=? AND state IN ('queued','retry_wait')`, now, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}
func (s *Service) Delete(ctx context.Context, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UnixMilli()
	if _, err = tx.ExecContext(ctx, `UPDATE webhook_endpoints SET enabled=0,deleted_at=?,updated_at=? WHERE id=? AND deleted_at IS NULL`, now, now, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE webhook_deliveries SET state='cancelled',updated_at=? WHERE endpoint_id=? AND state IN ('queued','retry_wait')`, now, id); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Service) Rotate(ctx context.Context, id string) (map[string]string, error) {
	if s.cipher == nil {
		return nil, ErrUnavailable
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	keyID, secret, now := "key_"+uuid.NewString(), newSecret(), time.Now().UnixMilli()
	r, err := tx.ExecContext(ctx, `UPDATE webhook_endpoints SET active_key_id=?,config_version=config_version+1,updated_at=? WHERE id=? AND deleted_at IS NULL`, keyID, now, id)
	if err != nil {
		return nil, err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return nil, sql.ErrNoRows
	}
	if _, err = tx.ExecContext(ctx, `UPDATE webhook_signing_keys SET retired_at=? WHERE endpoint_id=? AND retired_at IS NULL`, now, id); err != nil {
		return nil, err
	}
	if err = s.insertKey(ctx, tx, id, keyID, secret, now); err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return map[string]string{"keyId": keyID, "secret": secret}, nil
}
func (s *Service) Test(ctx context.Context, id string) (string, error) {
	var raw string
	var version int64
	if err := s.db.QueryRowContext(ctx, `SELECT url,config_version FROM webhook_endpoints WHERE id=? AND enabled=1 AND deleted_at IS NULL`, id).Scan(&raw, &version); err != nil {
		return "", err
	}
	if _, _, err := s.policy.validate(ctx, raw); err != nil {
		return "", err
	}
	if s.cipher == nil {
		return "", ErrUnavailable
	}
	now := time.Now().UnixMilli()
	eventID, deliveryID := "evt_"+uuid.NewString(), "dlv_"+uuid.NewString()
	body, _ := json.Marshal(map[string]any{"id": eventID, "schemaVersion": 1, "type": "webhook.test", "occurredAt": time.Now().UTC().Format(time.RFC3339Nano), "data": map[string]bool{"test": true}})
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO system_events(id,event_type,aggregate_id,aggregate_version,payload_bytes,occurred_at,dispatched_at) VALUES(?,'webhook.test',?,1,?,?,?)`, eventID, eventID, body, now, now); err != nil {
		return "", err
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO webhook_deliveries(id,endpoint_id,event_id,target_url,config_version,state,next_attempt_at,created_at,updated_at) SELECT ?,id,?,url,config_version,'queued',?,?,? FROM webhook_endpoints WHERE id=? AND enabled=1 AND deleted_at IS NULL AND config_version=?`, deliveryID, eventID, now, now, now, id, version)
	if err != nil {
		return "", err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if n != 1 {
		return "", ErrConflict
	}
	return deliveryID, tx.Commit()
}
