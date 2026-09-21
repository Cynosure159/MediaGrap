package library

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultNamingPattern = "${title} (${year})"
const maxFFprobeOutputBytes = 8 << 20

var (
	namingTokenPattern  = regexp.MustCompile(`\$\{([A-Za-z][A-Za-z0-9]*)\}`)
	unsafeFilenameChars = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	auditExtensions     = map[string]string{
		".nfo": "nfo", ".jpg": "image", ".jpeg": "image", ".png": "image", ".webp": "image",
		".srt": "subtitle", ".ass": "subtitle", ".ssa": "subtitle", ".sub": "subtitle", ".idx": "subtitle",
	}
)

type MediaInspection struct {
	ProbeStatus         string           `json:"probeStatus"`
	ProbeError          string           `json:"probeError,omitempty"`
	Cached              bool             `json:"cached"`
	ProbedAt            string           `json:"probedAt,omitempty"`
	Format              ProbeFormat      `json:"format"`
	Video               []VideoStream    `json:"video"`
	Audio               []AudioStream    `json:"audio"`
	Subtitles           []SubtitleStream `json:"subtitles"`
	Files               []FileAuditEntry `json:"files"`
	SupplementalFiles   []FileAuditEntry `json:"supplementalFiles,omitempty"`
	SupplementalStatus  string           `json:"supplementalStatus,omitempty"`
	SupplementalWarning string           `json:"supplementalWarning,omitempty"`
}

type ProbeFormat struct {
	Name            string  `json:"name"`
	DurationSeconds float64 `json:"durationSeconds"`
	BitRate         int64   `json:"bitRate"`
}

type VideoStream struct {
	Index       int    `json:"index"`
	Codec       string `json:"codec"`
	Profile     string `json:"profile,omitempty"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	PixelFormat string `json:"pixelFormat,omitempty"`
	BitDepth    int    `json:"bitDepth,omitempty"`
	HDR         string `json:"hdr,omitempty"`
	Language    string `json:"language,omitempty"`
}

type AudioStream struct {
	Index         int    `json:"index"`
	Codec         string `json:"codec"`
	Channels      int    `json:"channels"`
	ChannelLayout string `json:"channelLayout,omitempty"`
	Language      string `json:"language,omitempty"`
	Title         string `json:"title,omitempty"`
}

type SubtitleStream struct {
	Index    int    `json:"index"`
	Codec    string `json:"codec"`
	Language string `json:"language,omitempty"`
	Title    string `json:"title,omitempty"`
}

type FileAuditEntry struct {
	RelativePath string   `json:"relativePath"`
	Kind         string   `json:"kind"`
	Size         int64    `json:"size"`
	MIMEType     string   `json:"mimeType"`
	ModifiedAt   string   `json:"modifiedAt"`
	Permissions  string   `json:"permissions"`
	Writable     bool     `json:"writable"`
	Regular      bool     `json:"regular"`
	Symlink      bool     `json:"symlink"`
	Valid        bool     `json:"valid"`
	Warnings     []string `json:"warnings"`
}

type NamingValues struct {
	Title         string
	OriginalTitle string
	Year          *int
}

type NamingPreview struct {
	Pattern  string              `json:"pattern"`
	ReadOnly bool                `json:"readOnly"`
	Items    []NamingPreviewItem `json:"items"`
	Warnings []string            `json:"warnings"`
}

type NamingPreviewItem struct {
	Kind        string `json:"kind"`
	CurrentPath string `json:"currentPath"`
	PlannedPath string `json:"plannedPath"`
	Operation   string `json:"operation"`
	Conflict    bool   `json:"conflict"`
}

type mediaProber interface {
	Probe(context.Context, string) (MediaInspection, error)
}

type ffprobeRunner struct {
	path string
}

func (p ffprobeRunner) Probe(ctx context.Context, mediaPath string) (MediaInspection, error) {
	if strings.TrimSpace(p.path) == "" {
		return MediaInspection{}, errors.New("ffprobe is disabled")
	}
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	command := exec.CommandContext(probeCtx, p.path, "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", mediaPath)
	command.Stderr = io.Discard
	stdout, err := command.StdoutPipe()
	if err != nil {
		return MediaInspection{}, fmt.Errorf("open ffprobe output: %w", err)
	}
	if err := command.Start(); err != nil {
		if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
			return MediaInspection{}, errors.New("ffprobe is not installed")
		}
		return MediaInspection{}, fmt.Errorf("start ffprobe: %w", err)
	}
	output, readErr := io.ReadAll(io.LimitReader(stdout, maxFFprobeOutputBytes+1))
	if len(output) > maxFFprobeOutputBytes {
		_ = command.Process.Kill()
		_ = command.Wait()
		return MediaInspection{}, errors.New("ffprobe output exceeds the safety limit")
	}
	if readErr != nil {
		_ = command.Process.Kill()
		_ = command.Wait()
		return MediaInspection{}, fmt.Errorf("read ffprobe output: %w", readErr)
	}
	if err := command.Wait(); err != nil {
		if errors.Is(probeCtx.Err(), context.DeadlineExceeded) {
			return MediaInspection{}, errors.New("ffprobe timed out")
		}
		return MediaInspection{}, fmt.Errorf("ffprobe failed: %w", err)
	}
	return parseFFprobe(output)
}

type ffprobeDocument struct {
	Format struct {
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
		BitRate    string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		Index            int               `json:"index"`
		CodecType        string            `json:"codec_type"`
		CodecName        string            `json:"codec_name"`
		Profile          string            `json:"profile"`
		Width            int               `json:"width"`
		Height           int               `json:"height"`
		PixelFormat      string            `json:"pix_fmt"`
		BitsPerRawSample string            `json:"bits_per_raw_sample"`
		ColorTransfer    string            `json:"color_transfer"`
		Channels         int               `json:"channels"`
		ChannelLayout    string            `json:"channel_layout"`
		Tags             map[string]string `json:"tags"`
		SideData         []struct {
			Type string `json:"side_data_type"`
		} `json:"side_data_list"`
	} `json:"streams"`
}

func parseFFprobe(data []byte) (MediaInspection, error) {
	var document ffprobeDocument
	if err := json.Unmarshal(data, &document); err != nil {
		return MediaInspection{}, fmt.Errorf("decode ffprobe output: %w", err)
	}
	inspection := MediaInspection{ProbeStatus: "ready", Video: []VideoStream{}, Audio: []AudioStream{}, Subtitles: []SubtitleStream{}, Files: []FileAuditEntry{}}
	inspection.Format.Name = document.Format.FormatName
	inspection.Format.DurationSeconds, _ = strconv.ParseFloat(document.Format.Duration, 64)
	inspection.Format.BitRate, _ = strconv.ParseInt(document.Format.BitRate, 10, 64)
	for _, stream := range document.Streams {
		language := normalizedTag(stream.Tags, "language")
		title := normalizedTag(stream.Tags, "title")
		switch stream.CodecType {
		case "video":
			bitDepth, _ := strconv.Atoi(stream.BitsPerRawSample)
			if bitDepth == 0 && strings.Contains(stream.PixelFormat, "10") {
				bitDepth = 10
			}
			inspection.Video = append(inspection.Video, VideoStream{Index: stream.Index, Codec: stream.CodecName, Profile: stream.Profile, Width: stream.Width, Height: stream.Height, PixelFormat: stream.PixelFormat, BitDepth: bitDepth, HDR: detectHDR(stream.ColorTransfer, stream.SideData), Language: language})
		case "audio":
			inspection.Audio = append(inspection.Audio, AudioStream{Index: stream.Index, Codec: stream.CodecName, Channels: stream.Channels, ChannelLayout: stream.ChannelLayout, Language: language, Title: title})
		case "subtitle":
			inspection.Subtitles = append(inspection.Subtitles, SubtitleStream{Index: stream.Index, Codec: stream.CodecName, Language: language, Title: title})
		}
	}
	return inspection, nil
}

func normalizedTag(tags map[string]string, key string) string {
	for candidate, value := range tags {
		if strings.EqualFold(candidate, key) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func detectHDR(transfer string, sideData []struct {
	Type string `json:"side_data_type"`
}) string {
	for _, data := range sideData {
		if strings.Contains(strings.ToLower(data.Type), "dovi") || strings.Contains(strings.ToLower(data.Type), "dolby vision") {
			return "Dolby Vision"
		}
	}
	switch strings.ToLower(transfer) {
	case "smpte2084":
		return "HDR10"
	case "arib-std-b67":
		return "HLG"
	default:
		return ""
	}
}

func (s *Service) SetFFprobePath(path string) {
	s.prober = ffprobeRunner{path: strings.TrimSpace(path)}
}

func (s *Service) InspectMedia(ctx context.Context, id int64) (MediaInspection, error) {
	location, err := s.LocateMedia(ctx, id)
	if err != nil {
		return MediaInspection{}, err
	}
	root := s.sourceRoot(ctx, location.Item.SourceID)
	if !resolvedParentWithinRoot(root, location.AbsolutePath) {
		return MediaInspection{}, errors.New("media path escapes its configured root through a symlink")
	}
	supplemental := s.auditSupplementalFiles(ctx, location)
	if err := ctx.Err(); err != nil {
		return MediaInspection{}, err
	}
	files := s.auditFiles(ctx, location)
	mainInfo, err := os.Lstat(location.AbsolutePath)
	if err != nil {
		return MediaInspection{ProbeStatus: "failed", ProbeError: "media file is unavailable", Files: files, SupplementalFiles: supplemental.files, SupplementalStatus: supplemental.status, SupplementalWarning: supplemental.warning, Video: []VideoStream{}, Audio: []AudioStream{}, Subtitles: []SubtitleStream{}}, nil
	}
	if mainInfo.Mode()&os.ModeSymlink != 0 || !mainInfo.Mode().IsRegular() {
		return MediaInspection{ProbeStatus: "failed", ProbeError: "media file is not a regular file", Files: files, SupplementalFiles: supplemental.files, SupplementalStatus: supplemental.status, SupplementalWarning: supplemental.warning, Video: []VideoStream{}, Audio: []AudioStream{}, Subtitles: []SubtitleStream{}}, nil
	}
	lockValue, _ := s.inspectionLocks.LoadOrStore(id, &sync.Mutex{})
	lock := lockValue.(*sync.Mutex)
	lock.Lock()
	defer lock.Unlock()
	mainInfo, err = os.Lstat(location.AbsolutePath)
	if err != nil || mainInfo.Mode()&os.ModeSymlink != 0 || !mainInfo.Mode().IsRegular() {
		return MediaInspection{ProbeStatus: "failed", ProbeError: "media file changed before inspection", Files: files, SupplementalFiles: supplemental.files, SupplementalStatus: supplemental.status, SupplementalWarning: supplemental.warning, Video: []VideoStream{}, Audio: []AudioStream{}, Subtitles: []SubtitleStream{}}, nil
	}
	modifiedAt := mainInfo.ModTime().UTC().Format(time.RFC3339Nano)
	if cached, ok := s.cachedInspection(ctx, id, mainInfo.Size(), modifiedAt); ok {
		cached.Cached = true
		cached.Files = files
		cached.SupplementalFiles = supplemental.files
		cached.SupplementalStatus = supplemental.status
		cached.SupplementalWarning = supplemental.warning
		return cached, nil
	}
	probeStarted := time.Now()
	s.logger.Info("media probe started", "media_item_id", id)
	inspection, probeErr := s.prober.Probe(ctx, location.AbsolutePath)
	if probeErr != nil {
		s.logger.Warn("media probe failed", "media_item_id", id, "duration", time.Since(probeStarted), "error", probeErr)
		return MediaInspection{ProbeStatus: "unavailable", ProbeError: probeErr.Error(), Files: files, SupplementalFiles: supplemental.files, SupplementalStatus: supplemental.status, SupplementalWarning: supplemental.warning, Video: []VideoStream{}, Audio: []AudioStream{}, Subtitles: []SubtitleStream{}}, nil
	}
	inspection.ProbedAt = time.Now().UTC().Format(time.RFC3339)
	inspection.Files = files
	inspection.SupplementalFiles = supplemental.files
	inspection.SupplementalStatus = supplemental.status
	inspection.SupplementalWarning = supplemental.warning
	afterProbe, statErr := os.Lstat(location.AbsolutePath)
	if statErr != nil || afterProbe.Size() != mainInfo.Size() || !afterProbe.ModTime().Equal(mainInfo.ModTime()) {
		return MediaInspection{ProbeStatus: "failed", ProbeError: "media file changed during inspection", Files: files, SupplementalFiles: supplemental.files, SupplementalStatus: supplemental.status, SupplementalWarning: supplemental.warning, Video: []VideoStream{}, Audio: []AudioStream{}, Subtitles: []SubtitleStream{}}, nil
	}
	if err := s.storeInspection(ctx, id, mainInfo.Size(), modifiedAt, inspection); err != nil {
		s.logger.Warn("media probe cache write failed", "media_item_id", id, "error", err)
	}
	s.logger.Info("media probe completed", "media_item_id", id, "duration", time.Since(probeStarted), "video_stream_count", len(inspection.Video), "audio_stream_count", len(inspection.Audio), "subtitle_stream_count", len(inspection.Subtitles))
	return inspection, nil
}

func (s *Service) cachedInspection(ctx context.Context, id, size int64, modifiedAt string) (MediaInspection, bool) {
	var raw, probedAt string
	err := s.db.QueryRowContext(ctx, `SELECT probe_json, probed_at FROM media_probe_cache WHERE media_item_id=? AND file_size=? AND modified_at=?`, id, size, modifiedAt).Scan(&raw, &probedAt)
	if err != nil {
		return MediaInspection{}, false
	}
	var inspection MediaInspection
	if json.Unmarshal([]byte(raw), &inspection) != nil {
		return MediaInspection{}, false
	}
	inspection.ProbedAt = probedAt
	return inspection, true
}

func (s *Service) storeInspection(ctx context.Context, id, size int64, modifiedAt string, inspection MediaInspection) error {
	copyForCache := inspection
	copyForCache.Files = nil
	copyForCache.SupplementalFiles = nil
	copyForCache.SupplementalStatus = ""
	copyForCache.SupplementalWarning = ""
	copyForCache.Cached = false
	raw, err := json.Marshal(copyForCache)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO media_probe_cache(media_item_id,file_size,modified_at,probe_json,probed_at) VALUES(?,?,?,?,?) ON CONFLICT(media_item_id) DO UPDATE SET file_size=excluded.file_size,modified_at=excluded.modified_at,probe_json=excluded.probe_json,probed_at=excluded.probed_at`, id, size, modifiedAt, string(raw), inspection.ProbedAt)
	return err
}

func (s *Service) auditFiles(ctx context.Context, location MediaLocation) []FileAuditEntry {
	root := s.sourceRoot(ctx, location.Item.SourceID)
	paths := []struct{ path, kind string }{{location.AbsolutePath, "video"}}
	directory := filepath.Dir(location.AbsolutePath)
	mediaBase := strings.TrimSuffix(filepath.Base(location.AbsolutePath), filepath.Ext(location.AbsolutePath))
	entries, _ := os.ReadDir(directory)
	for _, entry := range entries {
		extension := strings.ToLower(filepath.Ext(entry.Name()))
		kind, supported := auditExtensions[extension]
		if !supported {
			continue
		}
		base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		isNamedForMedia := base == mediaBase || strings.HasPrefix(base, mediaBase+".")
		isDirectorySidecar := directoryHasOneVideo(directory) && isStandardDirectorySidecar(entry.Name())
		if isNamedForMedia || isDirectorySidecar {
			paths = append(paths, struct{ path, kind string }{filepath.Join(directory, entry.Name()), kind})
		}
	}
	result := make([]FileAuditEntry, 0, len(paths))
	seen := make(map[string]struct{})
	for _, candidate := range paths {
		if _, ok := seen[candidate.path]; ok {
			continue
		}
		seen[candidate.path] = struct{}{}
		result = append(result, auditFile(root, candidate.path, candidate.kind))
	}
	sidecars := result[1:]
	sort.SliceStable(sidecars, func(i, j int) bool { return sidecars[i].RelativePath < sidecars[j].RelativePath })
	return result
}

func resolvedParentWithinRoot(root, path string) bool {
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return false
	}
	resolvedParent, err := filepath.EvalSymlinks(filepath.Dir(path))
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(resolvedRoot, resolvedParent)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func isStandardDirectorySidecar(name string) bool {
	base := strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
	return base == "movie" || base == "poster" || base == "fanart" || base == "folder" || base == "cover" || base == "clearlogo" || base == "clearart" || base == "discart" || base == "banner" || base == "landscape"
}

func auditFile(root, path, kind string) FileAuditEntry {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		relative = filepath.Base(path)
	}
	entry := FileAuditEntry{RelativePath: relative, Kind: kind, Valid: true, Warnings: []string{}}
	info, err := os.Lstat(path)
	if err != nil {
		entry.Valid = false
		entry.Warnings = append(entry.Warnings, "missing")
		return entry
	}
	entry.Size = info.Size()
	entry.ModifiedAt = info.ModTime().UTC().Format(time.RFC3339)
	entry.Permissions = info.Mode().Perm().String()
	entry.Symlink = info.Mode()&os.ModeSymlink != 0
	entry.Regular = info.Mode().IsRegular()
	entry.Writable = info.Mode().Perm()&0o222 != 0
	extensionMIME := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if entry.Symlink {
		entry.Valid = false
		entry.Warnings = append(entry.Warnings, "symlink")
		return entry
	}
	if !entry.Regular {
		entry.Valid = false
		entry.Warnings = append(entry.Warnings, "not_regular")
		return entry
	}
	if !entry.Writable {
		entry.Warnings = append(entry.Warnings, "read_only")
	}
	file, openErr := os.Open(path)
	if openErr != nil {
		entry.Valid = false
		entry.Warnings = append(entry.Warnings, "unreadable")
		return entry
	}
	defer file.Close()
	header := make([]byte, 512)
	read, _ := file.Read(header)
	detected := http.DetectContentType(header[:read])
	entry.MIMEType = detected
	if detected == "application/octet-stream" && extensionMIME != "" {
		entry.MIMEType = extensionMIME
	}
	if kind == "image" && !strings.HasPrefix(detected, "image/") {
		entry.Valid = false
		entry.Warnings = append(entry.Warnings, "invalid_mime")
	}
	if kind == "nfo" {
		if _, seekErr := file.Seek(0, io.SeekStart); seekErr == nil {
			decoder := xml.NewDecoder(io.LimitReader(file, 4<<20))
			for {
				if _, decodeErr := decoder.Token(); decodeErr != nil {
					if !errors.Is(decodeErr, io.EOF) {
						entry.Valid = false
						entry.Warnings = append(entry.Warnings, "invalid_xml")
					}
					break
				}
			}
		}
	}
	return entry
}

func (s *Service) PreviewNaming(ctx context.Context, id int64, pattern string, values NamingValues) (NamingPreview, error) {
	location, err := s.LocateMedia(ctx, id)
	if err != nil {
		return NamingPreview{}, err
	}
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		pattern = defaultNamingPattern
	}
	if strings.ContainsAny(pattern, `/\\`) {
		return NamingPreview{}, errors.New("naming pattern must produce a filename, not a path")
	}
	inspection, err := s.InspectMedia(ctx, id)
	if err != nil {
		return NamingPreview{}, err
	}
	videoCodec, audioCodec, resolution := "", "", ""
	if len(inspection.Video) > 0 {
		videoCodec = strings.ToUpper(inspection.Video[0].Codec)
		resolution = resolutionLabel(inspection.Video[0].Width, inspection.Video[0].Height)
	}
	if len(inspection.Audio) > 0 {
		audioCodec = strings.ToUpper(inspection.Audio[0].Codec)
	}
	year := ""
	if values.Year != nil {
		year = strconv.Itoa(*values.Year)
	}
	tokens := map[string]string{"title": values.Title, "originalTitle": values.OriginalTitle, "year": year, "resolution": resolution, "videoCodec": videoCodec, "audioCodec": audioCodec}
	unknown := ""
	unavailable := ""
	stem := namingTokenPattern.ReplaceAllStringFunc(pattern, func(token string) string {
		name := namingTokenPattern.FindStringSubmatch(token)[1]
		value, ok := tokens[name]
		if !ok {
			unknown = name
			return token
		}
		if value == "" {
			unavailable = name
		}
		return value
	})
	if unknown != "" {
		return NamingPreview{}, fmt.Errorf("unsupported naming token %q", unknown)
	}
	if unavailable != "" {
		return NamingPreview{}, fmt.Errorf("naming token %q has no available value", unavailable)
	}
	stem = sanitizeFilename(stem)
	if stem == "" {
		return NamingPreview{}, errors.New("naming pattern produced an empty filename")
	}
	preview := NamingPreview{Pattern: pattern, ReadOnly: true, Items: []NamingPreviewItem{}, Warnings: []string{}}
	currentMediaBase := strings.TrimSuffix(filepath.Base(location.AbsolutePath), filepath.Ext(location.AbsolutePath))
	for _, file := range inspection.Files {
		currentName := filepath.Base(file.RelativePath)
		plannedName := currentName
		if file.Kind == "video" {
			plannedName = stem + filepath.Ext(currentName)
		} else {
			baseWithoutExtension := strings.TrimSuffix(currentName, filepath.Ext(currentName))
			if baseWithoutExtension == currentMediaBase || strings.HasPrefix(baseWithoutExtension, currentMediaBase+".") {
				suffix := strings.TrimPrefix(baseWithoutExtension, currentMediaBase)
				plannedName = stem + suffix + filepath.Ext(currentName)
			}
		}
		plannedPath := filepath.Join(filepath.Dir(file.RelativePath), plannedName)
		operation := "keep"
		conflict := false
		if plannedPath != file.RelativePath {
			operation = "rename"
			target := filepath.Join(filepath.Dir(location.AbsolutePath), plannedName)
			if _, statErr := os.Lstat(target); statErr == nil {
				operation = "conflict"
				conflict = true
			}
		}
		preview.Items = append(preview.Items, NamingPreviewItem{Kind: file.Kind, CurrentPath: file.RelativePath, PlannedPath: plannedPath, Operation: operation, Conflict: conflict})
	}
	return preview, nil
}

func sanitizeFilename(value string) string {
	value = unsafeFilenameChars.ReplaceAllString(value, " ")
	value = strings.Join(strings.Fields(value), " ")
	return strings.Trim(value, " .")
}

func resolutionLabel(width, height int) string {
	switch {
	case width >= 3800 || height >= 2100:
		return "2160p"
	case width >= 1900 || height >= 1060:
		return "1080p"
	case width >= 1200 || height >= 700:
		return "720p"
	case height > 0:
		return strconv.Itoa(height) + "p"
	default:
		return ""
	}
}
