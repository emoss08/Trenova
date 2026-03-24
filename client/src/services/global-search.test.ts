import { describe, expect, it, vi } from "vitest";

import { api } from "@/lib/api";

import { GlobalSearchService } from "./global-search";

vi.mock("@/lib/api", () => ({
  api: {
    get: vi.fn(),
  },
}));

describe("GlobalSearchService", () => {
  it("serializes entity types into the search query", async () => {
    vi.mocked(api.get).mockResolvedValue({ query: "sam", groups: [] });

    const service = new GlobalSearchService();

    await service.search("sam", 4, ["worker", "customer"]);

    expect(api.get).toHaveBeenCalledWith(
      "/search/global/?query=sam&limit=4&entityTypes=worker%2Ccustomer",
    );
  });

  it("omits entity types when none are provided", async () => {
    vi.mocked(api.get).mockResolvedValue({ query: "sam", groups: [] });

    const service = new GlobalSearchService();

    await service.search("sam", 4);

    expect(api.get).toHaveBeenCalledWith("/search/global/?query=sam&limit=4");
  });
});
