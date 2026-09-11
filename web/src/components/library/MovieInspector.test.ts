import { afterEach, describe, expect, it, vi } from "vitest";
import { flushPromises, shallowMount } from "@vue/test-utils";
import * as api from "@/api/library";
import MovieInspector from "./MovieInspector.vue";

vi.mock("@/api/library", () => ({
  mediaDetail: vi.fn(),
  mediaInspection: vi.fn().mockResolvedValue(null),
}));
const labels = {
  invalidNfoFallback: "NFO unavailable; showing indexed data",
  errorLoadMedia: "Failed",
  loading: "Loading",
};
function detail(
  id: number,
  warning = "",
): Awaited<ReturnType<typeof api.mediaDetail>> {
  return {
    item: {
      id,
      sourceId: 1,
      relativePath: `Movie${id}.mkv`,
      titleHint: `Movie ${id}`,
      yearHint: 2022,
      fileSize: 1,
      modifiedAt: "",
      sidecars: [],
      posterUrl: `/poster/${id}`,
    },
    metadata: {
      mediaItemId: id,
      provider: "",
      providerId: "",
      title: "",
      originalTitle: "",
      year: null,
      overview: "",
      runtimeMinutes: null,
      genres: [],
      posterUrl: "",
      backdropUrl: "",
      directors: [],
      writers: [],
      studios: [],
      cast: [],
      lockedFields: [],
      rating: null,
      votes: null,
      contentRating: "",
    },
    metadataOrigin: "empty",
    metadataWarning: warning,
    writable: true,
  };
}
function mount() {
  return shallowMount(MovieInspector, {
    props: { itemId: 1, csrfToken: "", labels },
  });
}
afterEach(() => vi.clearAllMocks());

describe("movie identity and degraded NFO detail", () => {
  it("clears the prior movie immediately and does not revive it on failure", async () => {
    vi.mocked(api.mediaDetail)
      .mockResolvedValueOnce(detail(1))
      .mockRejectedValueOnce(new Error("offline"));
    const wrapper = mount();
    await flushPromises();
    expect(
      wrapper.findComponent({ name: "MovieOverviewTab" }).props("item").id,
    ).toBe(1);
    await wrapper.setProps({ itemId: 2 });
    await flushPromises();
    expect(wrapper.findComponent({ name: "MovieOverviewTab" }).exists()).toBe(
      false,
    );
    expect(
      wrapper.findComponent({ name: "InspectorToolbar" }).props("hasDetail"),
    ).toBe(false);
    expect(wrapper.text()).toContain("offline");
    wrapper.unmount();
  });

  it("renders the selected movie hints and poster with a non-blocking NFO warning", async () => {
    vi.mocked(api.mediaDetail)
      .mockResolvedValueOnce(detail(1))
      .mockResolvedValueOnce(detail(2, "invalid_nfo"));
    const wrapper = mount();
    await flushPromises();
    await wrapper.setProps({ itemId: 2 });
    await flushPromises();
    const overview = wrapper.findComponent({ name: "MovieOverviewTab" });
    expect(overview.props("item").id).toBe(2);
    expect(overview.props("draft")).toMatchObject({
      title: "Movie 2",
      year: 2022,
      posterUrl: "/poster/2",
      overview: "",
      cast: [],
    });
    expect(wrapper.get('[role="status"]').text()).toBe(
      labels.invalidNfoFallback,
    );
    expect(wrapper.find(".inspector-error").exists()).toBe(false);
    wrapper.unmount();
  });

  it("aborts old requests and ignores late responses, including after deselection", async () => {
    let resolve!: (value: ReturnType<typeof detail>) => void;
    vi.mocked(api.mediaDetail)
      .mockImplementationOnce(
        () =>
          new Promise((r) => {
            resolve = r;
          }),
      )
      .mockResolvedValueOnce(detail(2));
    const wrapper = mount();
    const signal = vi.mocked(api.mediaDetail).mock.calls[0]?.[1];
    await wrapper.setProps({ itemId: 2 });
    await flushPromises();
    expect(signal?.aborted).toBe(true);
    resolve(detail(1));
    await flushPromises();
    expect(
      wrapper.findComponent({ name: "MovieOverviewTab" }).props("item").id,
    ).toBe(2);
    await wrapper.setProps({ itemId: null });
    await flushPromises();
    expect(wrapper.findComponent({ name: "MovieOverviewTab" }).exists()).toBe(
      false,
    );
    wrapper.unmount();
  });
});
