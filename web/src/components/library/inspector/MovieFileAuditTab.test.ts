import source from "./MovieFileAuditTab.vue?raw";
import { describe, expect, it } from "vitest";
import { mount } from "@vue/test-utils";
import MovieFileAuditTab from "./MovieFileAuditTab.vue";
import type { MediaInspection, MediaItem } from "@/api/types";

const labels = new Proxy<Record<string, string>>(
  {},
  { get: (_, key) => String(key) },
);
const item: MediaItem = {
  id: 7,
  sourceId: 1,
  relativePath: "Movie.mkv",
  titleHint: "Movie",
  yearHint: 2024,
  fileSize: 100,
  modifiedAt: "",
  sidecars: [],
};
const inspection: MediaInspection = {
  probeStatus: "ready",
  cached: true,
  probedAt: "",
  format: { name: "matroska", durationSeconds: 120, bitRate: 1000 },
  video: [
    {
      index: 0,
      codec: "hevc",
      width: 3840,
      height: 2160,
      bitDepth: 10,
      hdr: "HDR10",
    },
  ],
  audio: [
    {
      index: 1,
      codec: "eac3",
      channels: 6,
      channelLayout: "5.1",
      language: "eng",
    },
  ],
  subtitles: [{ index: 2, codec: "subrip", language: "zho" }],
  files: [
    {
      relativePath: "Movie.mkv",
      kind: "video",
      size: 100,
      mimeType: "video/x-matroska",
      modifiedAt: "",
      permissions: "-rw-r-----",
      writable: true,
      regular: true,
      symlink: false,
      valid: true,
      warnings: [],
    },
  ],
};

describe("MovieFileAuditTab", () => {
  it("keeps mobile rename actions above navigation with a full-width touch target", () => {
    // JSDOM has no layout: guard the responsive rules here; Ego checks actual pointer hit testing.
    const navStyles = source
      .split("@media (max-width: 768px) {")[1]
      ?.split("@media")[0];
    expect(navStyles).toContain(
      ".sticky-action-bar { bottom: calc(60px + env(safe-area-inset-bottom)); }",
    );
    const touchStyles = source.split("@media (max-width: 700px) {")[1];
    expect(touchStyles).toContain(".action-bar-right { flex-wrap: wrap; }");
    expect(touchStyles).toContain(
      ".action-bar-right .btn { flex: 1; min-height: 44px; }",
    );
    expect(touchStyles).toContain(
      ".action-bar-right .btn-success { flex-basis: 100%; }",
    );
  });

  it("renders real file and stream values and emits a read-only preview request", async () => {
    const wrapper = mount(MovieFileAuditTab, {
      props: {
        item,
        labels,
        inspection,
        inspectionLoading: false,
        inspectionError: null,
        namingPreview: null,
        previewLoading: false,
      },
    });
    expect(wrapper.text()).toContain("3840×2160");
    expect(wrapper.text()).toContain("video/x-matroska");
    expect(wrapper.text()).toContain("HDR10");

    await wrapper.find(".btn-primary").trigger("click");
    expect(wrapper.emitted("previewRename")?.[0]).toEqual([
      "${title} (${year})/${title} (${year})",
    ]);
  });

  it("renders read-only supplemental samples and extras with bounded status", async () => {
    const supplemental: MediaInspection = {
      ...inspection,
      supplementalStatus: "ready",
      supplementalFiles: [
        {
          ...inspection.files[0],
          relativePath: "Movie/Extras/Trailer.mkv",
          kind: "extra",
          size: 2048,
          modifiedAt: "2026-09-02T16:00:00Z",
          mimeType: "video/x-matroska",
        },
        {
          ...inspection.files[0],
          relativePath: "Movie-sample.mkv",
          kind: "sample",
        },
      ],
    };
    const wrapper = mount(MovieFileAuditTab, {
      props: { item, labels, inspection: supplemental },
    });
    expect(wrapper.find(".supplemental-card").text()).toContain("Trailer.mkv");
    expect(wrapper.find(".supplemental-card").text()).toContain("extra");
    expect(wrapper.find(".supplemental-card").text()).toContain("sample");
    expect(wrapper.findAll(".supplemental-card button")).toHaveLength(0);

    await wrapper.setProps({
      inspection: {
        ...supplemental,
        supplementalStatus: "truncated",
      },
    });
    expect(wrapper.find(".supplemental-status").text()).toContain(
      "supplementalTruncated",
    );
    expect(wrapper.findAll(".supplemental-table tbody tr")).toHaveLength(2);
    expect(wrapper.find(".supplemental-table").text()).toContain("Trailer.mkv");
    expect(wrapper.findAll(".supplemental-card button")).toHaveLength(0);
  });

  it("keeps the supplemental section backward compatible when the response omits it", () => {
    const wrapper = mount(MovieFileAuditTab, {
      props: { item, labels, inspection },
    });
    expect(wrapper.find(".supplemental-card").exists()).toBe(true);
    expect(wrapper.find(".supplemental-card").text()).toContain(
      "supplementalNone",
    );
  });

  it("shows filesystem safety warnings", () => {
    const warned: MediaInspection = {
      ...inspection,
      files: [
        {
          ...inspection.files[0],
          symlink: true,
          valid: false,
          warnings: ["symlink"],
        },
      ],
    };
    const wrapper = mount(MovieFileAuditTab, {
      props: {
        item,
        labels,
        inspection: warned,
        inspectionLoading: false,
        inspectionError: null,
        namingPreview: null,
        previewLoading: false,
      },
    });
    expect(wrapper.text()).toContain("audit_symlink");
    expect(wrapper.find(".audit-row--warning").exists()).toBe(true);
  });

  it("renders rename plan and emits apply-rename when confirmed", async () => {
    const renamePlan = {
      id: "plan-123",
      mediaItemId: 7,
      pattern: "${title} (${year})/${title} (${year})",
      state: "previewed" as const,
      hasConflicts: false,
      warnings: [],
      createdAt: "2026-09-02T16:00:00Z",
      items: [
        {
          kind: "directory",
          currentPath: "Movie",
          plannedPath: "Movie (2024)",
          operation: "rename_dir" as const,
          conflict: false,
          status: "pending",
        },
        {
          kind: "video",
          currentPath: "Movie/Movie.mkv",
          plannedPath: "Movie (2024)/Movie (2024).mkv",
          operation: "rename" as const,
          conflict: false,
          status: "pending",
        },
      ],
    };

    const wrapper = mount(MovieFileAuditTab, {
      props: {
        item,
        labels,
        inspection,
        inspectionLoading: false,
        inspectionError: null,
        renamePlan,
        previewLoading: false,
        isApplying: false,
      },
    });

    expect(wrapper.text()).toContain("Movie (2024)");
    expect(wrapper.text()).toContain("noConflictsDetected");
    expect(wrapper.find(".sticky-action-bar").exists()).toBe(true);

    await wrapper.find(".btn-success").trigger("click");
    expect(wrapper.emitted("applyRename")).toBeTruthy();

    await wrapper.setProps({ isApplying: true });
    expect(wrapper.get(".btn-success").attributes("disabled")).toBeDefined();
    await wrapper.get(".btn-success").trigger("click");
    expect(wrapper.emitted("applyRename")).toHaveLength(1);

    await wrapper.setProps({
      isApplying: false,
      renamePlan: { ...renamePlan, hasConflicts: true },
    });
    expect(wrapper.find(".sticky-action-bar").exists()).toBe(false);

    await wrapper.setProps({ renamePlan });
    await wrapper.get(".sticky-action-bar .btn-ghost").trigger("click");
    expect(wrapper.emitted("clearRename")).toHaveLength(1);
    expect(wrapper.find(".sticky-action-bar").exists()).toBe(false);
  });

  it("updates pattern when preset is changed", async () => {
    const wrapper = mount(MovieFileAuditTab, {
      props: {
        item,
        labels,
        inspection,
        inspectionLoading: false,
        inspectionError: null,
        previewLoading: false,
      },
    });

    const select = wrapper.find(".preset-select");
    await select.setValue("plex");
    expect(
      (wrapper.find(".pattern-input").element as HTMLInputElement).value,
    ).toBe("${title} (${year})/${title} (${year})");

    await wrapper.find(".btn-primary").trigger("click");
    expect(wrapper.emitted("previewRename")?.[0]).toEqual([
      "${title} (${year})/${title} (${year})",
    ]);
  });
});
