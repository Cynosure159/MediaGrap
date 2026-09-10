package metadata

import (
	"context"
	"errors"
	"github.com/mediagrap/mediagrap/internal/artwork"
	"os"
)

// ScrapeTVMetadata updates only the database draft. File output is deliberately
// a separate, approved plan; automation must never call the AndWrite helpers.
func (s *Service) ScrapeTVMetadata(ctx context.Context, showID int64, season int, episode *int) (TVRecord, error) {
	provider, ok := s.provider.(TVProvider)
	if !ok {
		return TVRecord{}, errors.New("TV provider unavailable")
	}
	record, err := s.TVRecord(ctx, showID)
	if err != nil {
		return TVRecord{}, err
	}
	if record.ProviderID == "" {
		return TVRecord{}, errors.New("match TV show first")
	}
	remote, err := provider.TVSeason(ctx, record.ProviderID, season)
	if err != nil {
		return TVRecord{}, err
	}
	if episode != nil {
		selected := []TVEpisodeDetails{}
		for _, item := range remote {
			if item.SeasonNumber == season && item.EpisodeNumber == *episode {
				selected = append(selected, item)
			}
		}
		if len(selected) == 0 {
			return TVRecord{}, errors.New("episode unavailable")
		}
		remote = selected
	}
	record.Episodes = mergeTVEpisodes(record.Episodes, remote, func(item TVEpisodeDetails) bool {
		return item.SeasonNumber == season && (episode == nil || item.EpisodeNumber == *episode)
	})
	return s.SaveTV(ctx, record)
}

// AutomationArtwork stages validated provider content outside the media tree.
// The safe plan engine owns publication under its rooted filesystem handle.
func (s *Service) AutomationArtwork(ctx context.Context, provider, url, mime string) ([]byte, error) {
	if err := validateArtworkSource(provider, url); err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp("", "mediagrap-artwork-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	s.artworkMu.RLock()
	client := s.artworkClient
	s.artworkMu.RUnlock()
	path, err := artwork.DownloadImage(ctx, client, url, directory, mime)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}
