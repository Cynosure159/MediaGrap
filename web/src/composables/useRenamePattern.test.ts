import { flushPromises, mount } from "@vue/test-utils";
import { defineComponent } from "vue";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { settings } from "@/api/library";
import type { Settings } from "@/api/types";
import fixtures from "./renamePatternFixtures.json";
import {
  renderRenamePattern,
  simulateRenamePattern,
  useRenamePattern,
} from "./useRenamePattern";
vi.mock("@/api/library", () => ({ settings: vi.fn() }));
beforeEach(() => vi.clearAllMocks());
const host = (kind: "movie" | "tv") =>
  defineComponent({
    setup: () => useRenamePattern(kind),
    template: '<input v-model="pattern" />',
  });
const movieTokens = ["title", "year", "edition", "resolution"] as const;

describe("rename pattern grammar", () => {
  it.each(fixtures)("matches shared fixture: $name", (fixture) => {
    if (fixture.error) {
      expect(() =>
        renderRenamePattern(
          fixture.pattern,
          fixture.values as Record<string, string>,
          fixture.allowed,
        ),
      ).toThrow();
      return;
    }
    expect(
      renderRenamePattern(
        fixture.pattern,
        fixture.values as Record<string, string>,
        fixture.allowed,
      ),
    ).toBe(fixture.rendered);
  });

  it("matches Go whitespace, control, and UTF-8 byte limits", () => {
    expect(
      renderRenamePattern("${title}", { title: "\uFEFF" }, ["title"]),
    ).toBe("\uFEFF");
    expect(() =>
      renderRenamePattern("${\u007f,title,}", { title: "Film" }, ["title"]),
    ).toThrow();
    expect(() =>
      renderRenamePattern("影".repeat(342), {}, ["title"]),
    ).toThrow();
  });

  it("checks presence before sanitizing metadata separators", () => {
    expect(
      renderRenamePattern(
        "${title}${ (,originalTitle,)}",
        { title: "Film", originalTitle: "/" },
        ["title", "originalTitle"],
      ),
    ).toBe("Film ( )");
    expect(() =>
      renderRenamePattern("${originalTitle}", { originalTitle: "/" }, [
        "originalTitle",
      ]),
    ).not.toThrow();
  });
  it("treats whitespace-only values as missing", () => {
    expect(() =>
      renderRenamePattern("${title}", { title: " \t\n" }, ["title"]),
    ).toThrow("no available value");
    expect(
      renderRenamePattern("${,edition,}", { edition: " \t\n" }, ["edition"]),
    ).toBe("");
  });
  it("rejects empty paths after optional omission and sanitization", () => {
    expect(
      simulateRenamePattern(
        "${title}/${,edition,}",
        { title: "Film", edition: "" },
        movieTokens,
      ),
    ).toMatchObject({ errorCode: "empty_filename" });
    expect(
      simulateRenamePattern(
        "${,edition,}/${title}",
        { title: "Film", edition: "" },
        movieTokens,
      ),
    ).toMatchObject({ errorCode: "invalid_directory_segment" });
    expect(
      simulateRenamePattern("${,edition,}", { edition: "/" }, movieTokens),
    ).toMatchObject({ errorCode: "empty_filename" });
  });
  it("renders optional affixes only when values are available", () => {
    expect(
      renderRenamePattern(
        "${title}${ - ,edition,}${ (,year,)}",
        { title: "Film", edition: "", year: "0" },
        movieTokens,
      ),
    ).toBe("Film (0)");
    expect(
      renderRenamePattern(
        "${title}${ - ,edition,}${ [,resolution,]}",
        { title: "Film", edition: "Cut", resolution: "1080p" },
        movieTokens,
      ),
    ).toBe("Film - Cut [1080p]");
  });

  it("keeps strict tokens strict and reports malformed or unsafe syntax", () => {
    expect(() =>
      renderRenamePattern("${edition}", { edition: "" }, movieTokens),
    ).toThrow("no available value");
    expect(() => renderRenamePattern("${unknown}", {}, movieTokens)).toThrow(
      "unsupported naming token",
    );
    expect(() => renderRenamePattern("${a,b}", {}, movieTokens)).toThrow(
      "exactly two commas",
    );
    expect(() =>
      renderRenamePattern("${x,edition,/}", {}, movieTokens),
    ).toThrow("safe filename");
  });

  it("does not allow metadata separators to create directories", () => {
    const preview = simulateRenamePattern(
      "${title}${,edition,}",
      { title: "A/B", edition: "0" },
      movieTokens,
    );
    expect(preview.dir).toBe("");
    expect(preview.filename).toBe("A B0.mkv");
  });
});

describe("saved rename defaults", () => {
  it.each(["movie", "tv"] as const)("loads the %s default", async (kind) => {
    vi.mocked(settings).mockResolvedValue({
      movieRenamePattern: "Movie/${title}",
      tvRenamePattern: "TV/${showTitle}",
    } as Settings);
    const wrapper = mount(host(kind));
    await flushPromises();
    expect(wrapper.get("input").element.value).toBe(
      kind === "movie" ? "Movie/${title}" : "TV/${showTitle}",
    );
    wrapper.unmount();
  });
  it("does not overwrite edits when settings arrive late", async () => {
    let resolve!: (value: Settings) => void;
    vi.mocked(settings).mockReturnValue(
      new Promise((done) => {
        resolve = done;
      }),
    );
    const wrapper = mount(host("movie"));
    await wrapper.get("input").setValue("My/${title}");
    resolve({ movieRenamePattern: "Saved/${title}" } as Settings);
    await flushPromises();
    expect(wrapper.get("input").element.value).toBe("My/${title}");
    wrapper.unmount();
  });
});
