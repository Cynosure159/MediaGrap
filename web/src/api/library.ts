import { request } from "./client";
import type {
  Session,
  Source,
  MediaItem,
  MediaInspection,
  NamingPreview,
  TVShow,
  TVShowDetail,
  TVMetadata,
  TVSelection,
  Job,
  Metadata,
  Candidate,
  WritePlan,
  ArtworkPlan,
  ArtworkCandidate,
  TVArtworkCandidate,
  TVArtworkPlan,
  Settings,
  SettingsUpdate,
  AuditEntry,
  OperationsStatus,
  ConnectionTest,
  RenamePlan,
} from "./types";

export * from "./types";
export { request } from "./client";

// Auth & Setup
export const setupStatus = () =>
  request<{ needsSetup: boolean }>("/api/v1/setup/status");

export const session = () => request<Session>("/api/v1/session");

export const signIn = (username: string, password: string) =>
  request<Session>("/api/v1/session", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });

export const setup = (username: string, password: string) =>
  request<Session>("/api/v1/setup", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });

// Sources
export const sources = () => request<{ items: Source[] }>("/api/v1/sources");

export const addSource = (csrf: string, name: string, rootPath: string) =>
  request<Source>("/api/v1/sources", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-CSRF-Token": csrf,
    },
    body: JSON.stringify({ name, rootPath }),
  });

export const deleteSource = (csrf: string, id: number) =>
  request<void>(`/api/v1/sources/${id}`, {
    method: "DELETE",
    headers: { "X-CSRF-Token": csrf },
  });

export const scanSource = (csrf: string, id: number) =>
  request<Job>(`/api/v1/sources/${id}/scans`, {
    method: "POST",
    headers: { "X-CSRF-Token": csrf },
  });

export const updateSourcePolicy = (
  csrf: string,
  id: number,
  policy: Pick<
    Source,
    "scanMode" | "scheduleEnabled" | "scheduleIntervalMinutes"
  >,
) =>
  request<Source>(`/api/v1/sources/${id}/policy`, {
    method: "PUT",
    headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
    body: JSON.stringify(policy),
  });

// Media (Movies)
export type CatalogOptions = { filter?: string; sort?: string };
export const media = (
  q = "",
  page = 1,
  options: CatalogOptions = {},
  signal?: AbortSignal,
) =>
  request<{ items: MediaItem[]; total: number }>(
    `/api/v1/media?page=${page}&pageSize=50&q=${encodeURIComponent(q)}&filter=${encodeURIComponent(options.filter || "all")}&sort=${encodeURIComponent(options.sort || "title")}`,
    { signal },
  );

export const mediaDetail = (id: number, signal?: AbortSignal) =>
  request<{
    item: MediaItem;
    metadata: Metadata;
    metadataOrigin: "draft" | "nfo" | "empty";
    metadataWarning?: string;
    writable: boolean;
  }>(`/api/v1/media/${id}`, { signal });

export const mediaInspection = (id: number, signal?: AbortSignal) =>
  request<MediaInspection>(`/api/v1/media/${id}/inspection`, { signal });

export const previewMediaNaming = (id: number, pattern: string) =>
  request<NamingPreview>(`/api/v1/media/${id}/naming-preview`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ pattern }),
  });

export const candidates = (id: number, q = "") =>
  request<{ items: Candidate[] }>(
    q
      ? `/api/v1/media/${id}/candidates?q=${encodeURIComponent(q)}`
      : `/api/v1/media/${id}/candidates`,
  );

export const selectCandidate = (
  csrf: string,
  id: number,
  candidateId: string,
) =>
  request<Metadata>(`/api/v1/media/${id}/select`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-CSRF-Token": csrf,
    },
    body: JSON.stringify({ candidateId }),
  });

export const saveMetadata = (
  csrf: string,
  id: number,
  metadataPayload: Metadata,
) =>
  request<Metadata>(`/api/v1/media/${id}/metadata`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-CSRF-Token": csrf,
    },
    body: JSON.stringify(metadataPayload),
  });

export const previewNfo = (
  csrf: string,
  id: number,
  metadataPayload: Metadata,
) =>
  request<WritePlan>(`/api/v1/media/${id}/write-plans`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-CSRF-Token": csrf,
    },
    body: JSON.stringify(metadataPayload),
  });

export const applyNfo = (csrf: string, id: string) =>
  request<WritePlan>(`/api/v1/write-plans/${id}/apply`, {
    method: "POST",
    headers: { "X-CSRF-Token": csrf },
  });

export const previewArtwork = (
  csrf: string,
  id: number,
  metadataPayload: Metadata,
) =>
  request<ArtworkPlan>(`/api/v1/media/${id}/artwork-plans`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-CSRF-Token": csrf,
    },
    body: JSON.stringify(metadataPayload),
  });

export const artworkCandidates = (id: number) =>
  request<{ items: ArtworkCandidate[] }>(
    `/api/v1/media/${id}/artwork-candidates`,
  );

export const scrapeArtworkCandidates = (csrf: string, id: number) =>
  request<{ items: ArtworkCandidate[] }>(
    `/api/v1/media/${id}/artwork-candidates`,
    {
      method: "POST",
      headers: { "X-CSRF-Token": csrf },
    },
  );

export const previewArtworkSelection = (
  csrf: string,
  id: number,
  selections: { kind: ArtworkCandidate["kind"]; candidateId: string }[],
) =>
  request<ArtworkPlan>(`/api/v1/media/${id}/artwork-plans`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
    body: JSON.stringify({ selections }),
  });

export const applyArtwork = (csrf: string, id: string) =>
  request<ArtworkPlan>(`/api/v1/artwork-plans/${id}/apply`, {
    method: "POST",
    headers: { "X-CSRF-Token": csrf },
  });

// TV Shows & Episodes
export const tvShows = (q = "") =>
  request<{ items: TVShow[] }>(`/api/v1/tv/shows?q=${encodeURIComponent(q)}`);

export const tvShowDetail = (id: number, signal?: AbortSignal) =>
  request<TVShowDetail>(`/api/v1/tv/shows/${id}`, { signal });

export const tvShowCandidates = (id: number, q = "") =>
  request<{ items: Candidate[] }>(
    `/api/v1/tv/shows/${id}/candidates?q=${encodeURIComponent(q)}`,
  );

export const selectTVShowCandidate = (
  csrf: string,
  id: number,
  candidateId: string,
) =>
  request<TVMetadata>(`/api/v1/tv/shows/${id}/select`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
    body: JSON.stringify({ candidateId }),
  });

export const scrapeTVSeason = (
  csrf: string,
  showId: number,
  seasonNumber: number,
) =>
  request<TVMetadata>(
    `/api/v1/tv/shows/${showId}/seasons/${seasonNumber}/scrape`,
    {
      method: "POST",
      headers: { "X-CSRF-Token": csrf },
    },
  );

export const scrapeTVEpisode = (
  csrf: string,
  showId: number,
  seasonNumber: number,
  episodeId: number,
) =>
  request<TVMetadata>(
    `/api/v1/tv/shows/${showId}/seasons/${seasonNumber}/episodes/${episodeId}/scrape`,
    {
      method: "POST",
      headers: { "X-CSRF-Token": csrf },
    },
  );

export const previewTVNfoPlans = (csrf: string, id: number) =>
  request<{ items: WritePlan[] }>(`/api/v1/tv/shows/${id}/nfo-plans`, {
    method: "POST",
    headers: { "X-CSRF-Token": csrf },
  });

export const tvArtworkUrl = (showId: number, artworkId: string) =>
  `/api/v1/tv/shows/${showId}/artwork/${encodeURIComponent(artworkId)}`;

export const mediaArtworkUrl = (mediaId: number, relativePath: string) => {
  const bytes = new TextEncoder().encode(relativePath);
  const assetID = btoa(String.fromCharCode(...bytes))
    .replaceAll("+", "-")
    .replaceAll("/", "_")
    .replaceAll("=", "");
  return `/api/v1/media/${mediaId}/artwork/${assetID}`;
};

export const artworkPreviewUrl = (mediaId: number, candidateId: string) =>
  `/api/v1/media/${mediaId}/artwork-preview/${encodeURIComponent(candidateId)}`;

export const tvArtworkCandidates = (
  showId: number,
  scope: "show" | "season",
  seasonNumber?: number,
) =>
  request<{ items: TVArtworkCandidate[] }>(
    `/api/v1/tv/shows/${showId}/artwork-candidates?${tvArtworkQuery(scope, seasonNumber)}`,
  );

export const scrapeTVArtworkCandidates = (
  csrf: string,
  showId: number,
  scope: "show" | "season",
  seasonNumber?: number,
) =>
  request<{ items: TVArtworkCandidate[] }>(
    `/api/v1/tv/shows/${showId}/artwork-candidates?${tvArtworkQuery(scope, seasonNumber)}`,
    {
      method: "POST",
      headers: { "X-CSRF-Token": csrf },
    },
  );

export const tvArtworkPreviewUrl = (showId: number, candidateId: string) =>
  `/api/v1/tv/shows/${showId}/artwork-preview/${encodeURIComponent(candidateId)}`;

export const previewTVArtworkSelection = (
  csrf: string,
  showId: number,
  scope: "show" | "season",
  seasonNumber: number | undefined,
  selections: { kind: TVArtworkCandidate["kind"]; candidateId: string }[],
) =>
  request<TVArtworkPlan>(`/api/v1/tv/shows/${showId}/artwork-plans`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
    body: JSON.stringify({ scope, seasonNumber, selections }),
  });

export const applyTVArtwork = (csrf: string, planId: string) =>
  request<TVArtworkPlan>(`/api/v1/tv/artwork-plans/${planId}/apply`, {
    method: "POST",
    headers: { "X-CSRF-Token": csrf },
  });

function tvArtworkQuery(scope: "show" | "season", seasonNumber?: number) {
  const query = new URLSearchParams({ scope });
  if (scope === "season" && seasonNumber !== undefined)
    query.set("season", String(seasonNumber));
  return query.toString();
}

export const tvNfoRaw = (showId: number, selection: TVSelection) => {
  const params = new URLSearchParams({ kind: selection.kind });
  if (selection.kind === "season")
    params.set("season", String(selection.seasonNumber));
  if (selection.kind === "episode")
    params.set("episode", String(selection.episodeId));
  return request<{ exists: boolean; targetPath: string; content: string }>(
    `/api/v1/tv/shows/${showId}/nfo?${params}`,
  );
};

// Jobs
export const jobs = (signal?: AbortSignal) =>
  request<{ items: Job[] }>("/api/v1/jobs", { signal });
export const activeScans = (after = 0, signal?: AbortSignal) =>
  request<{ items: Job[]; nextCursor: number }>(
    `/api/v1/jobs?activeScans=true&after=${after}`,
    { signal },
  );
export const job = (id: number, signal?: AbortSignal) =>
  request<Job>(`/api/v1/jobs/${id}`, { signal });

export const cancelJob = (csrf: string, id: number) =>
  request<Job>(`/api/v1/jobs/${id}/cancel`, {
    method: "POST",
    headers: { "X-CSRF-Token": csrf },
  });

export const retryJob = (csrf: string, id: number) =>
  request<Job>(`/api/v1/jobs/${id}/retry`, {
    method: "POST",
    headers: { "X-CSRF-Token": csrf },
  });

export const auditEntries = () =>
  request<{ items: AuditEntry[] }>("/api/v1/audit-entries");

export const operationsStatus = () =>
  request<OperationsStatus>("/api/v1/operations/status");

// Settings
export const settings = () => request<Settings>("/api/v1/settings");

export const saveSettings = (csrf: string, update: Partial<SettingsUpdate>) =>
  request<Settings>("/api/v1/settings", {
    method: "PUT",
    headers: {
      "Content-Type": "application/json",
      "X-CSRF-Token": csrf,
    },
    body: JSON.stringify(update),
  });

export const testConnection = (
  csrf: string,
  target: ConnectionTest["target"],
) =>
  request<ConnectionTest>("/api/v1/settings/connection-tests", {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
    body: JSON.stringify({ target }),
  });

export const previewMediaRename = (csrf: string, id: number, pattern: string) =>
  request<RenamePlan>(`/api/v1/media/${id}/rename-plans`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
    body: JSON.stringify({ pattern }),
  });

export const previewTVRename = (
  csrf: string,
  showId: number,
  pattern: string,
  scope?: { seasonNumber?: number; episodeId?: number },
) =>
  request<RenamePlan>(`/api/v1/tv/shows/${showId}/rename-plans`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
    body: JSON.stringify({
      pattern,
      seasonNumber: scope?.seasonNumber,
      episodeId: scope?.episodeId,
    }),
  });

export const getRenamePlan = (id: string) =>
  request<RenamePlan>(`/api/v1/rename-plans/${id}`);

export const applyRenamePlan = (csrf: string, id: string) =>
  request<any>(`/api/v1/rename-plans/${id}/apply`, {
    method: "POST",
    headers: { "X-CSRF-Token": csrf },
  });
