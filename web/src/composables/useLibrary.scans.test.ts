import { effectScope } from "vue";
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
  scanSource: vi.fn(),
}));
const scanJob = (state = "running", id = 1): api.Job => ({
  id,
  state,
  kind: "scan",
  sourceId: 1,
  progressCurrent: 25,
  progressTotal: 0,
  message: state,
  errorMessage: "",
  retryCount: 0,
  maxRetries: 2,
  createdAt: "",
  updatedAt: "",
});
const scopes: ReturnType<typeof effectScope>[] = [];
function mounted() {
  const scope = effectScope();
  scopes.push(scope);
  return { scope, library: scope.run(() => useLibrary(() => "csrf"))! };
}

describe("useLibrary persisted scans", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.mocked(api.sources).mockResolvedValue({ items: [] });
    vi.mocked(api.media).mockResolvedValue({ items: [], total: 0 });
    vi.mocked(api.tvShows).mockResolvedValue({ items: [] });
    vi.mocked(api.jobs).mockResolvedValue({ items: [scanJob()] });
    vi.mocked(api.activeScans).mockResolvedValue({
      items: [scanJob()],
      nextCursor: 0,
    });
    vi.mocked(api.scanSource).mockResolvedValue(scanJob("queued"));
  });
  afterEach(() => {
    scopes.splice(0).forEach((scope) => scope.stop());
    vi.clearAllTimers();
    vi.useRealTimers();
    vi.resetAllMocks();
  });

  it.each(["succeeded", "failed", "cancelled", "interrupted"])(
    "tracks beyond 60s and refreshes current query/details once on %s",
    async (state) => {
      const { library } = mounted();
      await library.refresh();
      await vi.advanceTimersByTimeAsync(70000);
      expect(library.scanning.value).toBe(true);
      expect(api.jobs).toHaveBeenCalledTimes(36);
      expect(library.scanRevision.value).toBe(0);
      await library.refreshMedia("new query", { sort: "year" });
      vi.mocked(api.jobs).mockResolvedValue({ items: [scanJob(state)] });
      vi.mocked(api.activeScans).mockResolvedValue({
        items: [],
        nextCursor: 0,
      });
      await vi.advanceTimersByTimeAsync(2000);
      expect(library.scanning.value).toBe(false);
      expect(library.scanRevision.value).toBe(1);
      expect(vi.mocked(api.media).mock.lastCall?.slice(0, 3)).toEqual([
        "new query",
        1,
        { sort: "year" },
      ]);
      expect(library.error.value).toBe(state === "succeeded" ? null : state);
      await vi.advanceTimersByTimeAsync(20000);
      expect(library.scanRevision.value).toBe(1);
    },
  );

  it.each(["succeeded", "failed", "cancelled", "interrupted"])(
    "detects external %s entirely between polls without replaying history",
    async (state) => {
      vi.mocked(api.activeScans).mockResolvedValue({
        items: [],
        nextCursor: 0,
      });
      vi.mocked(api.jobs).mockResolvedValue({
        items: [scanJob("succeeded", 20)],
      });
      const first = mounted();
      await first.library.refresh();
      expect(first.library.scanRevision.value).toBe(0);
      vi.mocked(api.jobs).mockResolvedValue({
        items: [scanJob(state, 21), scanJob("succeeded", 20)],
      });
      await vi.advanceTimersByTimeAsync(10000);
      expect(first.library.scanRevision.value).toBe(1);
      expect(first.library.error.value).toBe(
        state === "succeeded" ? null : state,
      );
      expect(api.media).toHaveBeenCalledTimes(2);
      await vi.advanceTimersByTimeAsync(20000);
      expect(first.library.scanRevision.value).toBe(1);
      expect(api.media).toHaveBeenCalledTimes(2);
      first.scope.stop();
      const second = mounted();
      await second.library.refresh();
      expect(second.library.scanRevision.value).toBe(0);
      await vi.advanceTimersByTimeAsync(10000);
      expect(second.library.scanRevision.value).toBe(0);
      expect(first.library.scanRevision.value).toBe(1);
    },
  );

  it.each(["failed", "cancelled", "interrupted"])(
    "detects same-ID %s retries entirely between polls exactly once per attempt",
    async (state) => {
      vi.mocked(api.activeScans).mockResolvedValue({
        items: [],
        nextCursor: 0,
      });
      vi.mocked(api.jobs).mockResolvedValue({ items: [scanJob(state, 3)] });
      const { library, scope } = mounted();
      await library.refresh();
      await vi.advanceTimersByTimeAsync(10000);
      expect(library.scanRevision.value).toBe(0);
      for (const retryCount of [1, 2]) {
        vi.mocked(api.jobs).mockResolvedValue({
          items: [{ ...scanJob(state, 3), retryCount }],
        });
        await vi.advanceTimersByTimeAsync(10000);
        expect(library.scanRevision.value).toBe(retryCount);
        expect(api.media).toHaveBeenCalledTimes(retryCount + 1);
        expect(api.tvShows).toHaveBeenCalledTimes(retryCount + 1);
        expect(api.sources).toHaveBeenCalledTimes(retryCount + 1);
        expect(library.error.value).toBe(state);
        // Unrelated fields and duplicate object instances are not new attempts.
        vi.mocked(api.jobs).mockResolvedValue({
          items: [
            {
              ...scanJob(state, 3),
              retryCount,
              updatedAt: "later",
              message: "duplicate",
            },
          ],
        });
        await vi.advanceTimersByTimeAsync(20000);
        expect(library.scanRevision.value).toBe(retryCount);
        expect(api.media).toHaveBeenCalledTimes(retryCount + 1);
      }
      scope.stop();
      const remounted = mounted();
      await remounted.library.refresh();
      await vi.advanceTimersByTimeAsync(10000);
      expect(remounted.library.scanRevision.value).toBe(0);
    },
  );

  it.each(["failed", "cancelled", "interrupted"])(
    "coalesces tracked and recent %s retry completion without duplicates",
    async (state) => {
      vi.mocked(api.activeScans).mockResolvedValue({
        items: [],
        nextCursor: 0,
      });
      vi.mocked(api.jobs).mockResolvedValue({ items: [scanJob(state)] });
      const { library } = mounted();
      await library.refreshJobs();
      vi.mocked(api.jobs).mockResolvedValue({
        items: [{ ...scanJob("running"), retryCount: 1 }],
      });
      await library.refreshJobs();
      vi.mocked(api.jobs).mockResolvedValue({
        items: [{ ...scanJob(state), retryCount: 1 }],
      });
      await library.refreshJobs();
      await library.refreshJobs();
      expect(library.scanRevision.value).toBe(1);
      expect(api.media).toHaveBeenCalledTimes(1);
    },
  );

  it("coalesces new terminal scans and handles older out-of-order completion", async () => {
    vi.mocked(api.activeScans).mockResolvedValue({ items: [], nextCursor: 0 });
    vi.mocked(api.jobs).mockResolvedValue({
      items: [scanJob("running", 1), scanJob("succeeded", 10)],
    });
    const { library } = mounted();
    await library.refreshJobs();
    vi.mocked(api.jobs).mockResolvedValue({
      items: [
        scanJob("running", 1),
        scanJob("succeeded", 10),
        scanJob("succeeded", 11),
        scanJob("succeeded", 12),
      ],
    });
    await library.refreshJobs();
    expect(library.scanRevision.value).toBe(1);
    expect(api.media).toHaveBeenCalledTimes(1);
    vi.mocked(api.jobs).mockResolvedValue({
      items: [
        scanJob("succeeded", 1),
        scanJob("succeeded", 10),
        scanJob("succeeded", 11),
        scanJob("succeeded", 12),
      ],
    });
    await library.refreshJobs();
    expect(library.scanRevision.value).toBe(2);
    await library.refreshJobs();
    expect(library.scanRevision.value).toBe(2);
    expect(api.media).toHaveBeenCalledTimes(2);
  });

  it("does not observe a new external terminal response after disposal", async () => {
    vi.mocked(api.activeScans).mockResolvedValue({ items: [], nextCursor: 0 });
    vi.mocked(api.jobs).mockResolvedValue({ items: [] });
    const { library, scope } = mounted();
    await library.refreshJobs();
    let resolve!: (value: { items: api.Job[] }) => void;
    vi.mocked(api.jobs).mockImplementationOnce(
      () =>
        new Promise((done) => {
          resolve = done;
        }),
    );
    const pending = library.refreshJobs();
    scope.stop();
    resolve({ items: [scanJob("succeeded", 2)] });
    await pending;
    expect(library.scanRevision.value).toBe(0);
    expect(api.media).not.toHaveBeenCalled();
    await vi.advanceTimersByTimeAsync(30000);
    expect(api.jobs).toHaveBeenCalledTimes(2);
  });

  it("rediscovers on navigation/reload and follows an old terminal job beyond 100 newer jobs", async () => {
    const recent = Array.from({ length: 100 }, (_, i) => ({
      ...scanJob("succeeded", i + 100),
      kind: "other",
    }));
    vi.mocked(api.jobs).mockResolvedValue({ items: recent });
    const first = mounted();
    await first.library.refresh();
    expect(first.library.scanning.value).toBe(true);
    first.scope.stop();
    const second = mounted();
    await second.library.refresh();
    expect(second.library.scanning.value).toBe(true);
    vi.mocked(api.activeScans).mockResolvedValue({ items: [], nextCursor: 0 });
    vi.mocked(api.job).mockResolvedValue(scanJob("succeeded"));
    await vi.advanceTimersByTimeAsync(2000);
    expect(api.job).toHaveBeenCalledWith(1, expect.any(AbortSignal));
    expect(second.library.scanning.value).toBe(false);
    expect(second.library.scanRevision.value).toBe(1);
    expect(first.library.scanRevision.value).toBe(0);
  });

  it("serializes overlapping refreshes and aborts/ignores a late response on unmount", async () => {
    let resolve!: (value: { items: api.Job[] }) => void;
    vi.mocked(api.jobs).mockImplementation(
      () =>
        new Promise((done) => {
          resolve = done;
        }),
    );
    const { scope, library } = mounted();
    const first = library.refreshJobs(),
      second = library.refreshJobs();
    await vi.advanceTimersByTimeAsync(90000);
    expect(api.jobs).toHaveBeenCalledTimes(1);
    expect(first).toBe(second);
    scope.stop();
    expect(vi.mocked(api.jobs).mock.lastCall?.[0]?.aborted).toBe(true);
    resolve({ items: [scanJob()] });
    await first;
    expect(library.jobItems.value).toEqual([]);
    await vi.advanceTimersByTimeAsync(90000);
    expect(api.jobs).toHaveBeenCalledTimes(1);
  });

  it("backs off errors, never treats absent/failed exact lookup as success, then recovers", async () => {
    const { library } = mounted();
    await library.refreshJobs();
    vi.mocked(api.jobs).mockResolvedValue({ items: [] });
    vi.mocked(api.activeScans).mockResolvedValue({ items: [], nextCursor: 0 });
    vi.mocked(api.job).mockRejectedValue(new Error("offline"));
    await vi.advanceTimersByTimeAsync(8000);
    expect(api.job).toHaveBeenCalledTimes(3);
    expect(library.scanning.value).toBe(true);
    expect(library.scanRevision.value).toBe(0);
    expect(library.error.value).toBe("offline");
    vi.mocked(api.job).mockResolvedValue(scanJob("cancelled"));
    await vi.advanceTimersByTimeAsync(8000);
    expect(library.scanning.value).toBe(false);
    expect(library.scanRevision.value).toBe(1);
  });

  it("clears a recovered polling error without expiring the scan", async () => {
    vi.mocked(api.jobs).mockRejectedValueOnce(new Error("offline"));
    const { library } = mounted();
    await library.refreshJobs();
    expect(library.error.value).toBe("offline");
    await vi.advanceTimersByTimeAsync(2000);
    expect(library.error.value).toBeNull();
    expect(library.scanning.value).toBe(true);
  });

  it("bounds active discovery to one page and exact lookups to eight per cycle", async () => {
    vi.mocked(api.jobs).mockResolvedValue({ items: [] });
    vi.mocked(api.activeScans)
      .mockResolvedValueOnce({
        items: Array.from({ length: 100 }, (_, i) => scanJob("running", i + 1)),
        nextCursor: 100,
      })
      .mockResolvedValue({ items: [], nextCursor: 0 });
    vi.mocked(api.job).mockImplementation(async (id) => scanJob("running", id));
    const { library } = mounted();
    await library.refreshJobs();
    expect(api.activeScans).toHaveBeenCalledTimes(1);
    await vi.advanceTimersByTimeAsync(2000);
    expect(api.activeScans).toHaveBeenLastCalledWith(
      100,
      expect.any(AbortSignal),
    );
    expect(api.job).toHaveBeenCalledTimes(8);
    await vi.advanceTimersByTimeAsync(2000);
    expect(api.job).toHaveBeenCalledTimes(16);
    expect(vi.mocked(api.job).mock.calls.map((call) => call[0])).toEqual(
      Array.from({ length: 16 }, (_, i) => i + 1),
    );
  });
});
