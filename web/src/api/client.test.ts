import { afterEach, describe, expect, it, vi } from "vitest";
import { request } from "./client";

describe("api client request", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it.each([404, 500])(
    "retains definitive HTTP status %s independently of error message",
    async (status) => {
      vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
        new Response("{}", { status }),
      );
      await expect(request("/api/v1/test")).rejects.toMatchObject({ status });
    },
  );

  it("performs successful json request", async () => {
    const mockData = { id: 1, name: "Test" };
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify(mockData), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );

    const result = await request<typeof mockData>("/api/v1/test");
    expect(result).toEqual(mockData);
  });

  it("handles 204 No Content response properly", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(null, { status: 204 }),
    );

    const result = await request<void>("/api/v1/test", { method: "DELETE" });
    expect(result).toBeUndefined();
  });

  it("throws error with message from api response on failure", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(JSON.stringify({ error: { message: "Custom API Error" } }), {
        status: 400,
        headers: { "Content-Type": "application/json" },
      }),
    );

    await expect(request("/api/v1/test")).rejects.toThrow("Custom API Error");
  });

  it("throws generic error when response is not JSON on failure", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response("Internal Server Error", { status: 500 }),
    );

    await expect(request("/api/v1/test")).rejects.toThrow(
      "Request failed with status 500",
    );
  });
});
