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
});
