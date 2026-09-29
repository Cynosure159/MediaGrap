import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import TVShowCatalog from "./TVShowCatalog.vue";

describe("TVShowCatalog", () => {
  it("uses refreshed metadata titles for rows, tooltips and search while retaining filename hints", async () => {
    const show = {
      id: 1,
      sourceId: 1,
      relativePath: "Original Folder",
      titleHint: "Original Name",
      yearHint: null,
      seasonCount: 1,
      episodeCount: 1,
    };
    const wrapper = mount(TVShowCatalog, {
      props: {
        items: [show],
        selected: null,
        activeJob: undefined,
        labels: {},
      },
    });
    expect(wrapper.get(".title-primary").text()).toBe("Original Name");
    await wrapper.setProps({ items: [{ ...show, title: "刮削后的剧名" }] });
    expect(wrapper.get(".title-primary").text()).toBe("刮削后的剧名");
    expect(wrapper.get(".title-primary").attributes("title")).toBe(
      "刮削后的剧名",
    );
    await wrapper.get("input").setValue("刮削后的剧名");
    expect(wrapper.find(".title-primary").exists()).toBe(true);
    await wrapper.get("input").setValue("Original Name");
    expect(wrapper.find(".title-primary").exists()).toBe(true);
    wrapper.unmount();
  });
  it("queues a scan when the refresh button is clicked", async () => {
    const wrapper = mount(TVShowCatalog, {
      props: {
        items: [],
        selected: null,
        activeJob: undefined,
        labels: { refresh: "Refresh", tvShows: "TV shows" },
      },
    });

    await wrapper.get('[title="Refresh"]').trigger("click");

    expect(wrapper.emitted("scan")).toHaveLength(1);
  });

  it("scrolls active TV show into view when selected or items change", async () => {
    const shows = Array.from({ length: 20 }, (_, i) => ({
      id: i + 1,
      sourceId: 1,
      relativePath: `Show ${i + 1}`,
      titleHint: `Show ${i + 1}`,
      yearHint: 2024,
      seasonCount: 1,
      episodeCount: 1,
    }));
    const wrapper = mount(TVShowCatalog, {
      props: {
        items: shows,
        selected: null,
        activeJob: undefined,
        labels: {},
      },
    });

    const list = wrapper.get(".catalog-list");
    Object.defineProperty(list.element, "clientHeight", {
      value: 400,
      configurable: true,
    });
    let scrolledTop = 0;
    Object.defineProperty(list.element, "scrollTop", {
      get: () => scrolledTop,
      set: (val: number) => { scrolledTop = val; },
      configurable: true,
    });

    await wrapper.setProps({ selected: { kind: "show", showId: 15 } });
    await new Promise((r) => setTimeout(r, 10));
    wrapper.unmount();
  });
});
