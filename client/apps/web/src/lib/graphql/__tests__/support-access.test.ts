import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  fetchCurrentSupportSession,
  fetchSupportAccessActions,
  fetchSupportAccessGrants,
} from "../support-access";

const mocks = vi.hoisted(() => ({ requestGraphQL: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql", () => ({ requestGraphQL: mocks.requestGraphQL }));

const emptyPage = (key: string, hasNextPage = false, endCursor: string | null = null) => ({
  [key]: { edges: [], pageInfo: { hasNextPage, endCursor } },
});

describe("support access reads", () => {
  beforeEach(() => {
    mocks.requestGraphQL.mockReset();
  });

  it("lists what was done under a grant by the grant's id as the audit principal", async () => {
    mocks.requestGraphQL.mockResolvedValue(emptyPage("auditEntries"));

    await fetchSupportAccessActions("sag_01JGRANT", null);

    const request = mocks.requestGraphQL.mock.calls[0]?.[0] as {
      operationName: string;
      variables: { input: { fieldFilters: unknown[]; after: string | null } };
    };
    expect(request.operationName).toBe("SupportAccessActions");
    expect(request.variables.input.fieldFilters).toEqual([
      { field: "principalId", operator: "eq", value: "sag_01JGRANT" },
    ]);
    expect(request.variables.input.after).toBeNull();
  });

  it("hands on the next cursor only while the server says there is a next page", async () => {
    mocks.requestGraphQL.mockResolvedValueOnce(emptyPage("supportAccessGrants", true, "cur_2"));
    mocks.requestGraphQL.mockResolvedValueOnce(emptyPage("supportAccessGrants", false, "cur_3"));

    expect((await fetchSupportAccessGrants(null)).nextCursor).toBe("cur_2");
    expect((await fetchSupportAccessGrants("cur_2")).nextCursor).toBeNull();
  });

  it("reports no support session when the server answers null", async () => {
    mocks.requestGraphQL.mockResolvedValue({ currentSupportSession: null });

    await expect(fetchCurrentSupportSession()).resolves.toBeNull();
  });
});
