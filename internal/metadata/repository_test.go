package metadata

import (
	"path/filepath"
	"testing"

	"github.com/mediagrap/mediagrap/internal/platform/database"
)

func newTestMetadataRepo(t *testing.T) Repository {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "meta_test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := database.Migrate(t.Context(), db); err != nil {
		t.Fatalf("migrate db: %v", err)
	}
	// Insert dummy source and media item for FK constraints
	res, err := db.ExecContext(t.Context(), `INSERT INTO sources(name, root_path) VALUES('Source', '/media')`)
	if err != nil {
		t.Fatalf("insert source: %v", err)
	}
	sourceID, _ := res.LastInsertId()
	res, err = db.ExecContext(t.Context(), `INSERT INTO media_items(source_id, relative_path, title_hint, file_size, modified_at) VALUES(?, 'movie.mkv', 'Movie', 1000, datetime('now'))`, sourceID)
	if err != nil {
		t.Fatalf("insert item: %v", err)
	}
	res, err = db.ExecContext(t.Context(), `INSERT INTO tv_shows(source_id, relative_path, title_hint) VALUES(?, 'Show', 'Show Title')`, sourceID)
	if err != nil {
		t.Fatalf("insert tv show: %v", err)
	}
	return NewRepository(db)
}

func TestMetadataRepositoryRecordCRUD(t *testing.T) {
	repo := newTestMetadataRepo(t)
	ctx := t.Context()

	year := 2024
	rating := 8.5
	record := Record{
		MediaItemID:   1,
		Provider:      "tmdb",
		ProviderID:    "12345",
		Title:         "Inception",
		OriginalTitle: "Inception",
		Year:          &year,
		Overview:      "Dream within a dream",
		Rating:        &rating,
		Genres:        []string{"Sci-Fi", "Action"},
		Directors:     []string{"Christopher Nolan"},
		Cast:          []Person{{Name: "Leonardo DiCaprio", Role: "Cobb"}},
	}

	if err := repo.SaveRecord(ctx, record); err != nil {
		t.Fatalf("SaveRecord failed: %v", err)
	}

	got, err := repo.GetRecord(ctx, 1)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}
	if got.Title != "Inception" || *got.Year != 2024 || *got.Rating != 8.5 || len(got.Genres) != 2 || len(got.Directors) != 1 || len(got.Cast) != 1 {
		t.Fatalf("unexpected record retrieved: %+v", got)
	}
}

func TestMetadataRepositoryTVRecordCRUD(t *testing.T) {
	repo := newTestMetadataRepo(t)
	ctx := t.Context()

	year := 2023
	rating := 9.0
	votes := 10000
	tvRecord := TVRecord{
		ShowID:        1,
		Provider:      "tmdb",
		ProviderID:    "67890",
		TVDBID:        "81189",
		IMDbID:        "tt0903747",
		Title:         "Succession",
		OriginalTitle: "Succession",
		Year:          &year,
		Overview:      "Corporate drama",
		Rating:        &rating,
		Votes:         &votes,
		Status:        "Ended",
		Network:       "HBO",
		Genres:        []string{"Drama"},
		Cast:          []Person{{Name: "Brian Cox", Role: "Logan Roy"}},
		Episodes:      []TVEpisodeDetails{{SeasonNumber: 1, EpisodeNumber: 1, Title: "Celebration"}},
	}

	if err := repo.SaveTVRecord(ctx, tvRecord); err != nil {
		t.Fatalf("SaveTVRecord failed: %v", err)
	}

	got, err := repo.GetTVRecord(ctx, 1)
	if err != nil {
		t.Fatalf("GetTVRecord failed: %v", err)
	}
	if got.Title != "Succession" || got.TVDBID != "81189" || got.IMDbID != "tt0903747" || got.Network != "HBO" || len(got.Episodes) != 1 || got.Episodes[0].Title != "Celebration" {
		t.Fatalf("unexpected tv record: %+v", got)
	}
}

func TestMetadataRepositoryTVArtworkCandidatesAndPlan(t *testing.T) {
	repo := newTestMetadataRepo(t)
	ctx := t.Context()
	season := 1
	candidate := TVArtworkCandidate{ID: "fanart:tv:1:season:1:season_poster:7", ShowID: 1, Scope: "season", SeasonNumber: &season, Provider: "fanart.tv", ProviderAssetID: "7", Kind: "season_poster", SourceURL: "https://assets.fanart.tv/fanart/tv/1/poster.jpg", PreviewURL: "https://assets.fanart.tv/preview/tv/1/poster.jpg", Language: "en", Likes: 4, Width: 1000, Height: 1426, MimeType: "image/jpeg"}
	if err := repo.ReplaceTVArtworkCandidates(ctx, 1, "season", &season, []TVArtworkCandidate{candidate}); err != nil {
		t.Fatal(err)
	}
	candidates, err := repo.ListTVArtworkCandidates(ctx, 1, "season", &season)
	if err != nil || len(candidates) != 1 || candidates[0].SeasonNumber == nil || *candidates[0].SeasonNumber != 1 {
		t.Fatalf("unexpected candidates: %#v, err=%v", candidates, err)
	}
	asset := TVArtworkAsset{ArtworkAsset: ArtworkAsset{Kind: candidate.Kind, CandidateID: candidate.ID, Provider: candidate.Provider, ProviderAssetID: candidate.ProviderAssetID, SourceURL: candidate.SourceURL, PreviewURL: candidate.PreviewURL, Language: candidate.Language, Likes: candidate.Likes, Width: candidate.Width, Height: candidate.Height, MimeType: candidate.MimeType, TargetPath: "/media/Show/season01-poster.jpg"}, Scope: "season", SeasonNumber: &season}
	plan := TVArtworkPlan{ID: "plan-1", ShowID: 1, State: "previewed", Assets: []TVArtworkAsset{asset}}
	if err := repo.SaveTVArtworkPlan(ctx, plan); err != nil {
		t.Fatal(err)
	}
	saved, err := repo.GetTVArtworkPlan(ctx, plan.ID)
	if err != nil || len(saved.Assets) != 1 || saved.Assets[0].Kind != "season_poster" {
		t.Fatalf("unexpected plan: %#v, err=%v", saved, err)
	}
	claimed, err := repo.ClaimTVArtworkPlan(ctx, plan.ID)
	if err != nil || !claimed {
		t.Fatalf("expected pending plan to be claimed, claimed=%t err=%v", claimed, err)
	}
	claimed, err = repo.ClaimTVArtworkPlan(ctx, plan.ID)
	if err != nil || claimed {
		t.Fatalf("expected queued plan not to be claimed twice, claimed=%t err=%v", claimed, err)
	}
	saved, err = repo.GetTVArtworkPlan(ctx, plan.ID)
	if err != nil || saved.State != "queued" {
		t.Fatalf("expected queued plan, got %#v err=%v", saved, err)
	}
}

func TestMetadataRepositoryPlansAndAudit(t *testing.T) {
	repo := newTestMetadataRepo(t)
	ctx := t.Context()

	// Write plan
	writePlan := WritePlan{
		ID:          "plan-1",
		MediaItemID: 1,
		TargetPath:  "/media/movie.nfo",
		Content:     "<movie><title>Test</title></movie>",
		State:       "previewed",
	}
	if err := repo.SaveWritePlan(ctx, writePlan); err != nil {
		t.Fatalf("SaveWritePlan failed: %v", err)
	}

	planGot, err := repo.GetWritePlan(ctx, "plan-1")
	if err != nil || planGot.Content != writePlan.Content {
		t.Fatalf("GetWritePlan failed: %v, got %+v", err, planGot)
	}

	if err := repo.UpdateWritePlanState(ctx, "plan-1", "applied"); err != nil {
		t.Fatalf("UpdateWritePlanState failed: %v", err)
	}

	// Artwork plan
	artPlan := ArtworkPlan{
		ID:          "art-1",
		MediaItemID: 1,
		State:       "previewed",
		Assets: []ArtworkAsset{
			{Kind: "poster", SourceURL: "https://image.tmdb.org/t/p/original/poster.jpg", TargetPath: "/media/poster.jpg"},
		},
	}
	if err := repo.SaveArtworkPlan(ctx, artPlan); err != nil {
		t.Fatalf("SaveArtworkPlan failed: %v", err)
	}

	artGot, err := repo.GetArtworkPlan(ctx, "art-1")
	if err != nil || len(artGot.Assets) != 1 || artGot.Assets[0].Kind != "poster" {
		t.Fatalf("GetArtworkPlan failed: %v, got %+v", err, artGot)
	}

	// Audit entry
	if err := repo.InsertAuditEntry(ctx, "test.action", 1, "/media/movie.nfo", "Applied successfully"); err != nil {
		t.Fatalf("InsertAuditEntry failed: %v", err)
	}
}
