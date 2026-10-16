import { ScimDirectoriesListDocument } from "@trenova/graphql/generated/graphql";
import { requestGraphQL } from "@trenova/shared/lib/graphql";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { listSCIMDirectoriesGraphQL } from "../scim-directory";

vi.mock("@trenova/shared/lib/graphql", () => ({
  requestGraphQL: vi.fn(),
}));

const requestGraphQLMock = vi.mocked(requestGraphQL);

type Call = {
  document: unknown;
  operationName: string;
  signal?: AbortSignal;
  variables: { input: Record<string, unknown> };
};

function lastCall(): Call {
  return requestGraphQLMock.mock.calls.at(-1)?.[0] as unknown as Call;
}

/** A directory as the `scimDirectories` connection returns it (security_activity.graphqls). */
function directory(id: string, tenantSlug: string) {
  return {
    id,
    organizationId: "org_01",
    businessUnitId: "bu_01",
    tenantSlug,
    enabled: true,
    createdAt: 1_760_000_000,
    updatedAt: 1_760_000_100,
  };
}

beforeEach(() => {
  requestGraphQLMock.mockReset();
});

describe("listSCIMDirectoriesGraphQL", () => {
  it("asks for the first page by size alone, with no cursor and no offset", async () => {
    requestGraphQLMock.mockResolvedValue({
      scimDirectories: {
        edges: [{ node: directory("sd_1", "acme") }, { node: directory("sd_2", "acme-eu") }],
        pageInfo: { hasNextPage: true, endCursor: "c2" },
      },
    } as never);
    const controller = new AbortController();

    const page = await listSCIMDirectoriesGraphQL({ first: 20 }, { signal: controller.signal });

    expect(lastCall()).toEqual({
      document: ScimDirectoriesListDocument,
      operationName: "SCIMDirectoriesList",
      variables: { input: { first: 20 } },
      signal: controller.signal,
    });
    expect(page).toEqual({
      directories: [directory("sd_1", "acme"), directory("sd_2", "acme-eu")],
      hasNextPage: true,
      endCursor: "c2",
    });
  });

  it("continues from the previous page's cursor", async () => {
    requestGraphQLMock.mockResolvedValue({
      scimDirectories: {
        edges: [{ node: directory("sd_3", "acme-apac") }],
        pageInfo: { hasNextPage: false, endCursor: null },
      },
    } as never);

    const page = await listSCIMDirectoriesGraphQL({ first: 20, after: "c2" });

    expect(lastCall().variables).toEqual({ input: { first: 20, after: "c2" } });
    expect(page.hasNextPage).toBe(false);
    expect(page.endCursor).toBeNull();
  });

  it("reads an empty organization as an empty last page", async () => {
    requestGraphQLMock.mockResolvedValue({
      scimDirectories: { edges: [], pageInfo: { hasNextPage: false, endCursor: null } },
    } as never);

    expect(await listSCIMDirectoriesGraphQL({ first: 20 })).toEqual({
      directories: [],
      hasNextPage: false,
      endCursor: null,
    });
  });
});
