package webhooks

import (
	"bytes"
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Delivery struct {
	ID              string `json:"id"`
	EventID         string `json:"eventId"`
	State           string `json:"state"`
	Attempts        int    `json:"attempts"`
	RetryGeneration int    `json:"retryGeneration"`
	NextAttemptAt   int64  `json:"nextAttemptAt"`
	Status          int    `json:"status"`
	ErrorCode       string `json:"errorCode"`
	CreatedAt       int64  `json:"createdAt"`
}

func (s *Service) Deliveries(ctx context.Context, id string, before int64) ([]Delivery, error) {
	if before <= 0 {
		before = time.Now().Add(time.Second).UnixMilli()
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,event_id,state,attempt_count,retry_generation,next_attempt_at,last_status,last_error_code,created_at FROM webhook_deliveries WHERE endpoint_id=? AND created_at<? ORDER BY created_at DESC,id DESC LIMIT 100`, id, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Delivery{}
	for rows.Next() {
		var d Delivery
		if err = rows.Scan(&d.ID, &d.EventID, &d.State, &d.Attempts, &d.RetryGeneration, &d.NextAttemptAt, &d.Status, &d.ErrorCode, &d.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, rows.Err()
}
func (s *Service) Retry(ctx context.Context, endpointID, id string) error {
	now := time.Now().UnixMilli()
	r, err := s.db.ExecContext(ctx, `UPDATE webhook_deliveries SET state='queued',retry_generation=retry_generation+1,generation_attempts=0,next_attempt_at=?,updated_at=? WHERE id=? AND endpoint_id=? AND state IN ('dead','cancelled') AND updated_at>CASE WHEN state='dead' THEN ? ELSE ? END AND EXISTS(SELECT 1 FROM webhook_endpoints WHERE id=endpoint_id AND enabled=1 AND deleted_at IS NULL)`, now, now, id, endpointID, now-int64(30*24*time.Hour/time.Millisecond), now-int64(7*24*time.Hour/time.Millisecond))
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return ErrConflict
	}
	return nil
}
func (s *Service) Dispatch(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Acquire SQLite's writer lock before selecting, avoiding deferred lock upgrades.
	if _, err = tx.ExecContext(ctx, `UPDATE system_events SET dispatched_at=dispatched_at WHERE 0`); err != nil {
		return err
	}
	rows, err := tx.QueryContext(ctx, `SELECT id,event_type,source_id,occurred_at FROM system_events WHERE dispatched_at IS NULL ORDER BY occurred_at,id LIMIT 100`)
	if err != nil {
		return err
	}
	type pending struct {
		id, kind string
		source   *int64
		created  int64
	}
	list := []pending{}
	for rows.Next() {
		var e pending
		if err = rows.Scan(&e.id, &e.kind, &e.source, &e.created); err != nil {
			rows.Close()
			return err
		}
		list = append(list, e)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, e := range list {
		subscriptions, err := tx.QueryContext(ctx, `SELECT id,url,config_version,event_types_json,(SELECT json_group_array(source_id) FROM webhook_endpoint_sources WHERE endpoint_id=webhook_endpoints.id) FROM webhook_endpoints WHERE enabled=1 AND deleted_at IS NULL AND created_at<=?`, e.created)
		if err != nil {
			return err
		}
		type target struct {
			id, url string
			version int64
		}
		targets := []target{}
		for subscriptions.Next() {
			var t target
			var types, sources string
			if err = subscriptions.Scan(&t.id, &t.url, &t.version, &types, &sources); err != nil {
				subscriptions.Close()
				return err
			}
			var kinds []string
			var ids []int64
			if json.Unmarshal([]byte(types), &kinds) != nil || json.Unmarshal([]byte(sources), &ids) != nil {
				subscriptions.Close()
				return ErrInvalid
			}
			matchKind, matchSource := false, false
			for _, k := range kinds {
				if k == e.kind {
					matchKind = true
				}
			}
			for _, id := range ids {
				if e.source != nil && id == *e.source {
					matchSource = true
				}
			}
			if matchKind && matchSource {
				targets = append(targets, t)
			}
		}
		err = subscriptions.Err()
		subscriptions.Close()
		if err != nil {
			return err
		}
		now := time.Now().UnixMilli()
		for _, t := range targets {
			if _, err = tx.ExecContext(ctx, `INSERT INTO webhook_deliveries(id,endpoint_id,event_id,target_url,config_version,state,next_attempt_at,created_at,updated_at) VALUES(?,?,?,?,?,'queued',?,?,?) ON CONFLICT(endpoint_id,event_id) DO NOTHING`, "dlv_"+uuid.NewString(), t.id, e.id, t.url, t.version, now, now, now); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, `UPDATE system_events SET dispatched_at=? WHERE id=?`, now, e.id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type claim struct {
	id, endpoint, event, kind, url, owner, keyID string
	body, secret                                 []byte
	attempt, generation                          int
	started                                      int64
}

func (s *Service) claim(ctx context.Context) (claim, error) {
	var c claim
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return c, err
	}
	defer tx.Rollback()
	now := time.Now().UnixMilli()
	if _, err = tx.ExecContext(ctx, `UPDATE webhook_deliveries SET state=CASE WHEN generation_attempts>=6 THEN 'dead' ELSE 'retry_wait' END,last_error_code='LEASE_EXPIRED',updated_at=?,lease_owner=NULL,lease_expires_at=NULL WHERE state='delivering' AND lease_expires_at<=?`, now, now); err != nil {
		return c, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE webhook_deliveries SET state='cancelled',updated_at=? WHERE state IN ('queued','retry_wait') AND NOT EXISTS(SELECT 1 FROM webhook_endpoints WHERE id=endpoint_id AND enabled=1 AND deleted_at IS NULL)`, now); err != nil {
		return c, err
	}
	err = tx.QueryRowContext(ctx, `SELECT d.id,d.endpoint_id,d.event_id,e.event_type,d.target_url,e.payload_bytes,k.id,k.encrypted_secret,d.attempt_count+1,d.generation_attempts+1 FROM webhook_deliveries d JOIN webhook_endpoints w ON w.id=d.endpoint_id JOIN webhook_signing_keys k ON k.id=w.active_key_id JOIN system_events e ON e.id=d.event_id WHERE d.state IN ('queued','retry_wait') AND d.next_attempt_at<=? AND w.enabled=1 AND w.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM webhook_deliveries active WHERE active.endpoint_id=d.endpoint_id AND active.state='delivering') ORDER BY d.next_attempt_at,d.id LIMIT 1`, now).Scan(&c.id, &c.endpoint, &c.event, &c.kind, &c.url, &c.body, &c.keyID, &c.secret, &c.attempt, &c.generation)
	if err != nil {
		return c, err
	}
	c.secret, err = decrypt(s.cipher, c.keyID, c.secret)
	if err != nil {
		return c, err
	}
	c.owner = uuid.NewString()
	c.started = now
	if _, err = tx.ExecContext(ctx, `UPDATE webhook_deliveries SET state='delivering',attempt_count=?,generation_attempts=?,lease_owner=?,lease_expires_at=?,updated_at=? WHERE id=?`, c.attempt, c.generation, c.owner, now+30000, now, c.id); err != nil {
		return c, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO webhook_attempts(delivery_id,attempt_number,key_id,started_at) VALUES(?,?,?,?)`, c.id, c.attempt, c.keyID, now); err != nil {
		return c, err
	}
	return c, tx.Commit()
}
func retryDelay(attempt int, header string, now time.Time) time.Duration {
	delays := []time.Duration{time.Minute, 5 * time.Minute, 15 * time.Minute, time.Hour, 6 * time.Hour}
	index := min(max(attempt-1, 0), 4)
	delay := delays[index] + time.Duration(rand.Int64N(int64(delays[index]/5)+1))
	var remote time.Duration
	if seconds, err := strconv.ParseInt(header, 10, 32); err == nil && seconds > 0 {
		remote = time.Duration(seconds) * time.Second
	} else if date, err := http.ParseTime(header); err == nil {
		remote = date.Sub(now)
	}
	return min(max(delay, remote), 24*time.Hour)
}
func (s *Service) deliver(ctx context.Context, c claim) error {
	started := time.Now()
	status := 0
	code := ""
	retry := false
	retryAfter := ""
	client, err := s.newClient(ctx, c.url)
	if err != nil {
		code = "URL_REJECTED"
		if errors.Is(err, ErrResolution) {
			code = "DNS_UNAVAILABLE"
			retry = true
		}
	} else {
		req, requestErr := http.NewRequestWithContext(ctx, http.MethodPost, c.url, bytes.NewReader(c.body))
		if requestErr != nil {
			code = "URL_REJECTED"
		} else {
			timestamp := strconv.FormatInt(time.Now().Unix(), 10)
			for k, v := range map[string]string{"Content-Type": "application/json", "User-Agent": "MediaGrap-Webhook/1", "X-MediaGrap-Event": c.kind, "X-MediaGrap-Event-ID": c.event, "X-MediaGrap-Delivery-ID": c.id, "X-MediaGrap-Timestamp": timestamp, "X-MediaGrap-Key-ID": c.keyID, "X-MediaGrap-Signature": Signature(c.secret, timestamp, c.body)} {
				req.Header.Set(k, v)
			}
			s.logger.Info("webhook request started", "endpoint_id", c.endpoint, "delivery_id", c.id, "attempt", c.attempt)
			resp, sendErr := client.Do(req)
			if sendErr != nil {
				code = "NETWORK_ERROR"
				retry = true
				var unknown x509.UnknownAuthorityError
				var host x509.HostnameError
				var invalid x509.CertificateInvalidError
				if errors.As(sendErr, &unknown) || errors.As(sendErr, &host) || errors.As(sendErr, &invalid) {
					code = "TLS_INVALID"
					retry = false
				}
			} else {
				status = resp.StatusCode
				_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 8192))
				resp.Body.Close()
				if status < 200 || status >= 300 {
					code = "HTTP_ERROR"
					retry = status == 408 || status == 429 || status >= 500
				}
				if status == 429 || status == 503 {
					retryAfter = resp.Header.Get("Retry-After")
				}
			}
		}
	}
	state := "dead"
	if code == "" {
		state = "succeeded"
	} else if retry && c.generation < 6 {
		state = "retry_wait"
	}
	now := time.Now()
	next := now.Add(retryDelay(c.generation, retryAfter, now)).UnixMilli()
	duration := time.Since(started).Milliseconds()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	r, err := tx.ExecContext(ctx, `UPDATE webhook_deliveries SET state=?,last_status=?,last_error_code=?,next_attempt_at=?,updated_at=?,delivered_at=CASE WHEN ?='succeeded' THEN ? ELSE NULL END,lease_owner=NULL,lease_expires_at=NULL WHERE id=? AND state='delivering' AND lease_owner=?`, state, status, code, next, now.UnixMilli(), state, now.UnixMilli(), c.id, c.owner)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return ErrConflict
	}
	if _, err = tx.ExecContext(ctx, `UPDATE webhook_attempts SET duration_ms=?,status_code=?,error_code=? WHERE delivery_id=? AND attempt_number=?`, duration, status, code, c.id, c.attempt); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	s.logger.Info("webhook request completed", "endpoint_id", c.endpoint, "delivery_id", c.id, "event_id", c.event, "state", state, "http_status", status, "error_code", code, "duration_ms", duration)
	return nil
}
func (s *Service) Run(ctx context.Context) {
	var wg sync.WaitGroup
	defer wg.Wait()
	for range 2 {
		wg.Go(func() {
			ticker := time.NewTicker(time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if s.paused || s.cipher == nil {
						continue
					}
					c, err := s.claim(ctx)
					if errors.Is(err, sql.ErrNoRows) {
						continue
					}
					if err != nil {
						s.logger.Error("webhook claim failed", "error_code", "QUEUE_OR_KEY_UNAVAILABLE")
						continue
					}
					if err = s.deliver(ctx, c); err != nil {
						s.logger.Error("webhook result commit failed", "delivery_id", c.id)
					}
				}
			}
		})
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	cycles := 0
	s.heartbeat.Store(time.Now().UnixMilli())
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.heartbeat.Store(time.Now().UnixMilli())
			if err := s.Dispatch(ctx); err != nil {
				s.logger.Error("webhook dispatch failed")
			}
			cycles++
			if cycles%60 == 0 {
				if err := s.cleanup(ctx); err != nil {
					s.logger.Error("webhook retention failed")
				}
			}
		}
	}
}
func (s *Service) cleanup(ctx context.Context) error {
	now := time.Now().UnixMilli()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM webhook_deliveries WHERE id IN (SELECT id FROM webhook_deliveries WHERE (state IN ('succeeded','cancelled') AND updated_at<?) OR (state='dead' AND updated_at<?) LIMIT 500)`, now-int64(7*24*time.Hour/time.Millisecond), now-int64(30*24*time.Hour/time.Millisecond)); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM system_events WHERE id IN (SELECT id FROM system_events e WHERE dispatched_at IS NOT NULL AND occurred_at<? AND NOT EXISTS(SELECT 1 FROM webhook_deliveries d WHERE d.event_id=e.id) LIMIT 500)`, now-int64(7*24*time.Hour/time.Millisecond)); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Service) Status(ctx context.Context) (map[string]any, error) {
	result := map[string]any{"signingReady": s.cipher != nil, "paused": s.paused, "workerHeartbeatAt": s.heartbeat.Load()}
	var pending, dead int
	var oldest *int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*),MIN(occurred_at) FROM system_events WHERE dispatched_at IS NULL`).Scan(&pending, &oldest); err != nil {
		return nil, err
	}
	result["undispatched"] = pending
	result["oldestUndispatchedAt"] = oldest
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhook_deliveries WHERE state IN ('queued','delivering','retry_wait')`).Scan(&pending); err != nil {
		return nil, err
	}
	result["pendingDeliveries"] = pending
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM webhook_deliveries WHERE state='dead'`).Scan(&dead); err != nil {
		return nil, err
	}
	result["deadDeliveries"] = dead
	return result, nil
}
