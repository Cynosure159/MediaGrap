import { shallowMount, flushPromises } from "@vue/test-utils";
import { describe, expect, it, vi } from "vitest";
import App from "./App.vue";

vi.mock("vue-router", () => ({
  RouterView: { template: "<div />" },
  useRoute: () => ({ meta: {} }),
  useRouter: () => ({ push: vi.fn() }),
}));
vi.mock("@/api/library", () => ({
  setupStatus: async () => ({ needsSetup: true }),
}));
vi.mock("@/composables/useTheme", () => ({
  useTheme: () => ({ theme: "dark", setTheme: vi.fn() }),
}));

describe("App same-build source link", () => {
  it("offers a normal download before authentication, not a SPA route", async () => {
    const wrapper = shallowMount(App);
    await flushPromises();
    const link = wrapper.get("a.source-download");
    expect(link.attributes("href")).toBe("/source");
    expect(link.attributes()).toHaveProperty("download");
    expect(link.text()).toContain("AGPLv3");
    wrapper.unmount();
  });
});
