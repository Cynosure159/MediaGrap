import { computed, getCurrentScope, onScopeDispose, shallowRef } from "vue";
import * as api from "@/api/library";

export function useLibrary(csrfToken: () => string) {
  const sourceItems = shallowRef<api.Source[]>([]);
  const mediaItems = shallowRef<api.MediaItem[]>([]);
  const mediaTotal = shallowRef<number | null>(null);
  const mediaLoading = shallowRef(false);
  const mediaError = shallowRef<string | null>(null);
  const mediaHasMore = computed(
    () =>
      mediaTotal.value !== null && mediaItems.value.length < mediaTotal.value,
  );
  let mediaGeneration = 0;
  let mediaPage = 0;
  let mediaQuery = "";
  let mediaOptions: api.CatalogOptions = {};
  let mediaController: AbortController | undefined;
  if (getCurrentScope())
    onScopeDispose(() => {
      mediaGeneration++;
      mediaController?.abort();
    });
  const tvShowItems = shallowRef<api.TVShow[]>([]);
  const jobItems = shallowRef<api.Job[]>([]);
  const error = shallowRef<string | null>(null);
  const isLoading = shallowRef(false);

  const hasSources = computed(() => sourceItems.value.length > 0);

  async function refreshSources() {
    try {
      const res = await api.sources();
      sourceItems.value = res.items;
    } catch (caught) {
      error.value =
        caught instanceof Error ? caught.message : "Unable to load sources";
    }
  }

  async function fetchMediaBatch(generation: number, page: number) {
    mediaLoading.value = true;
    mediaError.value = null;
    const controller = new AbortController();
    mediaController = controller;
    try {
      const res = await api.media(
        mediaQuery,
        page,
        mediaOptions,
        controller.signal,
      );
      if (generation !== mediaGeneration) return;
      const seen = new Set(mediaItems.value.map((item) => item.id));
      const incoming = res.items.filter((item) => {
        if (seen.has(item.id)) return false;
        seen.add(item.id);
        return true;
      });
      if (
        page > 1 &&
        incoming.length === 0 &&
        mediaItems.value.length < res.total
      )
        throw new Error("Catalog changed; refresh the list");
      mediaItems.value = [...mediaItems.value, ...incoming];
      mediaTotal.value = res.total;
      mediaPage = page;
    } catch (caught) {
      if (generation !== mediaGeneration || controller.signal.aborted) return;
      mediaError.value =
        caught instanceof Error ? caught.message : "Unable to load media items";
    } finally {
      if (generation === mediaGeneration) mediaLoading.value = false;
    }
  }

  async function refreshMedia(query = "", options = mediaOptions) {
    mediaController?.abort();
    mediaGeneration++;
    mediaPage = 0;
    mediaQuery = query;
    mediaOptions = { ...options };
    mediaItems.value = [];
    mediaTotal.value = null;
    await fetchMediaBatch(mediaGeneration, 1);
  }

  async function loadMoreMedia() {
    if (mediaLoading.value || (mediaPage > 0 && !mediaHasMore.value)) return;
    await fetchMediaBatch(mediaGeneration, mediaPage + 1);
  }

  async function refreshTVShows(query = "") {
    try {
      const res = await api.tvShows(query);
      tvShowItems.value = res.items;
    } catch (caught) {
      error.value =
        caught instanceof Error ? caught.message : "Unable to load TV shows";
    }
  }

  async function refreshJobs() {
    try {
      const res = await api.jobs();
      jobItems.value = res.items;
    } catch (caught) {
      error.value =
        caught instanceof Error ? caught.message : "Unable to load jobs";
    }
  }

  async function refresh(query = "") {
    isLoading.value = true;
    error.value = null;
    const results = await Promise.allSettled([
      api.sources(),
      refreshMedia(query),
      api.tvShows(query),
      api.jobs(),
    ]);

    const [sourceResult, , tvResult, jobResult] = results;
    if (sourceResult.status === "fulfilled")
      sourceItems.value = sourceResult.value.items;
    if (mediaError.value) error.value = mediaError.value;
    if (tvResult.status === "fulfilled")
      tvShowItems.value = tvResult.value.items;
    if (jobResult.status === "fulfilled")
      jobItems.value = jobResult.value.items;

    const failed = results.find((result) => result.status === "rejected");
    if (failed?.status === "rejected") {
      error.value =
        failed.reason instanceof Error
          ? failed.reason.message
          : "Unable to load the library";
    }
    isLoading.value = false;
  }

  async function createSource(name: string, rootPath: string) {
    await api.addSource(csrfToken(), name, rootPath);
    await refreshSources();
  }

  async function removeSource(id: number) {
    await api.deleteSource(csrfToken(), id);
    await refreshSources();
  }

  async function saveSourcePolicy(
    id: number,
    policy: Pick<
      api.Source,
      "scanMode" | "scheduleEnabled" | "scheduleIntervalMinutes"
    >,
  ) {
    await api.updateSourcePolicy(csrfToken(), id, policy);
    await refreshSources();
  }

  async function scan(id: number, query = "") {
    const job = await api.scanSource(csrfToken(), id);
    await refreshJobs();

    // Scans run in the server worker. Poll job status efficiently.
    for (let attempt = 0; attempt < 120; attempt += 1) {
      await new Promise((resolve) => window.setTimeout(resolve, 500));
      await refreshJobs();
      const currentJob = jobItems.value.find((item) => item.id === job.id);
      if (
        currentJob &&
        ["succeeded", "failed", "cancelled"].includes(currentJob.state)
      ) {
        await Promise.all([
          refreshMedia(query),
          refreshTVShows(query),
          refreshSources(),
        ]);
        return;
      }
    }
  }

  return {
    sourceItems,
    mediaItems,
    mediaTotal,
    mediaLoading,
    mediaError,
    mediaHasMore,
    loadMoreMedia,
    tvShowItems,
    jobItems,
    error,
    isLoading,
    hasSources,
    refresh,
    refreshSources,
    refreshMedia,
    refreshTVShows,
    refreshJobs,
    createSource,
    removeSource,
    saveSourcePolicy,
    scan,
  };
}
