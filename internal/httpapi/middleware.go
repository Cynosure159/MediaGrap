package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/mediagrap/mediagrap/internal/auth"
	"github.com/mediagrap/mediagrap/internal/library"
	"github.com/mediagrap/mediagrap/internal/metadata"
	"github.com/mediagrap/mediagrap/internal/settings"
)

// LibraryService defines the subset of library methods needed by httpapi handlers.
type LibraryService interface {
	Allowed(path string) bool
	ListSources(ctx context.Context) ([]library.Source, error)
	CreateSource(ctx context.Context, name, rootPath string) (library.Source, error)
	DeleteSource(ctx context.Context, id int64) error
	QueueScan(ctx context.Context, sourceID int64) (library.Job, error)
	ListJobs(ctx context.Context) ([]library.Job, error)
	Job(ctx context.Context, id int64) (library.Job, error)
	CancelJob(ctx context.Context, id int64) (library.Job, error)
	RetryJob(ctx context.Context, id int64) (library.Job, error)
	JobEventsAfter(ctx context.Context, cursor int64, limit int) ([]library.JobEvent, error)
	ListMedia(ctx context.Context, query string, page, pageSize int) (library.Page, error)
	LocateMedia(ctx context.Context, id int64) (library.MediaLocation, error)
	InspectMedia(ctx context.Context, id int64) (library.MediaInspection, error)
	PreviewNaming(ctx context.Context, id int64, pattern string, values library.NamingValues) (library.NamingPreview, error)
	MetadataSearchHint(item library.MediaItem) (string, *int, string)
	ListTVShows(ctx context.Context, query string) ([]library.TVShow, error)
	TVShow(ctx context.Context, id int64) (library.TVShowDetail, error)
}

// MetadataService defines the subset of metadata methods needed by httpapi handlers.
type MetadataService interface {
	ConfigureTMDb(apiKey, language, proxy string) error
	ConfigureFanart(apiKey, language, proxy string) error
	Search(ctx context.Context, query string, year *int) ([]metadata.Candidate, error)
	Select(ctx context.Context, itemID int64, providerID string) (metadata.Record, error)
	Record(ctx context.Context, itemID int64) (metadata.Record, error)
	Save(ctx context.Context, record metadata.Record) (metadata.Record, error)
	SelectAndWrite(ctx context.Context, itemID int64, providerID, mediaPath string, writable bool, allowed func(string) bool) (metadata.Record, error)
	Preview(ctx context.Context, record metadata.Record, mediaPath string, writable bool) (metadata.WritePlan, error)
	Plan(ctx context.Context, id string) (metadata.WritePlan, error)
	Apply(ctx context.Context, id string, allowed func(string) bool) (metadata.WritePlan, error)
	PreviewArtwork(ctx context.Context, record metadata.Record, mediaPath string, writable bool) (metadata.ArtworkPlan, error)
	ArtworkPlan(ctx context.Context, id string) (metadata.ArtworkPlan, error)
	ApplyArtwork(ctx context.Context, id string, allowed func(string) bool) (metadata.ArtworkPlan, error)
	ArtworkCandidates(ctx context.Context, itemID int64) ([]metadata.ArtworkCandidate, error)
	CachedArtworkCandidates(ctx context.Context, itemID int64) ([]metadata.ArtworkCandidate, error)
	OpenArtworkPreview(ctx context.Context, itemID int64, candidateID string) (metadata.ArtworkPreview, error)
	PreviewArtworkSelection(ctx context.Context, itemID int64, selections []metadata.ArtworkSelection, mediaPath string, writable bool) (metadata.ArtworkPlan, error)
	QueueArtwork(ctx context.Context, id string, allowed func(string) bool) (metadata.ArtworkPlan, error)
	SearchTV(ctx context.Context, query string, year *int) ([]metadata.Candidate, error)
	TVRecord(ctx context.Context, showID int64) (metadata.TVRecord, error)
	SaveTV(ctx context.Context, record metadata.TVRecord) (metadata.TVRecord, error)
	SelectTVAndWrite(ctx context.Context, showID int64, providerID string, seasons []int, inputs []metadata.TVNFOInput, writable bool, allowed func(string) bool) (metadata.TVRecord, error)
	ScrapeTVSeasonAndWrite(ctx context.Context, showID int64, seasonNumber int, inputs []metadata.TVNFOInput, writable bool, allowed func(string) bool) (metadata.TVRecord, error)
	ScrapeTVEpisodeAndWrite(ctx context.Context, showID int64, seasonNumber, episodeNumber int, input metadata.TVNFOInput, writable bool, allowed func(string) bool) (metadata.TVRecord, error)
	PreviewTVNFO(ctx context.Context, record metadata.TVRecord, inputs []metadata.TVNFOInput, writable bool) ([]metadata.WritePlan, error)
	ReadExistingNFO(mediaPath string, itemID int64) (metadata.Record, bool, error)
	ReadExistingTVNFO(showID int64, inputs []metadata.TVNFOInput) (metadata.TVRecord, bool, error)
	HydrateExistingNFO(ctx context.Context, itemID int64, mediaPath string) error
}

// SettingsService defines the subset of settings methods needed by httpapi handlers.
type SettingsService interface {
	View(ctx context.Context) (settings.View, error)
	Update(ctx context.Context, update settings.Update) (settings.Snapshot, error)
	Current(ctx context.Context) (settings.Snapshot, error)
}

// AuthService defines the subset of authentication methods needed by httpapi handlers.
type AuthService interface {
	NeedsSetup(ctx context.Context) (bool, error)
	Setup(ctx context.Context, username, password string) (*auth.Session, error)
	Login(ctx context.Context, username, password string) (*auth.Session, error)
	Logout(ctx context.Context, token string) error
	Authenticate(ctx context.Context, token string) (*auth.Session, error)
}

// Context keys
type contextKey string

const sessionContextKey contextKey = "mediagrap.session"

// sessionFromContext retrieves the authenticated session from the request context.
func sessionFromContext(ctx context.Context) (*auth.Session, bool) {
	session, ok := ctx.Value(sessionContextKey).(*auth.Session)
	return session, ok && session != nil
}

// requireAuth wraps a handler ensuring a valid session cookie exists.
func (s *server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, ok := s.requireSession(w, r, false)
		if !ok {
			return
		}
		ctx := context.WithValue(r.Context(), sessionContextKey, session)
		next(w, r.WithContext(ctx))
	}
}

// requireAuthCSRF wraps a handler ensuring a valid session and valid X-CSRF-Token header.
func (s *server) requireAuthCSRF(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, ok := s.requireSession(w, r, true)
		if !ok {
			return
		}
		ctx := context.WithValue(r.Context(), sessionContextKey, session)
		next(w, r.WithContext(ctx))
	}
}

// loggingMiddleware provides structured HTTP access logging.
func loggingMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)
		duration := time.Since(start)

		if !isStaticAssetPath(r.URL.Path) {
			logger.Debug("HTTP request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration_ms", duration.Milliseconds(),
			)
		}
	})
}

type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *statusResponseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func isStaticAssetPath(path string) bool {
	return path == "/" ||
		path == "/favicon.ico" ||
		len(path) > 8 && path[:8] == "/assets/"
}
