import { describe, expect, it, vi } from "vitest";

import { requestGraphQL } from "@trenova/shared/lib/graphql";

import { GlobalSearchService } from "./global-search";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const emptyResult = { globalSearch: { query: "sam", groups: [] } };

describe("GlobalSearchService", () => {
  it("sends the entity types with the search", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue(emptyResult);

    const service = new GlobalSearchService();

    await service.search("sam", 4, ["worker", "customer"]);

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        operationName: "GlobalSearch",
        variables: { input: { query: "sam", limit: 4, entityTypes: ["worker", "customer"] } },
      }),
    );
  });

  it("omits entity types when none are provided", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue(emptyResult);

    const service = new GlobalSearchService();

    await service.search("sam", 4);

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({
        variables: { input: { query: "sam", limit: 4, entityTypes: null } },
      }),
    );
  });

  it("forwards the abort signal so a superseded search is cancelled", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue(emptyResult);
    const controller = new AbortController();

    const service = new GlobalSearchService();

    await service.search("sam", 4, undefined, { signal: controller.signal });

    expect(requestGraphQL).toHaveBeenCalledWith(
      expect.objectContaining({ signal: controller.signal }),
    );
  });

  it("keeps the hit shape the palette reads", async () => {
    vi.mocked(requestGraphQL).mockResolvedValue({
      globalSearch: {
        query: "acme",
        groups: [
          {
            entityType: "customer",
            label: "Customers",
            total: 3,
            hits: [
              {
                id: "cus_1",
                entityType: "customer",
                title: "Acme",
                subtitle: "",
                href: "/customers/cus_1",
                metadata: { Balance: "$10.00" },
              },
            ],
          },
        ],
      },
    });

    const result = await new GlobalSearchService().search("acme");

    expect(result.groups[0].hits[0]).toEqual({
      id: "cus_1",
      entityType: "customer",
      title: "Acme",
      subtitle: undefined,
      href: "/customers/cus_1",
      metadata: { Balance: "$10.00" },
    });
  });
});
