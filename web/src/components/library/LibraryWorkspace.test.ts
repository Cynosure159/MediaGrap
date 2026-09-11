import { flushPromises, mount } from "@vue/test-utils";
import { shallowRef } from "vue";
import { describe, expect, it, vi } from "vitest";
import { useLibrary } from "@/composables/useLibrary";
import LibraryWorkspace from "./LibraryWorkspace.vue";

vi.mock("@/composables/useLibrary", () => ({ useLibrary: vi.fn() }));

describe("catalog scanning", () => {
  it("refreshes the TV list after inspector metadata writes", async () => {
    const refreshTVShows = vi.fn();
    vi.mocked(useLibrary).mockReturnValue({
      sourceItems: shallowRef([]),
      mediaItems: shallowRef([]),
      tvShowItems: shallowRef([]),
      jobItems: shallowRef([]),
      error: shallowRef(null),
      hasSources: shallowRef(true),
      refresh: vi.fn(),
      refreshTVShows,
    } as unknown as ReturnType<typeof useLibrary>);
    const wrapper = mount(LibraryWorkspace, {
      props: { mediaKind: "shows", csrfToken: "", username: "", labels: {} },
      global: {
        stubs: {
          TVShowCatalog: true,
          TVShowInspector: {
            props: ["labels"],
            emits: ["metadataSaved"],
            template:
              "<button @click=\"$emit('metadataSaved')\">Saved</button>",
          },
        },
      },
    });
    await wrapper.get("button").trigger("click");
    expect(refreshTVShows).toHaveBeenCalledWith("");
    wrapper.unmount();
  });
  it.each(["movies", "shows"] as const)(
    "scans all enabled sources from %s without duplicate clicks",
    async (mediaKind) => {
      let finish!: () => void;
      const scan = vi
        .fn()
        .mockImplementationOnce(
          () =>
            new Promise<void>((resolve) => {
              finish = resolve;
            }),
        )
        .mockResolvedValue(undefined);
      vi.mocked(useLibrary).mockReturnValue({
        sourceItems: shallowRef([
          { id: 1, enabled: true },
          { id: 2, enabled: true },
          { id: 3, enabled: false },
        ]),
        mediaItems: shallowRef([]),
        tvShowItems: shallowRef([]),
        jobItems: shallowRef([]),
        error: shallowRef(null),
        hasSources: shallowRef(true),
        refresh: vi.fn(),
        scan,
      } as unknown as ReturnType<typeof useLibrary>);
      const catalog = {
        props: ["labels"],
        emits: ["scan"],
        template: "<button @click=\"$emit('scan')\">Scan</button>",
      };
      const inspectorMounted = vi.fn();
      const inspector = { props: ['labels'], setup() { inspectorMounted(); return {} }, template: '<div />' };
      const wrapper = mount(LibraryWorkspace, {
        props: { mediaKind, csrfToken: "csrf", username: "admin", labels: {} },
        global: {
          stubs: {
            MediaCatalog: catalog,
            TVShowCatalog: catalog,
            MovieInspector: inspector,
            TVShowInspector: inspector,
          },
        },
      });
      await flushPromises();
      await wrapper.get("button").trigger("click");
      await wrapper.get("button").trigger("click");
      expect(scan).toHaveBeenCalledTimes(1);
      expect(inspectorMounted).toHaveBeenCalledTimes(1);
      finish();
      await flushPromises();
      expect(inspectorMounted).toHaveBeenCalledTimes(3);
      expect(scan.mock.calls).toEqual([
        [1, ""],
        [2, ""],
      ]);
      wrapper.unmount();
    },
  );
});
