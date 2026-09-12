import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import * as api from "@/api/library";
import { useLibrary } from "./useLibrary";

vi.mock("@/api/library", () => ({
  sources: vi.fn(),
  media: vi.fn(),
  tvShows: vi.fn(),
  jobs: vi.fn(),
  activeScans: vi.fn(),
  job: vi.fn(),
  addSource: vi.fn(),
  deleteSource: vi.fn(),
  scanSource: vi.fn(),
}));

const movie = (id: number) => ({
  id,
  sourceId: 1,
  relativePath: `Movie${id}.mkv`,
  titleHint: `Movie ${id}`,
  yearHint: 2024,
  fileSize: 1,
  modifiedAt: "",
  sidecars: [],
});

describe("useLibrary", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.mocked(api.activeScans).mockResolvedValue({ items: [], nextCursor: 0 });
  });
  it("loads beyond 50 only on demand and stops at the true total", async () => {
    vi.mocked(api.media)
      .mockResolvedValueOnce({
        items: Array.from({ length: 50 }, (_, i) => movie(i + 1)),
        total: 73,
      })
      .mockResolvedValueOnce({
        items: Array.from({ length: 23 }, (_, i) => movie(i + 51)),
        total: 73,
      });
    const library = useLibrary(() => "");
    await library.refreshMedia("query");
    expect(library.mediaTotal.value).toBe(73);
    expect(library.mediaItems.value).toHaveLength(50);
    expect(api.media).toHaveBeenCalledTimes(1);
    await Promise.all([library.loadMoreMedia(), library.loadMoreMedia()]);
    expect(api.media).toHaveBeenCalledTimes(2);
    expect(library.mediaItems.value.at(-1)?.id).toBe(73);
    expect(library.mediaHasMore.value).toBe(false);
    await library.loadMoreMedia();
    expect(api.media).toHaveBeenCalledTimes(2);
  });

  it("ignores a stale search even if its transport ignores abort", async () => {
    let resolveOld!: (value: {
      items: ReturnType<typeof movie>[];
      total: number;
    }) => void;
    vi.mocked(api.media)
      .mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            resolveOld = resolve;
          }),
      )
      .mockResolvedValueOnce({ items: [movie(2)], total: 1 });
    const library = useLibrary(() => "");
    const old = library.refreshMedia("old");
    await library.refreshMedia("new", { sort: "year", filter: "4k" });
    resolveOld({ items: [movie(1)], total: 99 });
    await old;
    expect(library.mediaItems.value.map((m) => m.id)).toEqual([2]);
    expect(library.mediaTotal.value).toBe(1);
  });

  it("retains loaded rows on failure and retries the same batch", async () => {
    vi.mocked(api.media)
      .mockResolvedValueOnce({ items: [movie(1)], total: 2 })
      .mockRejectedValueOnce(new Error("offline"))
      .mockResolvedValueOnce({ items: [movie(1), movie(2)], total: 2 });
    const library = useLibrary(() => "");
    await library.refreshMedia();
    await library.loadMoreMedia();
    expect(library.mediaError.value).toBe("offline");
    expect(library.mediaItems.value).toHaveLength(1);
    await library.loadMoreMedia();
    expect(library.mediaItems.value.map((m) => m.id)).toEqual([1, 2]);
    expect(vi.mocked(api.media).mock.calls[1]?.[1]).toBe(2);
    expect(vi.mocked(api.media).mock.calls[2]?.[1]).toBe(2);
  });
  afterEach(() => {
    vi.clearAllMocks();
    vi.clearAllTimers();
    vi.useRealTimers();
  });

  it("loads sources, media, tvShows, and jobs on refresh", async () => {
    vi.mocked(api.sources).mockResolvedValueOnce({
      items: [
        {
          id: 1,
          name: "Movies",
          rootPath: "/media",
          enabled: true,
          itemCount: 10,
          writable: true,
          scanMode: "incremental",
          scheduleEnabled: false,
          scheduleIntervalMinutes: 1440,
        },
      ],
    });
    vi.mocked(api.media).mockResolvedValueOnce({
      items: [
        {
          id: 101,
          sourceId: 1,
          relativePath: "Movie.mkv",
          titleHint: "Movie",
          yearHint: 2024,
          fileSize: 1000,
          modifiedAt: "now",
          sidecars: [],
        },
      ],
      total: 1,
    });
    vi.mocked(api.tvShows).mockResolvedValueOnce({
      items: [
        {
          id: 201,
          sourceId: 1,
          relativePath: "Show",
          titleHint: "Show",
          yearHint: 2024,
          episodeCount: 1,
          seasonCount: 1,
        },
      ],
    });
    vi.mocked(api.jobs).mockResolvedValueOnce({ items: [] });

    const library = useLibrary(() => "test-csrf");
    expect(library.hasSources.value).toBe(false);

    await library.refresh();

    expect(library.isLoading.value).toBe(false);
    expect(library.error.value).toBeNull();
    expect(library.sourceItems.value).toHaveLength(1);
    expect(library.mediaItems.value).toHaveLength(1);
    expect(library.tvShowItems.value).toHaveLength(1);
    expect(library.hasSources.value).toBe(true);
  });

  it("handles error when api fails during refresh", async () => {
    vi.mocked(api.sources).mockRejectedValueOnce(new Error("Network error"));
    vi.mocked(api.media).mockResolvedValueOnce({ items: [], total: 0 });
    vi.mocked(api.tvShows).mockResolvedValueOnce({ items: [] });
    vi.mocked(api.jobs).mockResolvedValueOnce({ items: [] });

    const library = useLibrary(() => "test-csrf");
    await library.refresh();

    expect(library.error.value).toBe("Network error");
    expect(library.isLoading.value).toBe(false);
  });

  it("keeps successful library sections available when one request fails", async () => {
    vi.mocked(api.sources).mockResolvedValueOnce({
      items: [
        {
          id: 1,
          name: "TV",
          rootPath: "/tv",
          enabled: true,
          itemCount: 1,
          writable: true,
          scanMode: "incremental",
          scheduleEnabled: false,
          scheduleIntervalMinutes: 1440,
        },
      ],
    });
    vi.mocked(api.media).mockRejectedValueOnce(new Error("Movies unavailable"));
    vi.mocked(api.tvShows).mockResolvedValueOnce({
      items: [
        {
          id: 201,
          sourceId: 1,
          relativePath: "Show",
          titleHint: "Show",
          yearHint: 2024,
          episodeCount: 1,
          seasonCount: 1,
        },
      ],
    });
    vi.mocked(api.jobs).mockResolvedValueOnce({ items: [] });

    const library = useLibrary(() => "test-csrf");
    await library.refresh();

    expect(library.sourceItems.value).toHaveLength(1);
    expect(library.tvShowItems.value).toHaveLength(1);
    expect(library.error.value).toBe("Movies unavailable");
  });

  it("calls createSource and triggers refresh", async () => {
    vi.mocked(api.addSource).mockResolvedValueOnce({
      id: 2,
      name: "TV",
      rootPath: "/tv",
      enabled: true,
      itemCount: 0,
      writable: true,
      scanMode: "incremental",
      scheduleEnabled: false,
      scheduleIntervalMinutes: 1440,
    });
    vi.mocked(api.sources).mockResolvedValueOnce({ items: [] });
    vi.mocked(api.media).mockResolvedValueOnce({ items: [], total: 0 });
    vi.mocked(api.tvShows).mockResolvedValueOnce({ items: [] });
    vi.mocked(api.jobs).mockResolvedValueOnce({ items: [] });

    const library = useLibrary(() => "test-csrf");
    await library.createSource("TV", "/tv");

    expect(api.addSource).toHaveBeenCalledWith("test-csrf", "TV", "/tv");
    expect(api.sources).toHaveBeenCalled();
  });

  it("calls removeSource and triggers refresh", async () => {
    vi.mocked(api.deleteSource).mockResolvedValueOnce(undefined);
    vi.mocked(api.sources).mockResolvedValueOnce({ items: [] });
    vi.mocked(api.media).mockResolvedValueOnce({ items: [], total: 0 });
    vi.mocked(api.tvShows).mockResolvedValueOnce({ items: [] });
    vi.mocked(api.jobs).mockResolvedValueOnce({ items: [] });

    const library = useLibrary(() => "test-csrf");
    await library.removeSource(5);

    expect(api.deleteSource).toHaveBeenCalledWith("test-csrf", 5);
    expect(api.sources).toHaveBeenCalled();
  });
});
