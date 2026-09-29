package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mediagrap/mediagrap/internal/files"
	"github.com/mediagrap/mediagrap/internal/renamepattern"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/mediagrap/mediagrap/internal/jobs"
)

type RenamePlan struct {
	ID           string           `json:"id"`
	MediaItemID  *int64           `json:"mediaItemId,omitempty"`
	TVShowID     *int64           `json:"tvShowId,omitempty"`
	Pattern      string           `json:"pattern"`
	State        string           `json:"state"` // previewed, applied, partial, failed, cancelled
	Items        []RenamePlanItem `json:"items"`
	Warnings     []string         `json:"warnings"`
	HasConflicts bool             `json:"hasConflicts"`
	CreatedAt    string           `json:"createdAt"`
	AppliedAt    string           `json:"appliedAt,omitempty"`
}

type RenamePlanItem struct {
	Kind        string `json:"kind"`        // video, nfo, image, subtitle, directory
	CurrentPath string `json:"currentPath"` // source-relative path
	PlannedPath string `json:"plannedPath"` // source-relative planned path
	Operation   string `json:"operation"`   // keep, rename, rename_dir, conflict
	Conflict    bool   `json:"conflict"`
	Status      string `json:"status"` // pending, success, failed, skipped
}

// renderNamingTemplate parses and renders a template, then validates every
// resulting path segment after metadata sanitization. Only literal slashes in
// the template can remain directory boundaries.
func renderNamingTemplate(pattern string, tokens map[string]string) (dirTemplate, fileTemplate string, err error) {
	if err := renamepattern.Validate(pattern, renamepattern.Allowed(keys(tokens)...)); err != nil {
		return "", "", err
	}
	stem, err := renamepattern.Render(pattern, tokens, renamepattern.Allowed(keys(tokens)...))
	if err != nil {
		return "", "", err
	}

	parts := strings.Split(stem, "/")
	for i := range parts {
		parts[i] = sanitizeFilename(parts[i])
		if parts[i] == "" || parts[i] == "." || parts[i] == ".." {
			if i == len(parts)-1 {
				return "", "", errors.New("naming pattern produced an empty filename")
			}
			return "", "", errors.New("naming pattern produced an invalid directory segment")
		}
	}
	stem = strings.Join(parts, "/")

	if idx := strings.LastIndex(stem, "/"); idx != -1 {
		dirTemplate = stem[:idx]
		fileTemplate = stem[idx+1:]
	} else {
		fileTemplate = stem
	}
	return dirTemplate, fileTemplate, nil
}

func keys(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for key := range values {
		result = append(result, key)
	}
	return result
}

// evaluateFileRename checks if a planned rename path has collisions against the filesystem or other planned files.
func evaluateFileRename(rootPath, sourceRelPath, plannedRelPath string, plannedPaths map[string]bool) (op string, conflict bool) {
	if strings.EqualFold(sourceRelPath, plannedRelPath) {
		return "keep", false
	}

	targetAbs := filepath.Join(rootPath, plannedRelPath)
	if _, statErr := os.Stat(targetAbs); statErr == nil && targetAbs != filepath.Join(rootPath, sourceRelPath) {
		return "conflict", true
	}

	lower := strings.ToLower(plannedRelPath)
	if plannedPaths[lower] {
		return "conflict", true
	}
	plannedPaths[lower] = true

	return "rename", false
}

func (s *Service) PreviewRenamePlan(ctx context.Context, mediaID int64, pattern string) (RenamePlan, error) {
	location, err := s.LocateMedia(ctx, mediaID)
	if err != nil {
		return RenamePlan{}, err
	}
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return RenamePlan{}, errors.New("pattern cannot be empty")
	}

	var titleStr, originalTitleStr string
	var yearPtr *int
	{
		var metaTitle, metaOrigTitle sql.NullString
		var metaYear sql.NullInt64
		err := s.db.QueryRowContext(ctx, `SELECT title, original_title, year FROM media_metadata WHERE media_item_id=?`, mediaID).Scan(&metaTitle, &metaOrigTitle, &metaYear)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return RenamePlan{}, err
		}
		if metaTitle.Valid && metaTitle.String != "" {
			titleStr = metaTitle.String
		} else {
			titleStr = location.Item.TitleHint
		}
		if metaOrigTitle.Valid {
			originalTitleStr = metaOrigTitle.String
		}
		if metaYear.Valid {
			v := int(metaYear.Int64)
			yearPtr = &v
		} else {
			yearPtr = location.Item.YearHint
		}
	}

	inspection, err := s.InspectMedia(ctx, mediaID)
	if err != nil {
		return RenamePlan{}, err
	}

	videoCodec, audioCodec, resolution := "", "", ""
	if len(inspection.Video) > 0 {
		videoCodec = strings.ToUpper(inspection.Video[0].Codec)
		resolution = resolutionLabel(inspection.Video[0].Width, inspection.Video[0].Height)
	}
	if len(inspection.Audio) > 0 {
		audioCodec = strings.ToUpper(inspection.Audio[0].Codec)
	}

	yearStr := ""
	if yearPtr != nil {
		yearStr = strconv.Itoa(*yearPtr)
	}

	tokens := map[string]string{
		"title":         titleStr,
		"originalTitle": originalTitleStr,
		"year":          yearStr,
		"resolution":    resolution,
		"videoCodec":    videoCodec,
		"audioCodec":    audioCodec,
		"edition":       "",
		"imdbId":        "",
	}

	dirTemplate, fileTemplate, err := renderNamingTemplate(pattern, tokens)
	if err != nil {
		return RenamePlan{}, err
	}

	rootPath := s.sourceRoot(ctx, location.Item.SourceID)
	sourceRelPath := location.Item.RelativePath
	sourceRelDir := filepath.Dir(sourceRelPath)
	plannedRelDir := ""

	var items []RenamePlanItem
	if dirTemplate != "" {
		newRelDir := filepath.Clean(dirTemplate)
		if newRelDir == "." || newRelDir == "/" || strings.HasPrefix(newRelDir, "..") {
			return RenamePlan{}, errors.New("invalid directory pattern")
		}
		plannedRelDir = newRelDir

		if sourceRelDir != "." && sourceRelDir != plannedRelDir && filepath.Dir(sourceRelDir) == filepath.Dir(plannedRelDir) {
			items = append(items, RenamePlanItem{
				Kind:        "directory",
				CurrentPath: sourceRelDir,
				PlannedPath: plannedRelDir,
				Operation:   "rename_dir",
				Status:      "pending",
			})
		}
	}

	currentMediaBase := strings.TrimSuffix(filepath.Base(sourceRelPath), filepath.Ext(sourceRelPath))
	hasConflicts := false
	plannedPaths := make(map[string]bool)

	for _, file := range inspection.Files {
		currentName := filepath.Base(file.RelativePath)
		plannedName := currentName
		if file.Kind == "video" {
			plannedName = fileTemplate + filepath.Ext(currentName)
		} else if strings.HasPrefix(currentName, currentMediaBase) {
			suffix := currentName[len(currentMediaBase):]
			plannedName = fileTemplate + suffix
		}

		plannedRelPath := plannedName
		if plannedRelDir != "" {
			plannedRelPath = filepath.Join(plannedRelDir, plannedName)
		}

		op, conflict := evaluateFileRename(rootPath, file.RelativePath, plannedRelPath, plannedPaths)
		if conflict {
			hasConflicts = true
		}

		items = append(items, RenamePlanItem{
			Kind:        file.Kind,
			CurrentPath: file.RelativePath,
			PlannedPath: plannedRelPath,
			Operation:   op,
			Conflict:    conflict,
			Status:      "pending",
		})
	}

	itemsBytes, _ := json.Marshal(items)
	planID := uuid.New().String()

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO rename_plans (id, media_item_id, pattern, state, items_json, warnings_json, has_conflicts) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		planID, mediaID, pattern, "previewed", string(itemsBytes), "[]", hasConflicts)
	if err != nil {
		return RenamePlan{}, err
	}

	return RenamePlan{
		ID:           planID,
		MediaItemID:  &mediaID,
		Pattern:      pattern,
		State:        "previewed",
		Items:        items,
		Warnings:     []string{},
		HasConflicts: hasConflicts,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *Service) PreviewTVRenamePlan(ctx context.Context, showID int64, seasonNumber *int, episodeID *int64, pattern string) (RenamePlan, error) {
	showDetail, err := s.TVShow(ctx, showID)
	if err != nil {
		return RenamePlan{}, err
	}
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return RenamePlan{}, errors.New("pattern cannot be empty")
	}

	rootPath := s.sourceRoot(ctx, showDetail.Show.SourceID)

	var showTitle, showOrigTitle sql.NullString
	var showYear sql.NullInt64
	_ = s.db.QueryRowContext(ctx, `SELECT title, original_title, year FROM tv_metadata WHERE show_id=?`, showID).Scan(&showTitle, &showOrigTitle, &showYear)
	showTitleVal := showDetail.Show.TitleHint
	if showTitle.Valid && showTitle.String != "" {
		showTitleVal = showTitle.String
	}
	showOrigTitleVal := ""
	if showOrigTitle.Valid {
		showOrigTitleVal = showOrigTitle.String
	}
	showYearVal := ""
	if showYear.Valid {
		showYearVal = strconv.Itoa(int(showYear.Int64))
	} else if showDetail.Show.YearHint != nil {
		showYearVal = strconv.Itoa(*showDetail.Show.YearHint)
	}

	episodeMetaMap := make(map[string]string)
	rows, err := s.db.QueryContext(ctx, `SELECT season_number, episode_number, title FROM tv_episode_metadata WHERE show_id=?`, showID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var sn, en int
			var epTitle string
			if rows.Scan(&sn, &en, &epTitle) == nil && epTitle != "" {
				episodeMetaMap[fmt.Sprintf("%d:%d", sn, en)] = epTitle
			}
		}
	}

	var targetEpisodes []TVEpisode
	for _, ep := range showDetail.Episodes {
		if episodeID != nil && ep.ID != *episodeID {
			continue
		}
		if seasonNumber != nil && ep.SeasonNumber != *seasonNumber {
			continue
		}
		targetEpisodes = append(targetEpisodes, ep)
	}
	if len(targetEpisodes) == 0 {
		return RenamePlan{}, errors.New("no episodes matched the selected scope")
	}

	var items []RenamePlanItem
	hasConflicts := false
	plannedPaths := make(map[string]bool)
	seenDirs := make(map[string]bool)

	for _, ep := range targetEpisodes {
		epTitleVal := episodeMetaMap[fmt.Sprintf("%d:%d", ep.SeasonNumber, ep.EpisodeStart)]
		if epTitleVal == "" {
			epTitleVal = ep.TitleHint
		}

		resVal, vCodecVal, aCodecVal := "", "", ""
		if inspection, inspectErr := s.InspectMedia(ctx, ep.ID); inspectErr == nil {
			if len(inspection.Video) > 0 {
				vCodecVal = strings.ToUpper(inspection.Video[0].Codec)
				resVal = resolutionLabel(inspection.Video[0].Width, inspection.Video[0].Height)
			}
			if len(inspection.Audio) > 0 {
				aCodecVal = strings.ToUpper(inspection.Audio[0].Codec)
			}
		}

		tokens := map[string]string{
			"showTitle":         showTitleVal,
			"originalTitle":     showOrigTitleVal,
			"showOriginalTitle": showOrigTitleVal,
			"seasonNumber":      strconv.Itoa(ep.SeasonNumber),
			"seasonNumberPad":   fmt.Sprintf("%02d", ep.SeasonNumber),
			"episodeNumber":     strconv.Itoa(ep.EpisodeStart),
			"episodeNumberPad":  fmt.Sprintf("%02d", ep.EpisodeStart),
			"episodeTitle":      epTitleVal,
			"year":              showYearVal,
			"resolution":        resVal,
			"videoCodec":        vCodecVal,
			"audioCodec":        aCodecVal,
		}

		dirTemplate, fileTemplate, templateErr := renderNamingTemplate(pattern, tokens)
		if templateErr != nil {
			return RenamePlan{}, templateErr
		}

		sourceRelPath := ep.RelativePath
		sourceRelDir := filepath.Dir(sourceRelPath)
		plannedRelDir := ""
		if dirTemplate != "" {
			plannedRelDir = filepath.Clean(dirTemplate)
		}

		if dirTemplate != "" && sourceRelDir != "." && sourceRelDir != plannedRelDir && filepath.Dir(sourceRelDir) == filepath.Dir(plannedRelDir) && !seenDirs[plannedRelDir] {
			seenDirs[plannedRelDir] = true
			items = append(items, RenamePlanItem{
				Kind:        "directory",
				CurrentPath: sourceRelDir,
				PlannedPath: plannedRelDir,
				Operation:   "rename_dir",
				Status:      "pending",
			})
		}

		currentVideoBase := strings.TrimSuffix(filepath.Base(sourceRelPath), filepath.Ext(sourceRelPath))
		plannedVideoName := fileTemplate + filepath.Ext(sourceRelPath)
		plannedVideoPath := plannedVideoName
		if plannedRelDir != "" {
			plannedVideoPath = filepath.Join(plannedRelDir, plannedVideoName)
		}

		videoOp, videoConflict := evaluateFileRename(rootPath, sourceRelPath, plannedVideoPath, plannedPaths)
		if videoConflict {
			hasConflicts = true
		}

		items = append(items, RenamePlanItem{
			Kind:        "video",
			CurrentPath: sourceRelPath,
			PlannedPath: plannedVideoPath,
			Operation:   videoOp,
			Conflict:    videoConflict,
			Status:      "pending",
		})

		for _, sidecar := range ep.Sidecars {
			sidecarName := filepath.Base(sidecar.RelativePath)
			sidecarBase := strings.TrimSuffix(sidecarName, filepath.Ext(sidecarName))
			plannedSidecarName := sidecarName
			if strings.HasPrefix(sidecarBase, currentVideoBase) {
				suffix := sidecarBase[len(currentVideoBase):]
				plannedSidecarName = fileTemplate + suffix + filepath.Ext(sidecarName)
			}
			plannedSidecarPath := plannedSidecarName
			if plannedRelDir != "" {
				plannedSidecarPath = filepath.Join(plannedRelDir, plannedSidecarName)
			}

			scOp, scConflict := evaluateFileRename(rootPath, sidecar.RelativePath, plannedSidecarPath, plannedPaths)
			if scConflict {
				hasConflicts = true
			}

			items = append(items, RenamePlanItem{
				Kind:        sidecar.Kind,
				CurrentPath: sidecar.RelativePath,
				PlannedPath: plannedSidecarPath,
				Operation:   scOp,
				Conflict:    scConflict,
				Status:      "pending",
			})
		}
	}

	itemsBytes, _ := json.Marshal(items)
	planID := uuid.New().String()

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO rename_plans (id, tv_show_id, pattern, state, items_json, warnings_json, has_conflicts) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		planID, showID, pattern, "previewed", string(itemsBytes), "[]", hasConflicts)
	if err != nil {
		return RenamePlan{}, err
	}

	return RenamePlan{
		ID:           planID,
		TVShowID:     &showID,
		Pattern:      pattern,
		State:        "previewed",
		Items:        items,
		Warnings:     []string{},
		HasConflicts: hasConflicts,
		CreatedAt:    time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *Service) GetRenamePlan(ctx context.Context, planID string) (RenamePlan, error) {
	var p RenamePlan
	var itemsJSON, warningsJSON string
	var hasConflicts int
	var mediaItemID, tvShowID sql.NullInt64
	var appliedAt sql.NullString

	err := s.db.QueryRowContext(ctx,
		`SELECT id, media_item_id, tv_show_id, pattern, state, items_json, warnings_json, has_conflicts, created_at, COALESCE(applied_at,'') FROM rename_plans WHERE id=?`, planID).
		Scan(&p.ID, &mediaItemID, &tvShowID, &p.Pattern, &p.State, &itemsJSON, &warningsJSON, &hasConflicts, &p.CreatedAt, &appliedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RenamePlan{}, errors.New("plan not found")
		}
		return RenamePlan{}, err
	}

	if mediaItemID.Valid {
		p.MediaItemID = &mediaItemID.Int64
	}
	if tvShowID.Valid {
		p.TVShowID = &tvShowID.Int64
	}
	if appliedAt.Valid && appliedAt.String != "" {
		p.AppliedAt = appliedAt.String
	}
	_ = json.Unmarshal([]byte(itemsJSON), &p.Items)
	_ = json.Unmarshal([]byte(warningsJSON), &p.Warnings)
	p.HasConflicts = hasConflicts != 0
	if p.Items == nil {
		p.Items = []RenamePlanItem{}
	}
	if p.Warnings == nil {
		p.Warnings = []string{}
	}

	return p, nil
}

func isCrossDevice(src, dst string) (bool, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return false, err
	}
	dstDir := filepath.Dir(dst)
	dstInfo, err := os.Stat(dstDir)
	if err != nil {
		return false, err
	}
	srcSys, ok1 := srcInfo.Sys().(*syscall.Stat_t)
	dstSys, ok2 := dstInfo.Sys().(*syscall.Stat_t)
	if ok1 && ok2 {
		return srcSys.Dev != dstSys.Dev, nil
	}
	return false, nil
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()
	if _, err := io.Copy(d, s); err != nil {
		return err
	}
	return d.Sync()
}

// moveFile safely moves a file within or across filesystems with parent directory creation.
func moveFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	cross, _ := isCrossDevice(src, dst)
	if cross {
		tempDst := dst + ".tmp"
		if err := copyFile(src, tempDst); err != nil {
			return err
		}
		sInfo, sErr := os.Stat(src)
		dInfo, dErr := os.Stat(tempDst)
		if sErr != nil || dErr != nil || sInfo.Size() != dInfo.Size() {
			_ = os.Remove(tempDst)
			return errors.New("cross-device file copy verification failed")
		}
		if err := os.Rename(tempDst, dst); err != nil {
			_ = os.Remove(tempDst)
			return err
		}
		return os.Remove(src)
	}

	return os.Rename(src, dst)
}

func (s *Service) ApplyRenamePlan(ctx context.Context, planID string, progressFn func(int, string)) error {
	release, lockErr := files.LockMutation(ctx)
	if lockErr != nil {
		return lockErr
	}
	defer release()
	plan, err := s.GetRenamePlan(ctx, planID)
	if err != nil {
		return err
	}

	if plan.State != "previewed" {
		return errors.New("plan is not in previewed state")
	}
	if plan.HasConflicts {
		return errors.New("plan has conflicts")
	}

	var sourceID int64
	if plan.MediaItemID != nil {
		location, locErr := s.LocateMedia(ctx, *plan.MediaItemID)
		if locErr != nil {
			return locErr
		}
		sourceID = location.Item.SourceID
	} else if plan.TVShowID != nil {
		showDetail, showErr := s.TVShow(ctx, *plan.TVShowID)
		if showErr != nil {
			return showErr
		}
		sourceID = showDetail.Show.SourceID
	} else {
		return errors.New("invalid plan without media or TV show reference")
	}

	rootPath := s.sourceRoot(ctx, sourceID)
	failedCount := 0

	// Handle directory renames first
	for i := range plan.Items {
		if plan.Items[i].Operation == "rename_dir" {
			dirItem := &plan.Items[i]
			src := filepath.Join(rootPath, dirItem.CurrentPath)
			dst := filepath.Join(rootPath, dirItem.PlannedPath)

			if !resolvedParentWithinRoot(rootPath, dst) {
				dirItem.Status = "failed"
				failedCount++
				continue
			}
			if _, statErr := os.Stat(dst); statErr == nil && src != dst {
				dirItem.Status = "failed"
				failedCount++
				continue
			}
			if err := moveFile(src, dst); err != nil {
				dirItem.Status = "failed"
				failedCount++
				continue
			}
			dirItem.Status = "success"

			// Adjust current paths for any files inside this renamed directory
			for j := range plan.Items {
				if plan.Items[j].Operation != "rename_dir" {
					if strings.HasPrefix(plan.Items[j].CurrentPath, dirItem.CurrentPath+"/") || plan.Items[j].CurrentPath == dirItem.CurrentPath {
						plan.Items[j].CurrentPath = strings.Replace(plan.Items[j].CurrentPath, dirItem.CurrentPath, dirItem.PlannedPath, 1)
					}
				}
			}
		}
	}

	processed := 0
	for i := range plan.Items {
		item := &plan.Items[i]
		if item.Operation == "rename_dir" || item.Operation == "keep" {
			if item.Operation == "keep" {
				item.Status = "success"
			}
			continue
		}
		if item.Status == "failed" {
			continue
		}

		src := filepath.Join(rootPath, item.CurrentPath)
		dst := filepath.Join(rootPath, item.PlannedPath)

		if !resolvedParentWithinRoot(rootPath, dst) {
			item.Status = "failed"
			failedCount++
			continue
		}

		if _, statErr := os.Stat(src); statErr != nil {
			item.Status = "failed"
			failedCount++
			continue
		}
		if _, statErr := os.Stat(dst); statErr == nil && src != dst {
			item.Status = "failed"
			failedCount++
			continue
		}

		if err := moveFile(src, dst); err != nil {
			item.Status = "failed"
			failedCount++
			continue
		}

		item.Status = "success"
		processed++
		if progressFn != nil {
			progressFn(processed, fmt.Sprintf("Renamed %s", item.Kind))
		}

		if item.Kind == "video" {
			var mediaItemID int64
			err := s.db.QueryRowContext(ctx, `SELECT id FROM media_items WHERE source_id=? AND relative_path=?`, sourceID, item.CurrentPath).Scan(&mediaItemID)
			if err == nil {
				_, _ = s.db.ExecContext(ctx, `UPDATE media_items SET relative_path=?,scan_fingerprint='' WHERE id=?`, item.PlannedPath, mediaItemID)
				s.recordSidecars(ctx, mediaItemID, rootPath, item.PlannedPath)
			}
		}

		var targetMediaID any = nil
		if plan.MediaItemID != nil {
			targetMediaID = *plan.MediaItemID
		}
		detail := fmt.Sprintf("Renamed from %s to %s", item.CurrentPath, item.PlannedPath)
		_, _ = s.db.ExecContext(ctx,
			`INSERT INTO audit_entries(action, media_item_id, target_path, detail, outcome, recoverability) VALUES (?, ?, ?, ?, ?, ?)`,
			"file_rename", targetMediaID, item.PlannedPath, detail, "success", "none")
	}

	newState := "applied"
	if failedCount > 0 {
		newState = "partial"
	}

	itemsBytes, _ := json.Marshal(plan.Items)
	_, _ = s.db.ExecContext(ctx,
		`UPDATE rename_plans SET state=?, items_json=?, applied_at=datetime('now') WHERE id=?`,
		newState, string(itemsBytes), planID)

	return nil
}

func (s *Service) registerRenameJobHandler() {
	s.jobs.RegisterHandler("rename_execute", func(ctx context.Context, job jobs.Job, progress func(int, string)) error {
		var payload struct {
			PlanID string `json:"planId"`
		}
		if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil {
			return err
		}
		return s.ApplyRenamePlan(ctx, payload.PlanID, progress)
	})
}
