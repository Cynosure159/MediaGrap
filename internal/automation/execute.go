package automation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/mediagrap/mediagrap/internal/files"
	"github.com/mediagrap/mediagrap/internal/jobs"
)

type fingerprint struct {
	Exists   bool
	Size     int64
	Modified int64
	Identity string
	Hash     string
}

func identity(info os.FileInfo) string {
	if stat, ok := info.Sys().(*syscall.Stat_t); ok {
		return strconv.FormatUint(uint64(stat.Dev), 10) + ":" + strconv.FormatUint(uint64(stat.Ino), 10)
	}
	return info.Name() + ":" + strconv.FormatInt(info.ModTime().UnixNano(), 10)
}
func fmtID(id int64) string { return strconv.FormatInt(id, 10) }
func inspect(root *os.Root, path string, allowMissing bool) (fingerprint, error) {
	if !filepath.IsLocal(path) || filepath.Clean(path) != path || path == "." {
		return fingerprint{}, ErrConflict
	}
	components := strings.Split(path, string(filepath.Separator))
	prefix := ""
	for i, part := range components {
		prefix = filepath.Join(prefix, part)
		info, err := root.Lstat(prefix)
		if errors.Is(err, os.ErrNotExist) && allowMissing {
			return fingerprint{}, nil
		}
		if err != nil {
			return fingerprint{}, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fingerprint{}, ErrConflict
		}
		if i < len(components)-1 {
			if !info.IsDir() {
				return fingerprint{}, ErrConflict
			}
			continue
		}
		if !info.Mode().IsRegular() {
			return fingerprint{}, ErrConflict
		}
		fp := fingerprint{Exists: true, Size: info.Size(), Modified: info.ModTime().UnixNano(), Identity: identity(info)}
		// Sidecars are cheap to fingerprint fully; large media files use stat identity
		// until a cross-device move needs streamed content verification.
		if info.Size() <= 32<<20 {
			f, err := root.Open(path)
			if err != nil {
				return fingerprint{}, err
			}
			h := sha256.New()
			_, err = io.Copy(h, io.LimitReader(f, 32<<20+1))
			f.Close()
			if err != nil {
				return fingerprint{}, err
			}
			fp.Hash = hex.EncodeToString(h.Sum(nil))
		}
		return fp, nil
	}
	return fingerprint{}, ErrConflict
}
func (s *Service) opState(ctx context.Context, id string, index int, state, expected, temp string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE automation_operations SET state=?,expected_hash=?,temporary_path=?,updated_at=? WHERE plan_id=? AND operation_index=?`, state, expected, temp, time.Now().UnixMilli(), id, index)
	return err
}
func (s *Service) runPlan(ctx context.Context, job jobs.Job, progress func(int, string)) error {
	var payload planPayload
	if json.Unmarshal([]byte(job.Payload), &payload) != nil {
		return ErrInvalid
	}
	p, err := s.getPlan(ctx, payload.PlanID)
	if err != nil {
		return err
	}
	if p.State != "queued" || p.Digest != payload.Digest || p.TokenID != payload.TokenID {
		return ErrConflict
	}
	token, err := s.tokens.Current(ctx, p.TokenID)
	if err != nil || !token.Has("plans:apply") || !token.Allows(p.SourceID) {
		s.failPlan(ctx, p.ID, "conflicted")
		return ErrDenied
	}
	ctx, cancel := s.watchRevocation(ctx, p.TokenID)
	defer cancel()
	release, err := files.LockMutation(ctx)
	if err != nil {
		return err
	}
	defer release()
	path, err := s.root(ctx, p.SourceID)
	if err != nil {
		return err
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return err
	}
	defer root.Close()
	info, err := root.Stat(".")
	if err != nil || identity(info) != p.rootIdentity {
		s.failPlan(ctx, p.ID, "conflicted")
		return ErrConflict
	}
	recordDigest := ""
	if record, recordErr := s.metadata.Record(ctx, p.MediaID); recordErr == nil {
		recordDigest = hash(record)
	}
	if recordDigest != p.metadataDigest {
		s.failPlan(ctx, p.ID, "conflicted")
		return ErrConflict
	}
	// Validate the complete frozen plan before the first mutation.
	for _, op := range p.ops {
		fp, err := inspect(root, op.Target, true)
		if err != nil || fp != op.Before {
			s.failPlan(ctx, p.ID, "conflicted")
			return ErrConflict
		}
		if op.Kind == "rename" {
			source, err := inspect(root, op.Source, false)
			if err != nil || source != op.SourceBefore {
				s.failPlan(ctx, p.ID, "conflicted")
				return ErrConflict
			}
		}
	}
	r, err := s.db.ExecContext(ctx, `UPDATE automation_plans SET state='running' WHERE id=? AND state='queued'`, p.ID)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	for i, op := range p.ops {
		if err = ctx.Err(); err == nil {
			err = s.executeOperation(ctx, root, p, i, op)
		}
		if err != nil {
			state := "failed"
			if i > 0 {
				state = "partial"
			}
			if errors.Is(err, context.Canceled) {
				state = "cancelled"
			}
			failureCtx, cancelFailure := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			_, _ = s.db.ExecContext(failureCtx, `UPDATE automation_operations SET error_code='EXECUTION_STOPPED',updated_at=? WHERE plan_id=? AND operation_index=?`, time.Now().UnixMilli(), p.ID, i)
			cancelFailure()
			s.failPlan(context.WithoutCancel(ctx), p.ID, state)
			return errors.New("PLAN_EXECUTION_STOPPED; inspect operation records before a new preview")
		}
		progress(i+1, "Approved operation completed")
	}
	return s.completePlan(ctx, p)
}
func (s *Service) failPlan(ctx context.Context, id, state string) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, _ = s.db.ExecContext(ctx, `UPDATE automation_plans SET state=? WHERE id=? AND state!='applied'`, state, id)
}
func (s *Service) executeOperation(ctx context.Context, root *os.Root, p storedPlan, index int, op operation) error {
	if err := s.opState(ctx, p.ID, index, "intent", "", ""); err != nil {
		return err
	}
	// os.Root confines resolution even if a directory is replaced during an
	// operation. Symlinks are also rejected by inspect before each publication.
	parentPath := filepath.Dir(op.Target)
	if err := root.MkdirAll(parentPath, 0750); err != nil {
		return err
	}
	parent, err := root.OpenRoot(parentPath)
	if err != nil {
		return err
	}
	defer parent.Close()
	base := filepath.Base(op.Target)
	temporary := ".mediagrap-" + uuid.NewString() + ".tmp"
	tempPath := filepath.Join(parentPath, temporary)
	var expected string
	if op.Kind == "rename" {
		current, checkErr := inspect(root, op.Source, false)
		if checkErr != nil || current != op.SourceBefore {
			return ErrConflict
		}
		// Link publishes without replacing any target. It also avoids cross-device
		// assumptions; EXDEV takes the verified streaming-copy path below.
		if err = s.opState(ctx, p.ID, index, "publishing", op.SourceBefore.Hash, ""); err != nil {
			return err
		}
		if err = s.linkSource(root, op.Source, op.Target); err == nil {
			now, checkErr := inspect(root, op.Source, false)
			if checkErr != nil || now != op.SourceBefore {
				return ErrConflict
			}
			if err = ctx.Err(); err != nil {
				return err
			}
			if err = root.Remove(op.Source); err != nil {
				return err
			}
			return s.commitOperation(ctx, p, index, op)
		} else if !errors.Is(err, syscall.EXDEV) {
			return err
		}
	}
	f, err := parent.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	staged := false
	defer func() {
		f.Close()
		if !staged {
			_ = parent.Remove(temporary)
		}
	}()
	hashWriter := sha256.New()
	writer := io.MultiWriter(f, hashWriter)
	if op.Kind == "rename" {
		src, openErr := root.Open(op.Source)
		if openErr != nil {
			return openErr
		}
		_, err = copyContext(ctx, writer, src)
		src.Close()
	} else {
		content := []byte(op.Content)
		if op.Kind == "artwork" {
			content, err = s.metadata.AutomationArtwork(ctx, op.Provider, op.URL, op.MIME)
			if err != nil {
				return errors.New("ARTWORK_DOWNLOAD_FAILED")
			}
		}
		_, err = writer.Write(content)
	}
	if err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	expected = hex.EncodeToString(hashWriter.Sum(nil))
	// Record content identity before any final name becomes visible.
	if err = s.opState(ctx, p.ID, index, "staged", expected, tempPath); err != nil {
		return err
	}
	staged = true
	current, err := inspect(root, op.Target, true)
	if err != nil || current != op.Before {
		return ErrConflict
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	if op.Kind == "rename" {
		source, err := inspect(root, op.Source, false)
		if err != nil || source != op.SourceBefore {
			return ErrConflict
		}
		if err = parent.Link(temporary, base); err != nil {
			return err
		}
		if err = verifyHash(ctx, root, op.Target, expected); err != nil {
			return err
		}
		source, err = inspect(root, op.Source, false)
		if err != nil || source != op.SourceBefore {
			return ErrConflict
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = root.Remove(op.Source); err != nil {
			return err
		}
		_ = parent.Remove(temporary)
	} else if op.Before.Exists {
		if err = parent.Rename(temporary, base); err != nil {
			return err
		}
	} else {
		if err = parent.Link(temporary, base); err != nil {
			return err
		}
		_ = parent.Remove(temporary)
	}
	// Durably publish the directory entry where the filesystem supports syncing.
	directory, err := parent.Open(".")
	if err != nil {
		return err
	}
	err = directory.Sync()
	directory.Close()
	if err != nil && !errors.Is(err, syscall.EINVAL) {
		return err
	}
	return s.commitOperation(ctx, p, index, op)
}
func copyContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	buffer := make([]byte, 128<<10)
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, readErr := src.Read(buffer)
		if n > 0 {
			written, err := dst.Write(buffer[:n])
			total += int64(written)
			if err != nil {
				return total, err
			}
			if written != n {
				return total, io.ErrShortWrite
			}
		}
		if readErr == io.EOF {
			return total, nil
		}
		if readErr != nil {
			return total, readErr
		}
	}
}
func verifyHash(ctx context.Context, root *os.Root, path, expected string) error {
	f, err := root.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = copyContext(ctx, h, f); err != nil {
		return err
	}
	if hex.EncodeToString(h.Sum(nil)) != expected {
		return ErrConflict
	}
	return nil
}
func (s *Service) commitOperation(ctx context.Context, p storedPlan, index int, op operation) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	r, err := tx.ExecContext(ctx, `UPDATE automation_operations SET state='done',updated_at=? WHERE plan_id=? AND operation_index=? AND state!='done'`, time.Now().UnixMilli(), p.ID, index)
	if err != nil {
		return err
	}
	n, _ := r.RowsAffected()
	if n == 0 {
		return nil
	}
	var rootPath string
	if err = tx.QueryRowContext(ctx, `SELECT root_path FROM sources WHERE id=?`, p.SourceID).Scan(&rootPath); err != nil {
		return err
	}
	if op.Kind == "artwork" {
		if _, err = tx.ExecContext(ctx, `INSERT INTO artwork_assets(media_item_id,kind,provider,provider_asset_id,target_path) VALUES(?,?,?,?,?) ON CONFLICT(media_item_id,kind) DO UPDATE SET provider=excluded.provider,provider_asset_id=excluded.provider_asset_id,target_path=excluded.target_path,updated_at=datetime('now')`, p.MediaID, op.AssetKind, op.Provider, op.ProviderAssetID, filepath.Join(rootPath, op.Target)); err != nil {
			return err
		}
	}
	if op.Kind != "rename" {
		kind := "nfo"
		if op.Kind == "artwork" {
			kind = "image"
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO sidecar_assets(media_item_id,relative_path,kind) VALUES(?,?,?) ON CONFLICT(media_item_id,relative_path) DO UPDATE SET kind=excluded.kind`, p.MediaID, op.Target, kind); err != nil {
			return err
		}
	}
	if op.Kind == "rename" {
		if _, err = tx.ExecContext(ctx, `UPDATE artwork_assets SET target_path=? WHERE media_item_id=? AND target_path=?`, filepath.Join(rootPath, op.Target), p.MediaID, filepath.Join(rootPath, op.Source)); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE media_items SET relative_path=? WHERE source_id=? AND relative_path=?`, op.Target, p.SourceID, op.Source); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE sidecar_assets SET relative_path=? WHERE media_item_id=? AND relative_path=?`, op.Target, p.MediaID, op.Source); err != nil {
			return err
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO audit_entries(action,media_item_id,target_path,detail,outcome,recoverability) VALUES(?,?,?,?,?,?)`, "automation."+op.Kind, p.MediaID, op.Target, "token="+p.TokenID+" plan="+p.ID, "applied", "atomic_replace_only"); err != nil {
		return err
	}
	return tx.Commit()
}
func (s *Service) completePlan(ctx context.Context, p storedPlan) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `UPDATE automation_plans SET state='applied' WHERE id=? AND state!='applied' AND NOT EXISTS(SELECT 1 FROM automation_operations WHERE plan_id=? AND state!='done')`, p.ID, p.ID)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return ErrConflict
	}
	eventType := map[string]string{"write": "write_plan.applied", "artwork": "artwork_plan.applied", "rename": "rename_plan.applied"}[p.Kind]
	id := "evt_" + uuid.NewString()
	now := time.Now()
	body, _ := json.Marshal(map[string]any{"id": id, "schemaVersion": 1, "type": eventType, "occurredAt": now.UTC().Format(time.RFC3339Nano), "aggregate": map[string]any{"type": "plan", "id": p.ID, "version": 1}, "data": map[string]any{"planId": p.ID, "sourceId": p.SourceID, "mediaId": p.MediaID, "operationCount": len(p.ops)}})
	if _, err = tx.ExecContext(ctx, `INSERT INTO system_events(id,event_type,aggregate_id,aggregate_version,source_id,payload_bytes,occurred_at) VALUES(?,?,?,1,?,?,?)`, id, eventType, p.ID, p.SourceID, body, now.UnixMilli()); err != nil {
		return err
	}
	return tx.Commit()
}

// Recover reconciles already-published operations; it never initiates another
// file mutation or deletes a source after restart. Ambiguous work needs review.
func (s *Service) Recover(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM automation_plans WHERE state IN ('running','partial','failed','cancelled')`)
	if err != nil {
		return err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		p, err := s.getPlan(ctx, id)
		if err != nil {
			return err
		}
		path, err := s.root(ctx, p.SourceID)
		if err != nil {
			s.failPlan(ctx, id, "needs_review")
			continue
		}
		root, err := os.OpenRoot(path)
		if err != nil {
			s.failPlan(ctx, id, "needs_review")
			continue
		}
		all := true
		info, rootErr := root.Stat(".")
		if rootErr != nil || identity(info) != p.rootIdentity {
			root.Close()
			s.failPlan(ctx, id, "needs_review")
			continue
		}
		for i, op := range p.ops {
			if p.Items[i].State == "done" {
				continue
			}
			var state, expected string
			if err = s.db.QueryRowContext(ctx, `SELECT state,expected_hash FROM automation_operations WHERE plan_id=? AND operation_index=?`, id, i).Scan(&state, &expected); err != nil {
				root.Close()
				return err
			}
			target, checkErr := inspect(root, op.Target, false)
			ok := checkErr == nil && target.Exists
			if op.Kind == "rename" {
				source, sourceErr := inspect(root, op.Source, true)
				ok = ok && sourceErr == nil && !source.Exists
				if expected != "" {
					ok = ok && verifyHash(ctx, root, op.Target, expected) == nil
				} else {
					ok = ok && target.Identity == op.SourceBefore.Identity && target.Size == op.SourceBefore.Size
				}
			}
			if op.Kind != "rename" {
				ok = ok && expected != "" && verifyHash(ctx, root, op.Target, expected) == nil
			}
			if ok && (state == "publishing" || state == "staged") {
				if err = s.commitOperation(ctx, p, i, op); err != nil {
					root.Close()
					return err
				}
			} else {
				all = false
			}
		}
		root.Close()
		if all {
			if err = s.completePlan(ctx, p); err != nil {
				return err
			}
		} else {
			s.failPlan(ctx, id, "needs_review")
		}
	}
	return nil
}
