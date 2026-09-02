package metadata

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/mediagrap/mediagrap/internal/providers/tmdb"
)

type Candidate = tmdb.Candidate
type Details = tmdb.MovieDetails
type Person = tmdb.Person
type TVDetails = tmdb.TVDetails
type TVEpisodeDetails = tmdb.TVEpisodeDetails

type Provider interface {
	SearchMovies(context.Context, string, *int) ([]Candidate, error)
	Movie(context.Context, string) (Details, error)
}

type TVProvider interface {
	SearchTV(context.Context, string, *int) ([]Candidate, error)
	TV(context.Context, string) (TVDetails, error)
	TVSeason(context.Context, string, int) ([]TVEpisodeDetails, error)
}

type TMDb = tmdb.Client

func NewTMDb(logger *slog.Logger, client *http.Client, apiKey string) *TMDb {
	return tmdb.NewClient(logger, client, apiKey)
}

func NewOutboundClient(proxy string, noProxy ...string) (*http.Client, error) {
	return tmdb.NewOutboundClient(proxy, noProxy...)
}
