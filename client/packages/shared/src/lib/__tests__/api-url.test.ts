import { describe, expect, it, vi } from "vitest";

vi.mock("../constants", () => ({ API_BASE_URL: "https://api.trenova.test/api/v1" }));

const { apiUrl } = await import("../api-url");

describe("apiUrl", () => {
  it("swaps the server's prefix for the configured base", () => {
    expect(apiUrl("/api/v1/capture/pages/cpg_1/content/?kind=thumbnail")).toBe(
      "https://api.trenova.test/api/v1/capture/pages/cpg_1/content/?kind=thumbnail",
    );
  });

  it("joins a path the server wrote without the prefix", () => {
    expect(apiUrl("capture/pages/cpg_1/content/")).toBe(
      "https://api.trenova.test/api/v1/capture/pages/cpg_1/content/",
    );
  });
});
