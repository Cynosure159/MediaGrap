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
  let disposed = false;
  let tvGeneration = 0;
  let tvQuery = "";
  const scanRevision = shallowRef(0);
  const trackedScans = new Map<number, api.Job>();
  // Last bounded history snapshot, not an all-time job cache or ID watermark.
  // Bootstrap establishes a baseline without replaying historical completions.
  let recentScanStates:
    | Map<number, Pick<api.Job, "state" | "retryCount">>
    | undefined;
  const scanning = computed(() =>
    jobItems.value.some((item) => item.kind === "scan" && isActive(item)),
  );
  let jobRequest: Promise<void> | undefined;
  let jobController: AbortController | undefined;
  let jobTimer: ReturnType<typeof setTimeout> | undefined;
  let activeCursor = 0;
  let exactCursor = 0;
  let failures = 0;
  let jobError: string | null = null;
  const isActive = (job: api.Job) =>
    job.state === "queued" || job.state === "running";
  let mediaGeneration = 0;
  let mediaPage = 0;
  let mediaQuery = "";
  let mediaOptions: api.CatalogOptions = {};
  let mediaController: AbortController | undefined;
  if (getCurrentScope())
    onScopeDispose(() => {
      disposed = true;
      mediaGeneration++;
      tvGeneration++;
      mediaController?.abort();
      jobController?.abort();
      clearTimeout(jobTimer);
    });
  const tvShowItems = shallowRef<api.TVShow[]>([]);
  const jobItems = shallowRef<api.Job[]>([]);
  const error = shallowRef<string | null>(null);
  const isLoading = shallowRef(false);

  const hasSources = computed(() => sourceItems.value.length > 0);

  async function refreshSources() {
    try {
      const res = await api.sources();
      if (!disposed) sourceItems.value = res.items;
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
    if (disposed) return;
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
    if (disposed) return;
    tvQuery = query;
    const generation = ++tvGeneration;
    try {
      const res = await api.tvShows(query);
      if (!disposed && generation === tvGeneration)
        tvShowItems.value = res.items;
    } catch (caught) {
      error.value =
        caught instanceof Error ? caught.message : "Unable to load TV shows";
    }
  }

  // One serialized scheduler per mounted library, not one timer per scan.
  // Recent history stays bounded; active pages discover old scans after reload,
  // and exact lookups retain their terminal states after they leave history.
  function refreshJobs(): Promise<void> {
    if (disposed) return Promise.resolve();
    if (jobRequest) return jobRequest;
    clearTimeout(jobTimer);
    const controller = new AbortController();
    jobController = controller;
    const timeout = setTimeout(() => controller.abort(), 15000);
    jobRequest = (async () => {
      try {
        const recent = await api.jobs(controller.signal);
        if (disposed) return;
        const active = await api.activeScans(activeCursor, controller.signal);
        if (disposed) return;
        activeCursor = active.nextCursor;
        const snapshot = new Map(
          [...recent.items, ...active.items].map((item) => [item.id, item]),
        );
        // At most eight exact lookups per cycle, rotated to avoid starvation.
        const absent = [...trackedScans.keys()]
          .filter((id) => !snapshot.has(id))
          .sort((a, b) => a - b);
        const ordered = [
          ...absent.filter((id) => id > exactCursor),
          ...absent.filter((id) => id <= exactCursor),
        ].slice(0, 8);
        for (const id of ordered) {
          snapshot.set(id, await api.job(id, controller.signal));
          if (disposed) return;
          exactCursor = id;
        }
        const currentRecentScans = new Map(
          recent.items
            .slice(0, 100)
            .filter((item) => item.kind === "scan")
            .map((item) => [
              item.id,
              { state: item.state, retryCount: item.retryCount },
            ]),
        );
        let completed = false;
        for (const item of snapshot.values()) {
          if (item.kind !== "scan") continue;
          if (isActive(item)) trackedScans.set(item.id, item);
          else if (
            ["succeeded", "failed", "cancelled", "interrupted"].includes(
              item.state,
            )
          ) {
            const wasTracked = trackedScans.delete(item.id);
            const newlyObservedTerminal =
              recentScanStates !== undefined &&
              currentRecentScans.has(item.id) &&
              (recentScanStates.get(item.id)?.state !== item.state ||
                recentScanStates.get(item.id)?.retryCount !== item.retryCount);
            if (!wasTracked && !newlyObservedTerminal) continue;
            completed = true;
            if (item.state !== "succeeded")
              error.value =
                item.errorMessage || item.message || `Scan ${item.state}`;
          }
        }
        recentScanStates = currentRecentScans;
        jobItems.value = [
          ...new Map(
            [...recent.items, ...trackedScans.values()].map((item) => [
              item.id,
              item,
            ]),
          ).values(),
        ];
        if (jobError !== null && error.value === jobError) error.value = null;
        jobError = null;
        failures = 0;
        if (completed) {
          await Promise.all([
            refreshMedia(mediaQuery),
            refreshTVShows(tvQuery),
            refreshSources(),
          ]);
          if (!disposed) scanRevision.value++;
        }
      } catch (caught) {
        if (!disposed) {
          failures++;
          jobError =
            caught instanceof Error ? caught.message : "Unable to load jobs";
          error.value = jobError;
        }
      } finally {
        clearTimeout(timeout);
        jobRequest = undefined;
        if (!disposed) {
          const delay = failures
            ? Math.min(30000, 2000 * 2 ** Math.min(failures - 1, 4))
            : trackedScans.size || activeCursor
              ? 2000
              : 10000;
          jobTimer = setTimeout(() => {
            void refreshJobs();
          }, delay);
        }
      }
    })();
    return jobRequest;
  }

  async function refresh(query = "") {
    isLoading.value = true;
    error.value = null;
    const results = await Promise.allSettled([
      api.sources(),
      refreshMedia(query),
      refreshTVShows(query),
      refreshJobs(),
    ]);

    if (disposed) return;
    const [sourceResult] = results;
    if (sourceResult.status === "fulfilled")
      sourceItems.value = sourceResult.value.items;
    if (mediaError.value) error.value = mediaError.value;

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

  async function scan(id: number, _query = "") {
    if (disposed) return;
    const job = await api.scanSource(csrfToken(), id);
    if (disposed) return;
    trackedScans.set(job.id, job);
    jobItems.value = [
      ...jobItems.value.filter((item) => item.id !== job.id),
      job,
    ];
    await refreshJobs();
  }

  return {
    scanning,
    scanRevision,
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
