package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mediagrap/mediagrap/internal/artwork"
	"github.com/mediagrap/mediagrap/internal/files"
	"github.com/mediagrap/mediagrap/internal/jobs"
	"github.com/mediagrap/mediagrap/internal/providers/fanart"
)

type TVArtworkCandidate struct {
	ID              string `json:"id"`
	ShowID          int64  `json:"showId"`
	Scope           string `json:"scope"`
	SeasonNumber    *int   `json:"seasonNumber,omitempty"`
	Provider        string `json:"provider"`
	ProviderAssetID string `json:"providerAssetId"`
	Kind            string `json:"kind"`
	SourceURL       string `json:"sourceUrl"`
	PreviewURL      string `json:"previewUrl"`
	Language        string `json:"language"`
	Likes           int    `json:"likes"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	MimeType        string `json:"mimeType"`
	SortOrder       int    `json:"-"`
}

type TVArtworkSelection struct {
	Kind        string `json:"kind"`
	CandidateID string `json:"candidateId"`
}

type TVArtworkAsset struct {
	ArtworkAsset
	Scope        string `json:"scope"`
	SeasonNumber *int   `json:"seasonNumber,omitempty"`
}

type TVArtworkPlan struct {
	ID        string           `json:"id"`
	ShowID    int64            `json:"showId"`
	State     string           `json:"state"`
	CreatedAt string           `json:"createdAt"`
	Assets    []TVArtworkAsset `json:"assets"`
}

func (s *Service) TVArtworkCandidates(ctx context.Context, showID int64, scope string, seasonNumber *int) ([]TVArtworkCandidate, error) {
	if err := validateTVArtworkScope(scope, seasonNumber); err != nil {
		return nil, err
	}
	record, err := s.TVRecord(ctx, showID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(record.TVDBID) == "" {
		return nil, errors.New("TheTVDB ID is unavailable; match the TV show with TMDb or import it from tvshow.nfo first")
	}
	if s.fanart == nil {
		return nil, errors.New("Fanart.tv provider is unavailable")
	}
	s.logger.Info("Fanart.tv TV artwork request started", "show_id", showID, "tvdb_id", record.TVDBID, "scope", scope, "season_number", seasonNumber)
	assets, err := s.fanart.TV(ctx, record.TVDBID)
	if err != nil {
		s.logger.Warn("Fanart.tv TV artwork request failed", "show_id", showID, "tvdb_id", record.TVDBID, "error", err)
		return nil, err
	}
	candidates := make([]TVArtworkCandidate, 0, len(assets))
	for _, asset := range assets {
		assetScope, assetSeason, scopeErr := fanartTVArtworkScope(asset)
		if scopeErr != nil || assetScope != scope || !sameSeason(assetSeason, seasonNumber) || asset.ID == "" || asset.URL == "" {
			continue
		}
		previewURL, previewErr := artwork.FanartPreviewURL(asset.URL)
		if previewErr != nil {
			continue
		}
		seasonToken := "show"
		if assetSeason != nil {
			seasonToken = strconv.Itoa(*assetSeason)
		}
		candidates = append(candidates, TVArtworkCandidate{
			ID: fmt.Sprintf("fanart:tv:%d:%s:%s:%s:%s", showID, scope, seasonToken, asset.Kind, asset.ID), ShowID: showID,
			Scope: scope, SeasonNumber: assetSeason, Provider: "fanart.tv", ProviderAssetID: asset.ID, Kind: asset.Kind,
			SourceURL: asset.URL, PreviewURL: previewURL, Language: asset.Lang, Likes: asset.Likes,
			Width: asset.Width, Height: asset.Height, MimeType: imageMIME(asset.URL),
			SortOrder: len(candidates),
		})
	}
	if err := s.repo.ReplaceTVArtworkCandidates(ctx, showID, scope, seasonNumber, candidates); err != nil {
		return nil, err
	}
	s.logger.Info("Fanart.tv TV artwork request completed", "show_id", showID, "tvdb_id", record.TVDBID, "scope", scope, "season_number", seasonNumber, "candidate_count", len(candidates))
	return s.repo.ListTVArtworkCandidates(ctx, showID, scope, seasonNumber)
}

func (s *Service) CachedTVArtworkCandidates(ctx context.Context, showID int64, scope string, seasonNumber *int) ([]TVArtworkCandidate, error) {
	if err := validateTVArtworkScope(scope, seasonNumber); err != nil {
		return nil, err
	}
	return s.repo.ListTVArtworkCandidates(ctx, showID, scope, seasonNumber)
}

func (s *Service) OpenTVArtworkPreview(ctx context.Context, showID int64, candidateID string) (ArtworkPreview, error) {
	candidate, err := s.repo.GetTVArtworkCandidate(ctx, candidateID)
	if err != nil || candidate.ShowID != showID {
		return ArtworkPreview{}, errors.New("TV artwork candidate not found")
	}
	return s.openArtworkPreview(ctx, candidate.Provider, candidate.PreviewURL)
}

func (s *Service) PreviewTVArtworkSelection(ctx context.Context, showID int64, scope string, seasonNumber *int, selections []TVArtworkSelection, showDirectory string, writable bool) (TVArtworkPlan, error) {
	if !writable {
		return TVArtworkPlan{}, errors.New("the media source is read-only")
	}
	if err := validateTVArtworkScope(scope, seasonNumber); err != nil {
		return TVArtworkPlan{}, err
	}
	if len(selections) == 0 {
		return TVArtworkPlan{}, errors.New("select at least one artwork candidate")
	}
	assets := make([]TVArtworkAsset, 0, len(selections))
	seenKinds := make(map[string]struct{}, len(selections))
	for _, selection := range selections {
		if !supportedTVArtworkKind(selection.Kind) {
			return TVArtworkPlan{}, fmt.Errorf("unsupported TV artwork kind %q", selection.Kind)
		}
		if _, exists := seenKinds[selection.Kind]; exists {
			return TVArtworkPlan{}, fmt.Errorf("multiple candidates selected for %s", selection.Kind)
		}
		seenKinds[selection.Kind] = struct{}{}
		candidate, err := s.repo.GetTVArtworkCandidate(ctx, selection.CandidateID)
		if err != nil || candidate.ShowID != showID || candidate.Scope != scope || !sameSeason(candidate.SeasonNumber, seasonNumber) || candidate.Kind != selection.Kind {
			return TVArtworkPlan{}, fmt.Errorf("TV artwork candidate %q is not available for this selection", selection.CandidateID)
		}
		if err := validateArtworkSource(candidate.Provider, candidate.SourceURL); err != nil {
			return TVArtworkPlan{}, fmt.Errorf("%s image: %w", candidate.Kind, err)
		}
		target, err := tvArtworkTarget(showDirectory, candidate)
		if err != nil {
			return TVArtworkPlan{}, err
		}
		willReplace, conflict, err := files.ValidateTarget(target)
		if err != nil {
			return TVArtworkPlan{}, err
		}
		assets = append(assets, TVArtworkAsset{ArtworkAsset: ArtworkAsset{Kind: candidate.Kind, CandidateID: candidate.ID, Provider: candidate.Provider, ProviderAssetID: candidate.ProviderAssetID, SourceURL: candidate.SourceURL, PreviewURL: candidate.PreviewURL, Language: candidate.Language, Likes: candidate.Likes, Width: candidate.Width, Height: candidate.Height, MimeType: candidate.MimeType, TargetPath: target, Conflict: conflict, WillReplace: willReplace}, Scope: scope, SeasonNumber: seasonNumber})
	}
	plan := TVArtworkPlan{ID: uuid.NewString(), ShowID: showID, State: "previewed", CreatedAt: time.Now().UTC().Format(time.RFC3339), Assets: assets}
	if err := s.repo.SaveTVArtworkPlan(ctx, plan); err != nil {
		return TVArtworkPlan{}, err
	}
	_ = s.repo.InsertTVAuditEntry(ctx, "tv.artwork.preview", showID, showDirectory, "TV artwork download plan created")
	return plan, nil
}

func (s *Service) TVArtworkPlan(ctx context.Context, id string) (TVArtworkPlan, error) {
	return s.repo.GetTVArtworkPlan(ctx, id)
}

func (s *Service) QueueTVArtwork(ctx context.Context, id string, allowed func(string) bool) (TVArtworkPlan, error) {
	if s.jobQueue == nil {
		return TVArtworkPlan{}, errors.New("artwork job service is unavailable")
	}
	plan, err := s.TVArtworkPlan(ctx, id)
	if err != nil {
		return TVArtworkPlan{}, err
	}
	if plan.State != "previewed" {
		return TVArtworkPlan{}, errors.New("TV artwork plan is no longer pending")
	}
	for _, asset := range plan.Assets {
		if err := files.CheckAllowed(asset.TargetPath, allowed); err != nil {
			return TVArtworkPlan{}, errors.New("TV artwork target is outside configured media roots")
		}
		if asset.Conflict {
			_ = s.repo.UpdateTVArtworkPlanState(ctx, id, "conflicted")
			return TVArtworkPlan{}, fmt.Errorf("%s target is not a regular file and cannot be replaced", asset.Kind)
		}
		if err := validateArtworkSource(asset.Provider, asset.SourceURL); err != nil {
			return TVArtworkPlan{}, fmt.Errorf("%s image: %w", asset.Kind, err)
		}
	}
	claimed, err := s.repo.ClaimTVArtworkPlan(ctx, id)
	if err != nil {
		return TVArtworkPlan{}, err
	}
	if !claimed {
		return TVArtworkPlan{}, errors.New("TV artwork plan is no longer pending")
	}
	payload, _ := json.Marshal(map[string]string{"planId": id})
	if _, err := s.jobQueue.QueuePayload(ctx, "tv_artwork_download", nil, payload); err != nil {
		// No worker can have started this plan before the job was inserted.
		// Restore the previewed state so the user can retry after a queue error.
		_ = s.repo.UpdateTVArtworkPlanState(ctx, id, "previewed")
		return TVArtworkPlan{}, err
	}
	return s.TVArtworkPlan(ctx, id)
}

func (s *Service) handleTVArtworkDownloadJob(ctx context.Context, job jobs.Job, updateProgress func(int, string)) error {
	var payload struct {
		PlanID string `json:"planId"`
	}
	if err := json.Unmarshal([]byte(job.Payload), &payload); err != nil || payload.PlanID == "" {
		return errors.New("TV artwork job payload is invalid")
	}
	plan, err := s.TVArtworkPlan(ctx, payload.PlanID)
	if err != nil {
		return err
	}
	if plan.State != "queued" && plan.State != "running" {
		return errors.New("TV artwork plan is not queued")
	}
	if err := s.repo.UpdateTVArtworkPlanState(ctx, plan.ID, "running"); err != nil {
		return err
	}
	_, err = s.applyTVArtworkFiles(ctx, plan, updateProgress)
	if err != nil && job.RetryCount < job.MaxRetries && !errors.Is(err, context.Canceled) {
		if current, getErr := s.TVArtworkPlan(ctx, plan.ID); getErr == nil && current.State == "failed" {
			_ = s.repo.UpdateTVArtworkPlanState(ctx, plan.ID, "queued")
		}
	}
	return err
}

func (s *Service) applyTVArtworkFiles(ctx context.Context, plan TVArtworkPlan, updateProgress func(int, string)) (TVArtworkPlan, error) {
	release, lockErr := files.LockMutation(ctx)
	if lockErr != nil {
		return TVArtworkPlan{}, lockErr
	}
	defer release()
	lockValue, _ := s.artworkLocks.LoadOrStore(fmt.Sprintf("tv:%d", plan.ShowID), &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	s.logger.Info("TV artwork download started", "show_id", plan.ShowID, "asset_count", len(plan.Assets))
	temps := make([]string, 0, len(plan.Assets))
	defer func() {
		for _, temp := range temps {
			_ = os.Remove(temp)
		}
	}()
	s.artworkMu.RLock()
	client := s.artworkClient
	s.artworkMu.RUnlock()
	for index, asset := range plan.Assets {
		updateProgress(index, "Downloading "+asset.Kind)
		if err := validateArtworkSource(asset.Provider, asset.SourceURL); err != nil {
			_ = s.repo.UpdateTVArtworkPlanState(ctx, plan.ID, "failed")
			s.logger.Warn("TV artwork download rejected", "show_id", plan.ShowID, "kind", asset.Kind, "error", err)
			return TVArtworkPlan{}, fmt.Errorf("%s image: %w", asset.Kind, err)
		}
		temp, err := downloadArtwork(ctx, client, asset.ArtworkAsset)
		if err != nil {
			_ = s.repo.UpdateTVArtworkPlanState(ctx, plan.ID, "failed")
			s.logger.Warn("TV artwork download failed", "show_id", plan.ShowID, "kind", asset.Kind, "error", err)
			return TVArtworkPlan{}, err
		}
		temps = append(temps, temp)
	}
	for index, asset := range plan.Assets {
		_, conflict, err := files.ValidateTarget(asset.TargetPath)
		if err != nil || conflict {
			_ = s.repo.UpdateTVArtworkPlanState(ctx, plan.ID, "conflicted")
			s.logger.Warn("TV artwork target changed", "show_id", plan.ShowID, "kind", asset.Kind, "error", err)
			return TVArtworkPlan{}, fmt.Errorf("%s target changed and cannot be replaced", asset.Kind)
		}
		if err := os.Rename(temps[index], asset.TargetPath); err != nil {
			_ = s.repo.UpdateTVArtworkPlanState(ctx, plan.ID, "failed")
			s.logger.Warn("TV artwork atomic replace failed", "show_id", plan.ShowID, "kind", asset.Kind, "error", err)
			return TVArtworkPlan{}, fmt.Errorf("replace TV %s artwork: %w", asset.Kind, err)
		}
		if err := s.repo.SetTVArtworkAsset(ctx, asset, plan.ShowID); err != nil {
			return TVArtworkPlan{}, err
		}
		updateProgress(index+1, "Saved "+asset.Kind)
	}
	if err := s.repo.UpdateTVArtworkPlanState(ctx, plan.ID, "applied"); err != nil {
		return TVArtworkPlan{}, err
	}
	for _, asset := range plan.Assets {
		detail := "TV artwork created atomically"
		if asset.WillReplace {
			detail = "Existing TV artwork replaced atomically"
		}
		_ = s.repo.InsertTVAuditEntry(ctx, "tv.artwork.apply", plan.ShowID, asset.TargetPath, detail)
	}
	s.logger.Info("TV artwork download completed", "show_id", plan.ShowID, "asset_count", len(plan.Assets))
	return s.TVArtworkPlan(ctx, plan.ID)
}

func fanartTVArtworkScope(asset fanart.Asset) (string, *int, error) {
	if strings.HasPrefix(asset.Kind, "season_") {
		season, err := strconv.Atoi(strings.TrimSpace(asset.Season))
		if err != nil || season < 0 {
			return "", nil, errors.New("Fanart.tv season artwork has an invalid season")
		}
		return "season", &season, nil
	}
	return "show", nil, nil
}

func validateTVArtworkScope(scope string, seasonNumber *int) error {
	if scope == "show" && seasonNumber == nil {
		return nil
	}
	if scope == "season" && seasonNumber != nil && *seasonNumber >= 0 {
		return nil
	}
	return errors.New("TV artwork scope must be show or a non-negative season")
}

func sameSeason(left, right *int) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func tvArtworkTarget(showDirectory string, candidate TVArtworkCandidate) (string, error) {
	name := candidate.Kind
	if candidate.Scope == "season" {
		if candidate.SeasonNumber == nil {
			return "", errors.New("season artwork candidate is missing its season")
		}
		name = fmt.Sprintf("season%02d-%s", *candidate.SeasonNumber, strings.TrimPrefix(candidate.Kind, "season_"))
	}
	extension := ".jpg"
	if candidate.MimeType == "image/png" {
		extension = ".png"
	}
	return filepath.Join(showDirectory, name+extension), nil
}

var tvArtworkKinds = map[string]bool{"poster": true, "fanart": true, "clearlogo": true, "logo": true, "clearart": true, "banner": true, "landscape": true, "character": true, "season_poster": true, "season_banner": true, "season_landscape": true}

func supportedTVArtworkKind(kind string) bool { return tvArtworkKinds[kind] }
