package library

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const scanBatchSize = 200

// scanCandidate contains only prepared index data. No filesystem operations or
// metadata parsing are allowed in the repository's batch transaction.
type scanCandidate struct {
	id                                                           int64
	relative, path, title, modified, fingerprint, nfoFingerprint string
	year                                                         *int
	size                                                         int64
	assets                                                       []Sidecar
	changed, classify, hydrate, nfoPresent                       bool
	tv                                                           *scanTVEpisode
}

type scanTVEpisode struct {
	showPath, showTitle, title string
	showYear                   *int
	season, first, last        int
}

func prepareScanCandidate(root, path string, entry fs.DirEntry, snapshot directorySnapshot) (scanCandidate, error) {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
		return scanCandidate{}, fmt.Errorf("invalid scan relative path")
	}
	// Revalidate instead of trusting a directory entry that may have been replaced.
	info, err := os.Lstat(path)
	if err != nil {
		return scanCandidate{}, err
	}
	if !info.Mode().IsRegular() {
		return scanCandidate{}, fmt.Errorf("scan video must remain a regular file")
	}
	title, year := parseHint(entry.Name())
	c := scanCandidate{relative: relative, path: path, title: title, year: year, size: info.Size(), modified: info.ModTime().UTC().Format(time.RFC3339)}
	c.assets = snapshot.sidecars(root, relative)
	h := sha256.New()
	fmt.Fprintf(h, "v1:%d:%d:%d\n", info.Size(), info.ModTime().UnixNano(), info.Mode())
	for _, asset := range c.assets {
		fingerprint, err := scanFileFingerprint(filepath.Join(root, asset.RelativePath))
		if err != nil {
			return scanCandidate{}, err
		}
		fmt.Fprintf(h, "%q:%s:%s\n", asset.RelativePath, asset.Kind, fingerprint)
	}
	// Hydration also accepts movie.nfo in a multi-video directory. Include both
	// candidates even if sidecar attribution does not assign the fallback to us.
	nh := sha256.New()
	for _, nfoPath := range []string{strings.TrimSuffix(path, filepath.Ext(path)) + ".nfo", filepath.Join(filepath.Dir(path), "movie.nfo")} {
		fingerprint, err := scanFileFingerprint(nfoPath)
		if err != nil {
			return scanCandidate{}, err
		}
		c.nfoPresent = c.nfoPresent || fingerprint != "absent"
		fmt.Fprintf(nh, "%q:%s\n", filepath.Base(nfoPath), fingerprint)
	}
	c.nfoFingerprint = fmt.Sprintf("%x", nh.Sum(nil))
	fmt.Fprint(h, c.nfoFingerprint)
	c.fingerprint = fmt.Sprintf("%x", h.Sum(nil))
	if season, first, last, ok := parseEpisodeHint(entry.Name()); ok {
		showPath, showTitle, showYear := tvShowHint(relative)
		if showTitle != "" {
			c.tv = &scanTVEpisode{showPath: showPath, showTitle: showTitle, showYear: showYear, season: season, first: first, last: last, title: episodeTitleHint(entry.Name())}
		}
	}
	return c, nil
}

// Cheap stat identity deliberately does not read video/artwork contents. Same
// size+mtime replacements can be missed; full scans bypass this comparison.
func scanFileFingerprint(path string) (string, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "absent", nil
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d:%d:%d", info.Size(), info.ModTime().UnixNano(), info.Mode()), nil
}

type scanRepository struct{ db *sql.DB }

// applyBatch loads only this bounded batch, then commits seen tracking and all
// changed video/sidecar/TV index rows atomically. IDs/hydration flags are returned
// in candidates so hydration can run after the transaction has released SQLite.
func (r scanRepository) applyBatch(ctx context.Context, sourceID int64, seenAt, mode string, candidates []scanCandidate) error {
	if len(candidates) == 0 {
		return nil
	}
	if len(candidates) > scanBatchSize {
		return errors.New("scan batch exceeds limit")
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(candidates)), ",")
	args := []any{sourceID}
	byPath := make(map[string]int, len(candidates))
	for i := range candidates {
		c := &candidates[i]
		byPath[c.relative] = i
		args = append(args, c.relative)
		c.changed, c.classify, c.hydrate = true, true, c.tv == nil && c.nfoPresent
	}
	rows, err := r.db.QueryContext(ctx, `SELECT m.id,m.relative_path,m.scan_fingerprint,m.nfo_fingerprint,COALESCE(meta.title,'') FROM media_items m LEFT JOIN media_metadata meta ON meta.media_item_id=m.id WHERE m.source_id=? AND m.relative_path IN (`+placeholders+`)`, args...)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		var relative, fingerprint, nfoFingerprint, metadataTitle string
		if err = rows.Scan(&id, &relative, &fingerprint, &nfoFingerprint, &metadataTitle); err != nil {
			rows.Close()
			return err
		}
		c := &candidates[byPath[relative]]
		c.id = id
		c.changed = mode == "full" || fingerprint != c.fingerprint
		c.classify = mode == "full" || fingerprint == ""
		c.hydrate = c.tv == nil && c.nfoPresent && (mode == "full" || nfoFingerprint != c.nfoFingerprint || metadataTitle == "")
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	// Keep the last filename-derived hint for each show in this batch, as the
	// old file-by-file scan did. Directory-derived hints are normally identical.
	lastShowHints := make(map[string]*scanTVEpisode)
	for _, c := range candidates {
		if c.tv != nil {
			lastShowHints[c.tv.showPath] = c.tv
		}
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Only batch-scoped identities are retained. Existing shows/seasons are read,
	// not repeatedly upserted for each episode or on subsequent batches/scans.
	shows := make(map[string]int64)
	seasons := make(map[[2]int64]int64)
	seenIDs := []any{seenAt}
	for i := range candidates {
		c := &candidates[i]
		if err := ctx.Err(); err != nil {
			return err
		}
		if c.changed {
			err := tx.QueryRowContext(ctx, `INSERT INTO media_items(source_id,relative_path,title_hint,year_hint,file_size,modified_at,missing,last_seen_at,scan_fingerprint,nfo_fingerprint) VALUES(?,?,?,?,?,?,0,?,?,?) ON CONFLICT(source_id,relative_path) DO UPDATE SET title_hint=excluded.title_hint,year_hint=excluded.year_hint,file_size=excluded.file_size,modified_at=excluded.modified_at,missing=0,last_seen_at=excluded.last_seen_at,scan_fingerprint=excluded.scan_fingerprint,nfo_fingerprint=excluded.nfo_fingerprint RETURNING id`, sourceID, c.relative, c.title, c.year, c.size, c.modified, seenAt, c.fingerprint, c.nfoFingerprint).Scan(&c.id)
			if err != nil {
				return err
			}
			if _, err := tx.ExecContext(ctx, `DELETE FROM sidecar_assets WHERE media_item_id=?`, c.id); err != nil {
				return err
			}
			for _, asset := range c.assets {
				if _, err := tx.ExecContext(ctx, `INSERT INTO sidecar_assets(media_item_id,relative_path,kind) VALUES(?,?,?)`, c.id, asset.RelativePath, asset.Kind); err != nil {
					return err
				}
			}
		} else {
			seenIDs = append(seenIDs, c.id)
		}
		if c.classify {
			if err := recordScanTVEpisode(ctx, tx, sourceID, *c, lastShowHints, shows, seasons); err != nil {
				return err
			}
		}
	}
	if len(seenIDs) > 1 {
		placeholders = strings.TrimSuffix(strings.Repeat("?,", len(seenIDs)-1), ",")
		if _, err := tx.ExecContext(ctx, `UPDATE media_items SET last_seen_at=?,missing=0 WHERE id IN (`+placeholders+`)`, seenIDs...); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func recordScanTVEpisode(ctx context.Context, tx *sql.Tx, sourceID int64, c scanCandidate, lastShowHints map[string]*scanTVEpisode, shows map[string]int64, seasons map[[2]int64]int64) error {
	if c.tv == nil {
		_, err := tx.ExecContext(ctx, `DELETE FROM tv_episodes WHERE media_item_id=?`, c.id)
		return err
	}
	tv := c.tv
	showID, ok := shows[tv.showPath]
	if !ok {
		var err error
		// Root-layout filename hints can differ across batches. Their final hint
		// is reconciled once after traversal, not oscillated by partial batches.
		showID, err = ensureScanShow(ctx, tx, sourceID, lastShowHints[tv.showPath], tv.showPath != ".")
		if err != nil {
			return err
		}
		shows[tv.showPath] = showID
	}
	key := [2]int64{showID, int64(tv.season)}
	seasonID, ok := seasons[key]
	if !ok {
		err := tx.QueryRowContext(ctx, `SELECT id FROM tv_seasons WHERE show_id=? AND season_number=?`, showID, tv.season).Scan(&seasonID)
		if errors.Is(err, sql.ErrNoRows) {
			err = tx.QueryRowContext(ctx, `INSERT INTO tv_seasons(show_id,season_number) VALUES(?,?) RETURNING id`, showID, tv.season).Scan(&seasonID)
		}
		if err != nil {
			return err
		}
		seasons[key] = seasonID
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO tv_episodes(media_item_id,show_id,season_id,season_number,episode_start,episode_end,title_hint) VALUES(?,?,?,?,?,?,?) ON CONFLICT(media_item_id) DO UPDATE SET show_id=excluded.show_id,season_id=excluded.season_id,season_number=excluded.season_number,episode_start=excluded.episode_start,episode_end=excluded.episode_end,title_hint=excluded.title_hint`, c.id, showID, seasonID, tv.season, tv.first, tv.last, tv.title)
	return err
}

func ensureScanShow(ctx context.Context, tx *sql.Tx, sourceID int64, hint *scanTVEpisode, updateHint bool) (int64, error) {
	var id int64
	var title string
	var year sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT id,title_hint,year_hint FROM tv_shows WHERE source_id=? AND relative_path=?`, sourceID, hint.showPath).Scan(&id, &title, &year)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(ctx, `INSERT INTO tv_shows(source_id,relative_path,title_hint,year_hint) VALUES(?,?,?,?) RETURNING id`, sourceID, hint.showPath, hint.showTitle, hint.showYear).Scan(&id)
	} else if err == nil && updateHint {
		sameYear := (!year.Valid && hint.showYear == nil) || (year.Valid && hint.showYear != nil && year.Int64 == int64(*hint.showYear))
		if title != hint.showTitle || !sameYear {
			_, err = tx.ExecContext(ctx, `UPDATE tv_shows SET title_hint=?,year_hint=?,updated_at=datetime('now') WHERE id=?`, hint.showTitle, hint.showYear, id)
		}
	}
	return id, err
}
