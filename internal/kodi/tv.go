package kodi

import "github.com/mediagrap/mediagrap/internal/nfo"

type TVShow = nfo.TVShow
type Episode = nfo.Episode

var ParseTVShow = nfo.ParseTVShow
var WriteTVShow = nfo.WriteTVShow
var ParseEpisode = nfo.ParseEpisode
var WriteEpisode = nfo.WriteEpisode
var WriteSeason = nfo.WriteSeason
