// Package events records immutable external events inside the business transaction.
package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// JobCompleted must share the transaction that commits the job terminal state.
func JobCompleted(ctx context.Context, tx *sql.Tx, id int64) error {
	var kind, state string
	var source *int64
	var version int64
	if err := tx.QueryRowContext(ctx, `UPDATE jobs SET event_version=event_version+1 WHERE id=? RETURNING kind,state,source_id,event_version`, id).Scan(&kind, &state, &source, &version); err != nil {
		return err
	}
	eventID := "evt_" + uuid.NewString()
	now := time.Now().UTC()
	body, err := json.Marshal(struct {
		ID            string `json:"id"`
		SchemaVersion int    `json:"schemaVersion"`
		Type          string `json:"type"`
		OccurredAt    string `json:"occurredAt"`
		Aggregate     any    `json:"aggregate"`
		Data          any    `json:"data"`
	}{eventID, 1, "job." + state, now.Format(time.RFC3339Nano), struct {
		Type    string `json:"type"`
		ID      string `json:"id"`
		Version int64  `json:"version"`
	}{"job", strconv.FormatInt(id, 10), version}, struct {
		JobID    int64  `json:"jobId"`
		Kind     string `json:"kind"`
		SourceID *int64 `json:"sourceId"`
		State    string `json:"state"`
	}{id, kind, source, state}})
	if err != nil {
		return err
	}
	if len(body) > 64<<10 {
		return errors.New("event exceeds payload limit")
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO system_events(id,event_type,aggregate_id,aggregate_version,source_id,payload_bytes,occurred_at) VALUES(?,?,?,?,?,?,?)`, eventID, "job."+state, id, version, source, body, now.UnixMilli())
	return err
}
