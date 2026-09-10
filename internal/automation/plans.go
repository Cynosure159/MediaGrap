package automation

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mediagrap/mediagrap/internal/files"
	"github.com/mediagrap/mediagrap/internal/metadata"
	"github.com/mediagrap/mediagrap/internal/tokens"
)

type PreviewInput struct {
	MediaID        int64                       `json:"mediaId"`
	Pattern        string                      `json:"pattern,omitempty"`
	Selections     []metadata.ArtworkSelection `json:"selections,omitempty"`
	IdempotencyKey string                      `json:"idempotencyKey"`
}
type operation struct {
	AssetKind       string
	ProviderAssetID string
	Kind            string
	Source          string
	Target          string
	Content         string
	CandidateID     string
	Provider        string
	URL             string
	MIME            string
	Before          fingerprint
	SourceBefore    fingerprint
}
type PlanItem struct {
	Kind        string `json:"kind"`
	Source      string `json:"source,omitempty"`
	Target      string `json:"target"`
	WillReplace bool   `json:"willReplace"`
	State       string `json:"state"`
	Content     string `json:"content,omitempty"`
	CandidateID string `json:"candidateId,omitempty"`
	ErrorCode   string `json:"errorCode,omitempty"`
}
type Plan struct {
	ID             string     `json:"id"`
	Version        int        `json:"version"`
	Digest         string     `json:"digest"`
	TokenID        string     `json:"tokenId"`
	SourceID       int64      `json:"sourceId"`
	MediaID        int64      `json:"mediaId"`
	Kind           string     `json:"kind"`
	State          string     `json:"state"`
	Items          []PlanItem `json:"items"`
	ExpiresAt      int64      `json:"expiresAt"`
	ApprovalURL    string     `json:"approvalUrl"`
	Recoverability string     `json:"recoverability"`
}
type storedPlan struct {
	Plan
	ops            []operation
	rootIdentity   string
	metadataDigest string
}
type planPayload struct {
	TokenID string `json:"tokenId"`
	PlanID  string `json:"planId"`
	Digest  string `json:"digest"`
}
type ApplyInput struct {
	PlanID         string `json:"planId"`
	Version        int    `json:"version"`
	Digest         string `json:"digest"`
	IdempotencyKey string `json:"idempotencyKey"`
}

func (s *Service) root(ctx context.Context, id int64) (string, error) {
	var path string
	if err := s.db.QueryRowContext(ctx, `SELECT root_path FROM sources WHERE id=? AND enabled=1`, id).Scan(&path); err != nil {
		return "", ErrDenied
	}
	return path, nil
}
func (s *Service) Preview(ctx context.Context, p tokens.Token, kind string, in PreviewInput) (Plan, error) {
	if !p.Has("plans:preview") || !p.Has("media:read") || in.MediaID <= 0 || len(in.IdempotencyKey) < 8 || len(in.IdempotencyKey) > 128 {
		return Plan{}, ErrDenied
	}
	source, err := s.source(ctx, p, TaskInput{MediaID: in.MediaID})
	if err != nil {
		return Plan{}, err
	}
	// Deterministic private ID gives previews idempotency without retaining another
	// mutable request log. Changed arguments conflict rather than change a plan.
	id := "plan_" + hash(struct{ Token, Kind, Key string }{p.ID, kind, in.IdempotencyKey})
	var existingRequest string
	if err = s.db.QueryRowContext(ctx, `SELECT request_hash FROM automation_plan_requests WHERE plan_id=?`, id).Scan(&existingRequest); err == nil {
		if existingRequest != hash(in) {
			return Plan{}, ErrConflict
		}
		plan, err := s.getPlan(ctx, id)
		return plan.Plan, err
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Plan{}, err
	}
	release, err := files.LockMutation(ctx)
	if err != nil {
		return Plan{}, err
	}
	defer release()
	if existing, err := s.getPlan(ctx, id); err == nil {
		var requestHash string
		if err = s.db.QueryRowContext(ctx, `SELECT request_hash FROM automation_plan_requests WHERE plan_id=?`, id).Scan(&requestHash); err != nil {
			return Plan{}, err
		}
		if requestHash != hash(in) {
			return Plan{}, ErrConflict
		}
		return existing.Plan, nil
	}
	var pending int
	if err = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM automation_plans WHERE token_id=? AND state IN ('previewed','approved','queued','running') AND expires_at>?`, p.ID, time.Now().UnixMilli()).Scan(&pending); err != nil {
		return Plan{}, err
	}
	if pending >= 100 {
		return Plan{}, errors.New("RATE_LIMITED")
	}
	location, err := s.library.LocateMedia(ctx, in.MediaID)
	if err != nil || !location.Writable {
		return Plan{}, ErrDenied
	}
	rootPath, err := s.root(ctx, source)
	if err != nil {
		return Plan{}, err
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return Plan{}, err
	}
	defer root.Close()
	rootInfo, err := root.Stat(".")
	if err != nil {
		return Plan{}, err
	}
	rootIdentity := identity(rootInfo)
	ops := []operation{}
	switch kind {
	case "write":
		record, err := s.metadata.Record(ctx, in.MediaID)
		if err != nil {
			return Plan{}, err
		}
		legacy, err := s.metadata.Preview(ctx, record, location.AbsolutePath, true)
		if err != nil {
			return Plan{}, err
		}
		target, err := filepath.Rel(rootPath, legacy.TargetPath)
		if err != nil {
			return Plan{}, err
		}
		ops = append(ops, operation{Kind: "write", Target: target, Content: legacy.Content})
	case "artwork":
		if len(in.Selections) == 0 || len(in.Selections) > 20 {
			return Plan{}, ErrInvalid
		}
		legacy, err := s.metadata.PreviewArtworkSelection(ctx, in.MediaID, in.Selections, location.AbsolutePath, true)
		if err != nil {
			return Plan{}, err
		}
		for _, asset := range legacy.Assets {
			target, err := filepath.Rel(rootPath, asset.TargetPath)
			if err != nil {
				return Plan{}, err
			}
			ops = append(ops, operation{Kind: "artwork", Target: target, CandidateID: asset.CandidateID, AssetKind: asset.Kind, ProviderAssetID: asset.ProviderAssetID, Provider: asset.Provider, URL: asset.SourceURL, MIME: asset.MimeType})
		}
	case "rename":
		if len(in.Pattern) == 0 || len(in.Pattern) > 512 {
			return Plan{}, ErrInvalid
		}
		legacy, err := s.library.PreviewRenamePlan(ctx, in.MediaID, in.Pattern)
		if err != nil {
			return Plan{}, err
		}
		if legacy.HasConflicts {
			return Plan{}, ErrConflict
		}
		for _, item := range legacy.Items {
			if item.Operation == "rename_dir" || item.Operation == "keep" {
				continue
			}
			ops = append(ops, operation{Kind: "rename", Source: item.CurrentPath, Target: item.PlannedPath})
		}
	default:
		return Plan{}, ErrInvalid
	}
	if len(ops) == 0 || len(ops) > 100 {
		return Plan{}, ErrInvalid
	}
	targets := map[string]bool{}
	for i := range ops {
		op := &ops[i]
		key := strings.ToLower(filepath.Clean(op.Target))
		if targets[key] {
			return Plan{}, ErrConflict
		}
		targets[key] = true
		op.Before, err = inspect(root, op.Target, true)
		if err != nil {
			return Plan{}, err
		}
		if op.Kind == "rename" {
			if op.Before.Exists {
				return Plan{}, ErrConflict
			}
			op.SourceBefore, err = inspect(root, op.Source, false)
			if err != nil || !op.SourceBefore.Exists {
				return Plan{}, ErrConflict
			}
		}
	}
	recordDigest := ""
	if record, recordErr := s.metadata.Record(ctx, in.MediaID); recordErr == nil {
		recordDigest = hash(record)
	}
	digest := hash(struct {
		Metadata string
		Source   int64
		Root     string
		Ops      []operation
	}{recordDigest, source, rootIdentity, ops})
	now := time.Now().UnixMilli()
	raw, _ := json.Marshal(ops)
	if len(raw) > 96<<10 {
		return Plan{}, errors.New("PLAN_TOO_LARGE")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Plan{}, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `INSERT INTO automation_plans(id,token_id,source_id,media_id,kind,digest,operations_json,state,created_at,expires_at,root_identity,metadata_digest) VALUES(?,?,?,?,?,?,?,'previewed',?,?,?,?)`, id, p.ID, source, in.MediaID, kind, digest, string(raw), now, now+int64(24*time.Hour/time.Millisecond), rootIdentity, recordDigest); err != nil {
		return Plan{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO automation_plan_requests(plan_id,request_hash) VALUES(?,?)`, id, hash(in)); err != nil {
		return Plan{}, err
	}
	for i := range ops {
		if _, err = tx.ExecContext(ctx, `INSERT INTO automation_operations(plan_id,operation_index,state,updated_at) VALUES(?,?,'pending',?)`, id, i, now); err != nil {
			return Plan{}, err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_entries(action,media_item_id,detail) VALUES('automation.preview',?,?)`, in.MediaID, "token="+p.ID+" plan="+id); err != nil {
		return Plan{}, err
	}
	if err = tx.Commit(); err != nil {
		return Plan{}, err
	}
	plan, err := s.getPlan(ctx, id)
	return plan.Plan, err
}
func (s *Service) getPlan(ctx context.Context, id string) (storedPlan, error) {
	var p storedPlan
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT id,token_id,source_id,media_id,kind,digest,operations_json,state,expires_at,root_identity,metadata_digest FROM automation_plans WHERE id=?`, id).Scan(&p.ID, &p.TokenID, &p.SourceID, &p.MediaID, &p.Kind, &p.Digest, &raw, &p.State, &p.ExpiresAt, &p.rootIdentity, &p.metadataDigest)
	if err != nil {
		return p, err
	}
	if err = json.Unmarshal([]byte(raw), &p.ops); err != nil {
		return p, err
	}
	p.Version = 1
	p.ApprovalURL = "/settings?approval=" + p.ID
	p.Recoverability = "atomic_replace_only; partial results may require manual recovery"
	p.Items = []PlanItem{}
	for _, op := range p.ops {
		p.Items = append(p.Items, PlanItem{Kind: op.Kind, Source: op.Source, Target: op.Target, WillReplace: op.Before.Exists, State: "pending", Content: op.Content, CandidateID: op.CandidateID})
	}
	rows, err := s.db.QueryContext(ctx, `SELECT operation_index,state,error_code FROM automation_operations WHERE plan_id=? ORDER BY operation_index`, id)
	if err != nil {
		return p, err
	}
	defer rows.Close()
	for rows.Next() {
		var i int
		var state, code string
		if err = rows.Scan(&i, &state, &code); err != nil {
			return p, err
		}
		if i < 0 || i >= len(p.Items) {
			return p, ErrConflict
		}
		p.Items[i].State = state
		p.Items[i].ErrorCode = code
	}
	return p, rows.Err()
}
func (s *Service) Plan(ctx context.Context, p tokens.Token, id string) (Plan, error) {
	plan, err := s.getPlan(ctx, id)
	if err != nil || plan.TokenID != p.ID || !p.Allows(plan.SourceID) {
		return Plan{}, ErrDenied
	}
	return plan.Plan, nil
}
func (s *Service) AdminPlans(ctx context.Context) ([]Plan, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM automation_plans ORDER BY created_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	items := []Plan{}
	for _, id := range ids {
		p, err := s.getPlan(ctx, id)
		if err != nil {
			return nil, err
		}
		items = append(items, p.Plan)
	}
	return items, nil
}
func (s *Service) Approve(ctx context.Context, owner int64, in ApplyInput) error {
	if in.Version != 1 {
		return ErrConflict
	}
	p, err := s.getPlan(ctx, in.PlanID)
	if err != nil {
		return err
	}
	if in.Digest != p.Digest {
		return ErrConflict
	}
	token, err := s.tokens.Current(ctx, p.TokenID)
	if err != nil || !token.Has("plans:apply") || !token.Allows(p.SourceID) {
		return ErrDenied
	}
	now := time.Now().UnixMilli()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE automation_plans SET state='approved',approved_by=?,approved_at=?,approval_expires_at=? WHERE id=? AND digest=? AND state='previewed' AND expires_at>?`, owner, now, now+300000, in.PlanID, in.Digest, now)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_entries(action,media_item_id,detail) VALUES('automation.approve',?,?)`, p.MediaID, "approver="+fmtID(owner)+" token="+p.TokenID+" plan="+p.ID); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Service) Apply(ctx context.Context, p tokens.Token, kind string, in ApplyInput) (Queued, error) {
	if !p.Has("plans:apply") || in.Version != 1 || len(in.IdempotencyKey) < 8 || len(in.IdempotencyKey) > 128 {
		return Queued{}, ErrDenied
	}
	plan, err := s.getPlan(ctx, in.PlanID)
	if err != nil || plan.TokenID != p.ID || !p.Allows(plan.SourceID) || plan.Kind != kind || plan.Digest != in.Digest {
		return Queued{}, ErrDenied
	}
	payload := planPayload{p.ID, plan.ID, plan.Digest}
	return s.queue(ctx, p, "apply_"+kind+"_plan", in.IdempotencyKey, hash(in), plan.SourceID, "automation_apply", payload, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE automation_plans SET state='queued' WHERE id=? AND state='approved' AND approval_expires_at>? AND expires_at>?`, plan.ID, time.Now().UnixMilli(), time.Now().UnixMilli())
		if err != nil {
			return err
		}
		n, _ := result.RowsAffected()
		if n != 1 {
			return errors.New("APPROVAL_REQUIRED")
		}
		return nil
	})
}
