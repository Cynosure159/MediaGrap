import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import MediaCatalog from "./MediaCatalog.vue";

const movies = Array.from({ length: 500 }, (_, i) => ({
  id: i + 1,
  sourceId: 1,
  relativePath: `Movie${i}.mkv`,
  titleHint: `Movie ${i}`,
  title: `Movie ${i}`,
  yearHint: 2024,
  fileSize: 1,
  modifiedAt: "",
  sidecars: [],
  posterUrl: "/poster.jpg",
}));

describe("MediaCatalog", () => {
  it("shows the server total while bounding rows and lazy images", async () => {
    const wrapper = mount(MediaCatalog, {
      props: {
        items: movies,
        total: 10000,
        selectedId: null,
        activeJob: undefined,
        labels: { movies: "Movies" },
        hasMore: true,
      },
    });
    expect(wrapper.get(".count-main").text()).toBe("10000 Movies");
    expect(wrapper.findAll(".media-row").length).toBeLessThan(30);
    expect(wrapper.get("img").attributes("loading")).toBe("lazy");
    const list = wrapper.get(".catalog-list");
    Object.defineProperty(list.element, "scrollTop", {
      value: 29000,
      configurable: true,
    });
    Object.defineProperty(list.element, "clientHeight", {
      value: 600,
      configurable: true,
    });
    Object.defineProperty(list.element, "scrollHeight", {
      value: 30000,
      configurable: true,
    });
    await list.trigger("scroll");
    expect(wrapper.text()).toContain("Movie 480");
    expect(wrapper.findAll(".media-row").length).toBeLessThan(30);
    Object.defineProperty(list.element, "scrollTop", {
      value: 29400,
      configurable: true,
    });
    await list.trigger("scroll");
    expect(wrapper.emitted("loadMore")).toHaveLength(1);
    wrapper.unmount();
  });

  it("offers retry without automatic retry loops", async () => {
    const wrapper = mount(MediaCatalog, {
      props: {
        items: [],
        total: null,
        selectedId: null,
        activeJob: undefined,
        labels: { catalogRetry: "Retry" },
        loadError: "offline",
        hasMore: true,
      },
    });
    await wrapper.get(".load-status button").trigger("click");
    expect(wrapper.emitted("loadMore")).toHaveLength(1);
    expect(wrapper.find(".catalog-empty").exists()).toBe(false);
    wrapper.unmount();
  });
  it("queues a scan when the refresh button is clicked", async () => {
    const wrapper = mount(MediaCatalog, {
      props: {
        items: [],
        selectedId: null,
        activeJob: undefined,
        labels: { refresh: "Refresh", movies: "Movies" },
      },
    });

    await wrapper.get('[title="Refresh"]').trigger("click");

    expect(wrapper.emitted("scan")).toHaveLength(1);
  });
});
