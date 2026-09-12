import { afterEach, describe, expect, it, vi } from "vitest";
import { flushPromises, mount, shallowMount } from "@vue/test-utils";
import * as api from "@/api/library";
import { RequestError } from "@/api/client";
import MovieInspector from "./MovieInspector.vue";
import TVShowInspector from "./TVShowInspector.vue";

vi.mock("@/api/library", () => ({
  mediaDetail: vi.fn(),
  tvShowDetail: vi.fn(),
  mediaInspection: vi.fn().mockResolvedValue(null),
  tvNfoRaw: vi.fn().mockResolvedValue({ content: "" }),
  saveMetadata: vi.fn(),
  previewNfo: vi.fn().mockResolvedValue({ id: "plan" }),
  applyNfo: vi.fn().mockResolvedValue(undefined),
  previewTVNfoPlans: vi.fn(),
  scrapeTVEpisode: vi.fn(),
  scrapeTVSeason: vi.fn(),
  selectTVShowCandidate: vi.fn(),
}));

function record(title = "Original") {
  return {
    item: { id: 1, titleHint: title, yearHint: 2020, sidecars: [] },
    show: { id: 1, titleHint: title, yearHint: 2020 },
    metadata: { title, episodes: [] },
    episodes: [],
    artwork: [],
    writable: true,
  };
}
function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (reason: Error) => void;
  const promise = new Promise<T>((yes, no) => {
    resolve = yes;
    reject = no;
  });
  return { promise, resolve, reject };
}
afterEach(() => {
  vi.useRealTimers();
  vi.resetAllMocks();
});

describe.each(["movie", "TV"] as const)("%s scan refresh", (kind) => {
  function setup() {
    const fetchDetail = vi.mocked(
      kind === "movie" ? api.mediaDetail : api.tvShowDetail,
    );
    fetchDetail.mockResolvedValue(record() as never);
    vi.mocked(api.tvNfoRaw).mockResolvedValue({
      content: "",
      exists: false,
      targetPath: "",
    });
    const wrapper = shallowMount(
      kind === "movie" ? MovieInspector : TVShowInspector,
      {
        props: {
          itemId: 1,
          selection: { kind: "show", showId: 1 },
          scanRevision: 0,
          csrfToken: "",
          labels: {},
        },
      },
    );
    const toolbar = () => wrapper.findComponent({ name: "InspectorToolbar" });
    const overview = () =>
      wrapper.findComponent({
        name: kind === "movie" ? "MovieOverviewTab" : "TVOverviewTab",
      });
    return { wrapper, fetchDetail, toolbar, overview };
  }

  it("preserves edits across multiple terminal notifications and refreshes once after discard", async () => {
    const { wrapper, fetchDetail, toolbar, overview } = setup();
    await flushPromises();
    toolbar().vm.$emit("toggleEdit");
    await flushPromises();
    overview().props("draft").title = "Unsaved title";
    await wrapper.setProps({ scanRevision: 1 });
    await wrapper.setProps({ scanRevision: 2 });
    expect(fetchDetail).toHaveBeenCalledTimes(3);
    expect(overview().props("draft").title).toBe("Unsaved title");
    // Merely toggling out of edit mode must not discard the dirty draft either.
    toolbar().vm.$emit("toggleEdit");
    await flushPromises();
    expect(fetchDetail).toHaveBeenCalledTimes(3);
    fetchDetail.mockResolvedValue(record("Scanned") as never);
    toolbar().vm.$emit("cancelEdit");
    await flushPromises();
    expect(fetchDetail).toHaveBeenCalledTimes(4);
    expect(overview().props("draft").title).toBe("Scanned");
    wrapper.unmount();
  });

  it("does not overwrite edits started during an in-flight refresh", async () => {
    const { wrapper, fetchDetail, toolbar, overview } = setup();
    await flushPromises();
    const pending = deferred<never>();
    fetchDetail.mockReturnValueOnce(pending.promise);
    await wrapper.setProps({ scanRevision: 1 });
    toolbar().vm.$emit("toggleEdit");
    await flushPromises();
    overview().props("draft").title = "Keep me";
    pending.resolve(record("Stale response") as never);
    await flushPromises();
    expect(overview().props("draft").title).toBe("Keep me");
    toolbar().vm.$emit("cancelEdit");
    await flushPromises();
    expect(fetchDetail).toHaveBeenCalledTimes(3);
    wrapper.unmount();
  });

  it.each([false, true])(
    "keeps pending save alive (failure=%s)",
    async (fails) => {
      const { wrapper, fetchDetail, toolbar, overview } = setup();
      await flushPromises();
      toolbar().vm.$emit("toggleEdit");
      await flushPromises();
      overview().props("draft").title = "Save draft";
      const save = deferred<never>();
      vi.mocked(
        kind === "movie" ? api.saveMetadata : api.previewTVNfoPlans,
      ).mockReturnValueOnce(save.promise);
      toolbar().vm.$emit("saveEdit");
      await flushPromises();
      await wrapper.setProps({ scanRevision: 1 });
      await flushPromises();
      expect(fetchDetail).toHaveBeenCalledTimes(2);
      expect(toolbar().props("isSaving")).toBe(true);
      expect(overview().props("draft").title).toBe("Save draft");
      const retainedDraft = overview().props("draft");
      if (fails) save.reject(new Error("Save failed"));
      else save.resolve({} as never);
      await flushPromises();
      expect(toolbar().props("isSaving")).toBe(false);
      if (fails) {
        expect(fetchDetail).toHaveBeenCalledTimes(2);
        expect(retainedDraft.title).toBe("Save draft");
        expect(toolbar().props("isEditing")).toBe(true);
        expect(wrapper.text()).toContain("Save failed");
      } else {
        expect(wrapper.emitted("metadataSaved")).toHaveLength(1);
        expect(fetchDetail).toHaveBeenCalledTimes(4);
      }
      wrapper.unmount();
    },
  );

  it("refreshes clean details without unmounting workshops and clears deleted items", async () => {
    const { wrapper, fetchDetail, overview, toolbar } = setup();
    await flushPromises();
    const original = overview().vm;
    fetchDetail.mockResolvedValueOnce(record("Updated") as never);
    await wrapper.setProps({ scanRevision: 1 });
    await flushPromises();
    expect(overview().vm).toBe(original);
    expect(overview().props("draft").title).toBe("Updated");
    fetchDetail.mockRejectedValueOnce(new RequestError("not found", 404));
    await wrapper.setProps({ scanRevision: 2 });
    await flushPromises();
    expect(overview().exists()).toBe(false);
    expect(toolbar().props("hasDetail")).toBe(false);
    expect(wrapper.text()).toContain("not found");
    wrapper.unmount();
  });

  it.each(["artwork", "files"] as const)(
    "defers refresh while %s owns previews/mutations",
    async (activeTab) => {
      const { wrapper, fetchDetail } = setup();
      await flushPromises();
      await wrapper.setProps({ activeTab, scanRevision: 1 });
      await flushPromises();
      expect(fetchDetail).toHaveBeenCalledTimes(2);
      await wrapper.setProps({ activeTab: "overview" });
      await flushPromises();
      expect(fetchDetail).toHaveBeenCalledTimes(3);
      wrapper.unmount();
    },
  );
  it.each([404, 500])(
    "retains dirty input and blocks writes on availability status %s, then revalidates",
    async (status) => {
      const { wrapper, fetchDetail, toolbar, overview } = setup();
      await flushPromises();
      toolbar().vm.$emit("toggleEdit");
      await flushPromises();
      overview().props("draft").title = "Recoverable input";
      fetchDetail.mockRejectedValueOnce(
        new RequestError("request failed", status),
      );
      await wrapper.setProps({ scanRevision: 1 });
      await flushPromises();
      expect(overview().props("draft").title).toBe("Recoverable input");
      expect(toolbar().props("isWritable")).toBe(false);
      expect(toolbar().props("mutationsBlocked")).toBe(true);
      expect(wrapper.get("[role=alert]").text()).toContain(
        status === 404 ? "no longer in the library" : "until verified",
      );
      toolbar().vm.$emit("saveEdit");
      toolbar().vm.$emit("scrape");
      await flushPromises();
      expect(api.saveMetadata).not.toHaveBeenCalled();
      expect(api.previewTVNfoPlans).not.toHaveBeenCalled();
      await wrapper.get("[role=alert] button").trigger("click");
      await flushPromises();
      expect(toolbar().props("mutationsBlocked")).toBe(false);
      expect(overview().props("draft").title).toBe("Recoverable input");
      wrapper.unmount();
    },
  );

  it("ignores pending availability responses after unmount", async () => {
    const { wrapper, fetchDetail } = setup();
    await flushPromises();
    const pending = deferred<never>();
    fetchDetail.mockReturnValueOnce(pending.promise);
    await wrapper.setProps({ scanRevision: 1 });
    const signal = fetchDetail.mock.calls.at(-1)?.[1];
    wrapper.unmount();
    expect(signal?.aborted).toBe(true);
    pending.resolve(record("late") as never);
    await flushPromises();
    expect(fetchDetail).toHaveBeenCalledTimes(2);
  });
});

it("retains a dirty TV episode draft but blocks actions when only that episode disappears", async () => {
  const initial = {
    ...record(),
    episodes: [
      {
        id: 7,
        seasonNumber: 1,
        episodeStart: 1,
        fileSize: 1,
        relativePath: "Show.S01E01.mkv",
      },
    ],
  };
  vi.mocked(api.tvShowDetail)
    .mockResolvedValueOnce(initial as never)
    .mockResolvedValue(record() as never);
  vi.mocked(api.tvNfoRaw).mockResolvedValue({
    exists: false,
    content: "",
    targetPath: "",
  });
  const wrapper = shallowMount(TVShowInspector, {
    props: {
      selection: { kind: "episode", showId: 1, episodeId: 7, seasonNumber: 1 },
      scanRevision: 0,
      csrfToken: "",
      labels: {},
    },
  });
  await flushPromises();
  const toolbar = wrapper.findComponent({ name: "InspectorToolbar" });
  toolbar.vm.$emit("toggleEdit");
  await flushPromises();
  const draft = wrapper.findComponent({ name: "TVOverviewTab" }).props("draft");
  draft.title = "Episode draft";
  await wrapper.setProps({ scanRevision: 1 });
  await flushPromises();
  expect(draft.title).toBe("Episode draft");
  expect(toolbar.props("mutationsBlocked")).toBe(true);
  expect(wrapper.get("[role=alert]").text()).toContain(
    "no longer in the library",
  );
  wrapper.unmount();
});

describe("TVShowInspector first-load selection ownership", () => {
  const episode = (episodeId = 7): api.TVSelection => ({
    kind: "episode",
    showId: 1,
    episodeId,
    seasonNumber: 1,
  });
  const showRecord = (title = "Current show") => ({
    ...record(title),
    episodes: [7, 8, 9].map((id) => ({
      id,
      seasonNumber: id === 9 ? 2 : 1,
      episodeStart: id - 6,
      relativePath: `Show.${id}.mkv`,
      fileSize: 1,
      sidecars: [],
    })),
  });
  function setup() {
    vi.mocked(api.tvNfoRaw).mockResolvedValue({
      content: "",
      exists: false,
      targetPath: "",
    });
    vi.mocked(api.mediaInspection).mockResolvedValue(null as never);
    const first = deferred<never>();
    const replacement = deferred<never>();
    vi.mocked(api.tvShowDetail)
      .mockReturnValueOnce(first.promise)
      .mockReturnValueOnce(replacement.promise)
      .mockResolvedValue(showRecord() as never);
    const wrapper = mount(TVShowInspector, {
      props: {
        selection: episode(),
        scanRevision: 0,
        csrfToken: "",
        labels: { season: "Season" },
      },
    });
    const toolbar = () => wrapper.findComponent({ name: "InspectorToolbar" });
    const overview = () => wrapper.findComponent({ name: "TVOverviewTab" });
    return { wrapper, toolbar, overview, first, replacement };
  }

  describe.each(["episode", "season"] as const)(
    "switch to %s before first detail",
    (kind) => {
      it.each(["initial-first", "replacement-first"])(
        "populates pristine detail with %s settlement",
        async (order) => {
          const { wrapper, toolbar, overview, first, replacement } = setup();
          try {
            await wrapper.setProps({
              selection:
                kind === "episode"
                  ? episode(8)
                  : { kind: "season", showId: 1, seasonNumber: 2 },
            });
            expect(api.tvShowDetail).toHaveBeenCalledTimes(2);
            const settleFirst = async () => {
              first.resolve(showRecord("Obsolete first load") as never);
              await flushPromises();
            };
            const settleReplacement = async () => {
              replacement.resolve(showRecord() as never);
              await flushPromises();
            };
            if (order === "initial-first") {
              await settleFirst();
              await settleReplacement();
            } else {
              await settleReplacement();
              await settleFirst();
            }
            expect(overview().exists()).toBe(true);
            expect(overview().props("draft").title).toBe("Current show");
            expect(wrapper.text()).not.toContain("Obsolete first load");
            expect(toolbar().props("hasDetail")).toBe(true);
            expect(toolbar().props("mutationsBlocked")).toBe(false);
            expect(overview().props("selectedUnitPath")).toBe(
              kind === "episode" ? "Show.8.mkv" : "",
            );
            // Subsequent detail changes remain usable, rather than requiring remount/reload.
            vi.mocked(api.tvShowDetail).mockResolvedValue(
              showRecord("Later detail") as never,
            );
            await wrapper.setProps({ scanRevision: 1 });
            await flushPromises();
            expect(overview().props("draft").title).toBe("Later detail");
          } finally {
            wrapper.unmount();
          }
        },
      );
    },
  );

  it("recovers pristine detail after initial failure, failed selection check and Retry", async () => {
    const { wrapper, toolbar, overview, first, replacement } = setup();
    try {
      first.reject(new RequestError("Initial detail failed", 500));
      await flushPromises();
      expect(wrapper.text()).toContain("Initial detail failed");
      await wrapper.setProps({ selection: episode(8) });
      replacement.reject(new RequestError("Retry needed", 500));
      await flushPromises();
      expect(toolbar().props("mutationsBlocked")).toBe(true);
      await wrapper.get("[role=alert] button").trigger("click");
      await flushPromises();
      expect(overview().exists()).toBe(true);
      expect(overview().props("draft").title).toBe("Current show");
      expect(overview().props("selectedUnitPath")).toBe("Show.8.mkv");
      expect(wrapper.text()).not.toContain("Initial detail failed");
      expect(toolbar().props("mutationsBlocked")).toBe(false);
      await wrapper.setProps({ selection: { kind: "show", showId: 1 } });
      await flushPromises();
      toolbar().vm.$emit("toggleEdit");
      await flushPromises();
      overview().props("draft").title = "Real unsaved draft";
      await wrapper.setProps({ selection: episode(7), scanRevision: 1 });
      await flushPromises();
      expect(overview().props("draft").title).toBe("Real unsaved draft");
    } finally {
      wrapper.unmount();
    }
  });
});

describe("same-show selection lifecycle", () => {
  const episode = (episodeId = 7, seasonNumber = 1) => ({
    kind: "episode" as const,
    showId: 1,
    episodeId,
    seasonNumber,
  });
  const showRecord = () => ({
    ...record(),
    episodes: [7, 8, 9].map((id) => ({
      id,
      seasonNumber: id === 9 ? 2 : 1,
      episodeStart: id - 6,
      relativePath: `Show.${id}.mkv`,
      fileSize: 1,
    })),
  });
  async function setup() {
    vi.mocked(api.tvShowDetail).mockResolvedValue(showRecord() as never);
    const wrapper = shallowMount(TVShowInspector, {
      props: {
        selection: episode(),
        scanRevision: 0,
        csrfToken: "",
        labels: {},
      },
    });
    await flushPromises();
    const toolbar = () => wrapper.findComponent({ name: "InspectorToolbar" });
    toolbar().vm.$emit("toggleEdit");
    await flushPromises();
    const draft = wrapper
      .findComponent({ name: "TVOverviewTab" })
      .props("draft");
    draft.title = "Keep same-show draft";
    return { wrapper, toolbar, draft };
  }

  it.each([false, true])(
    "reschedules checking for E02 and ignores stale E01 failure=%s",
    async (fails) => {
      const { wrapper, toolbar, draft } = await setup();
      const old = deferred<never>();
      const latest = deferred<never>();
      vi.mocked(api.tvShowDetail)
        .mockReturnValueOnce(old.promise)
        .mockReturnValueOnce(latest.promise);
      await wrapper.setProps({ scanRevision: 1 });
      await wrapper.setProps({ selection: episode(8) });
      expect(toolbar().props("mutationsBlocked")).toBe(true);
      if (fails) old.reject(new RequestError("old selection missing", 404));
      else old.resolve({ ...record(), episodes: [] } as never);
      await flushPromises();
      expect(api.tvShowDetail).toHaveBeenCalledTimes(3);
      expect(wrapper.text()).not.toContain("no longer in the library");
      expect(toolbar().props("mutationsBlocked")).toBe(true);
      latest.resolve(showRecord() as never);
      await flushPromises();
      expect(toolbar().props("mutationsBlocked")).toBe(false);
      expect(draft.title).toBe("Keep same-show draft");
      expect(toolbar().props("isEditing")).toBe(true);
      wrapper.unmount();
    },
  );

  it.each(["episode", "season"] as const)(
    "does not carry missing status onto a valid %s",
    async (kind) => {
      const { wrapper, toolbar, draft } = await setup();
      vi.mocked(api.tvShowDetail).mockResolvedValueOnce({
        ...record(),
        episodes: [],
      } as never);
      await wrapper.setProps({ scanRevision: 1 });
      await flushPromises();
      expect(wrapper.text()).toContain("no longer in the library");
      await wrapper.setProps({
        selection:
          kind === "episode"
            ? episode(8)
            : { kind: "season", showId: 1, seasonNumber: 2 },
      });
      await flushPromises();
      expect(toolbar().props("mutationsBlocked")).toBe(false);
      expect(draft.title).toBe("Keep same-show draft");
      wrapper.unmount();
    },
  );

  it("does not apply mutation detail data after same-show navigation", async () => {
    const { wrapper, toolbar, draft } = await setup();
    vi.mocked(api.previewTVNfoPlans).mockResolvedValue({} as never);
    const detail = deferred<never>();
    vi.mocked(api.tvShowDetail).mockReturnValueOnce(detail.promise);
    toolbar().vm.$emit("saveEdit");
    await flushPromises();
    await wrapper.setProps({ selection: episode(8) });
    await flushPromises();
    detail.resolve({
      ...showRecord(),
      metadata: { title: "Obsolete mutation result" },
    } as never);
    await flushPromises();
    expect(draft.title).toBe("Keep same-show draft");
    expect(toolbar().props("isSaving")).toBe(false);
    expect(wrapper.emitted("metadataSaved")).toBeUndefined();
    wrapper.unmount();
  });

  it("ignores a pending mutation on disposal", async () => {
    const { wrapper, toolbar } = await setup();
    const mutation = deferred<never>();
    vi.mocked(api.previewTVNfoPlans).mockReturnValueOnce(mutation.promise);
    toolbar().vm.$emit("saveEdit");
    await flushPromises();
    wrapper.unmount();
    mutation.reject(new Error("late save failure"));
    await flushPromises();
    expect(api.tvShowDetail).toHaveBeenCalledTimes(1);
    expect(wrapper.emitted("metadataSaved")).toBeUndefined();
  });

  it("coalesces rapid episode/season switches and disposal stops queued checks", async () => {
    const { wrapper } = await setup();
    const old = deferred<never>();
    vi.mocked(api.tvShowDetail).mockReturnValueOnce(old.promise);
    await wrapper.setProps({ scanRevision: 1 });
    await wrapper.setProps({ selection: episode(8) });
    await wrapper.setProps({
      selection: { kind: "season", showId: 1, seasonNumber: 2 },
    });
    const signal = vi.mocked(api.tvShowDetail).mock.calls.at(-1)?.[1];
    wrapper.unmount();
    expect(signal?.aborted).toBe(true);
    old.reject(new RequestError("late", 404));
    await flushPromises();
    expect(api.tvShowDetail).toHaveBeenCalledTimes(2);
  });

  it.each(["save", "episode scrape", "season scrape", "candidate"] as const)(
    "releases token-owned busy state after %s across selection changes",
    async (operation) => {
      for (const fails of [false, true]) {
        vi.clearAllMocks();
        const { wrapper, toolbar } = await setup();
        if (operation === "season scrape") {
          await wrapper.setProps({
            selection: { kind: "season", showId: 1, seasonNumber: 1 },
          });
          await flushPromises();
        }
        const mutate = vi.mocked(
          operation === "save"
            ? api.previewTVNfoPlans
            : operation === "episode scrape"
              ? api.scrapeTVEpisode
              : operation === "season scrape"
                ? api.scrapeTVSeason
                : api.selectTVShowCandidate,
        );
        const old = deferred<never>();
        mutate.mockReturnValueOnce(old.promise);
        const start = () =>
          operation === "candidate"
            ? wrapper
                .findComponent({ name: "ScraperModal" })
                .vm.$emit("select", { id: "candidate" })
            : toolbar().vm.$emit(operation === "save" ? "saveEdit" : "scrape");
        if (operation === "candidate") {
          await wrapper.setProps({ selection: { kind: "show", showId: 1 } });
          await flushPromises();
          toolbar().vm.$emit("scrape");
          await flushPromises();
        }
        start();
        await flushPromises();
        expect(toolbar().props("isSaving")).toBe(true);
        await wrapper.setProps({ selection: episode(8) });
        await flushPromises();
        // Duplicate UI events cannot start a newer operation while the owner is pending.
        toolbar().vm.$emit("saveEdit");
        await flushPromises();
        expect(api.previewTVNfoPlans).toHaveBeenCalledTimes(
          operation === "save" ? 1 : 0,
        );
        if (fails) old.reject(new Error("old mutation failed"));
        else old.resolve({} as never);
        await flushPromises();
        expect(toolbar().props("isSaving")).toBe(false);
        expect(wrapper.emitted("metadataSaved")).toBeUndefined();
        expect(wrapper.text()).not.toContain("old mutation failed");
        const newer = deferred<never>();
        vi.mocked(api.previewTVNfoPlans).mockReturnValueOnce(newer.promise);
        toolbar().vm.$emit("saveEdit");
        await flushPromises();
        expect(toolbar().props("isSaving")).toBe(true);
        await wrapper.setProps({ scanRevision: 1 });
        await flushPromises();
        expect(toolbar().props("isSaving")).toBe(true);
        newer.resolve({} as never);
        await flushPromises();
        expect(toolbar().props("isSaving")).toBe(false);
        expect(wrapper.emitted("metadataSaved")).toHaveLength(1);
        wrapper.unmount();
      }
    },
  );
});

describe("movie save-and-write availability ordering", () => {
  async function setup() {
    vi.mocked(api.mediaDetail).mockResolvedValue(record() as never);
    vi.mocked(api.previewNfo).mockResolvedValue({ id: "plan" } as never);
    const wrapper = shallowMount(MovieInspector, {
      props: { itemId: 1, scanRevision: 0, csrfToken: "", labels: {} },
    });
    await flushPromises();
    const toolbar = wrapper.findComponent({ name: "InspectorToolbar" });
    toolbar.vm.$emit("toggleEdit");
    await flushPromises();
    const draft = wrapper
      .findComponent({ name: "MovieOverviewTab" })
      .props("draft");
    draft.title = "Save draft";
    const save = deferred<never>();
    const check = deferred<never>();
    vi.mocked(api.saveMetadata).mockReturnValueOnce(save.promise);
    toolbar.vm.$emit("saveEdit");
    await flushPromises();
    vi.mocked(api.mediaDetail).mockReturnValueOnce(check.promise);
    await wrapper.setProps({ scanRevision: 1 });
    save.resolve({} as never);
    await flushPromises();
    expect(toolbar.props("isSaving")).toBe(true);
    expect(api.previewNfo).not.toHaveBeenCalled();
    return { wrapper, toolbar, draft, check };
  }

  it("waits for an independent check then writes exactly once without waiting for draft replacement", async () => {
    const { wrapper, toolbar, check } = await setup();
    check.resolve(record() as never);
    await flushPromises();
    expect(api.previewNfo).toHaveBeenCalledTimes(1);
    expect(api.previewNfo).toHaveBeenCalledWith(
      "",
      1,
      expect.objectContaining({ title: "Save draft" }),
    );
    expect(api.applyNfo).toHaveBeenCalledTimes(1);
    expect(toolbar.props("isSaving")).toBe(false);
    expect(wrapper.emitted("metadataSaved")).toHaveLength(1);
    wrapper.unmount();
  });

  it.each([404, 500])(
    "reports saved-metadata-only result after unavailable check %s",
    async (status) => {
      const { wrapper, toolbar, draft, check } = await setup();
      check.reject(new RequestError("unavailable", status));
      await flushPromises();
      expect(api.previewNfo).not.toHaveBeenCalled();
      expect(api.applyNfo).not.toHaveBeenCalled();
      expect(wrapper.text()).toContain(
        "Metadata saved, but NFO was not written",
      );
      expect(draft.title).toBe("Save draft");
      expect(toolbar.props("isEditing")).toBe(true);
      expect(toolbar.props("isSaving")).toBe(false);
      expect(wrapper.emitted("metadataSaved")).toBeUndefined();
      wrapper.unmount();
    },
  );

  it("waits through a superseding scan check before deciding availability", async () => {
    const { wrapper, toolbar, check } = await setup();
    const latest = deferred<never>();
    vi.mocked(api.mediaDetail).mockReturnValueOnce(latest.promise);
    await wrapper.setProps({ scanRevision: 2 });
    check.resolve(record() as never);
    await flushPromises();
    expect(api.previewNfo).not.toHaveBeenCalled();
    expect(toolbar.props("isSaving")).toBe(true);
    latest.resolve(record() as never);
    await flushPromises();
    expect(api.previewNfo).toHaveBeenCalledTimes(1);
    expect(api.applyNfo).toHaveBeenCalledTimes(1);
    wrapper.unmount();
  });

  it("reports the safe partial result when the availability deadline aborts", async () => {
    vi.useFakeTimers();
    const { wrapper, toolbar, check } = await setup();
    const signal = vi.mocked(api.mediaDetail).mock.calls.at(-1)?.[1];
    signal?.addEventListener("abort", () =>
      check.reject(new DOMException("Aborted", "AbortError")),
    );
    await vi.advanceTimersByTimeAsync(15_000);
    await flushPromises();
    expect(signal?.aborted).toBe(true);
    expect(toolbar.props("isSaving")).toBe(false);
    expect(wrapper.text()).toContain("Metadata saved, but NFO was not written");
    expect(api.previewNfo).not.toHaveBeenCalled();
    wrapper.unmount();
  });

  it("disposal settles the waiting continuation without writing or late refresh", async () => {
    const { wrapper, check } = await setup();
    wrapper.unmount();
    await flushPromises();
    check.resolve(record() as never);
    await flushPromises();
    expect(api.previewNfo).not.toHaveBeenCalled();
    expect(api.applyNfo).not.toHaveBeenCalled();
    expect(api.mediaDetail).toHaveBeenCalledTimes(2);
    expect(wrapper.emitted("metadataSaved")).toBeUndefined();
  });
});
