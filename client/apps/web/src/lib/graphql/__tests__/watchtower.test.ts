import { beforeEach, describe, expect, it, vi } from "vitest";
import { fetchWatchtowerFeed, resolveWatchtowerItem, snoozeWatchtowerItem } from "../watchtower";

const mocks = vi.hoisted(() => ({ requestGraphQL: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql", () => ({ requestGraphQL: mocks.requestGraphQL }));

const requestOf = (call: number) =>
  mocks.requestGraphQL.mock.calls[call][0] as {
    variables: Record<string, unknown>;
    idempotencyKey?: string;
  };
const variablesOf = (call: number) => requestOf(call).variables;

describe("watchtower exception changes", () => {
  beforeEach(() => {
    mocks.requestGraphQL.mockReset();
    mocks.requestGraphQL.mockResolvedValue({
      resolveWatchtowerItem: { id: "wt_1" },
      snoozeWatchtowerItem: { id: "wt_1" },
    });
  });

  it("trims the note it resolves with and sends the version the person saw", async () => {
    await resolveWatchtowerItem({ id: "wt_1", version: 3, code: "Other", note: "  Phoned in " });

    expect(variablesOf(0)).toEqual({
      id: "wt_1",
      input: { code: "Other", note: "Phoned in", expectedVersion: 3 },
    });
  });

  it("snoozes with no note when the note is blank, under the key it was given", async () => {
    await snoozeWatchtowerItem(
      { id: "wt_1", version: 4, until: 1_800_000_000, note: "   " },
      { idempotencyKey: "key-1" },
    );

    expect(variablesOf(0)).toEqual({
      id: "wt_1",
      input: { until: 1_800_000_000, note: null, expectedVersion: 4 },
    });
    expect(requestOf(0).idempotencyKey).toBe("key-1");
  });

  it("sends no key when none is given", async () => {
    await resolveWatchtowerItem({ id: "wt_1", version: 1, code: "Fixed" });

    expect(requestOf(0).idempotencyKey).toBeUndefined();
  });
});

describe("fetchWatchtowerFeed", () => {
  beforeEach(() => {
    mocks.requestGraphQL.mockReset();
    mocks.requestGraphQL.mockResolvedValue({
      watchtowerItems: {
        edges: [],
        pageInfo: { hasNextPage: false, endCursor: null },
        seenAt: 0,
      },
    });
  });

  it("asks for the open feed, newest first, when no view is chosen", async () => {
    await fetchWatchtowerFeed();

    expect(variablesOf(0)).toEqual({
      input: {
        first: 25,
        after: null,
        kinds: [],
        severities: [],
        unresolvedOnly: true,
        statuses: [],
        assignedToMe: false,
        unassigned: false,
        overdueOnly: false,
        subjectId: null,
        orderByDue: false,
      },
    });
  });

  it("passes a work list's owner, overdue and order through", async () => {
    await fetchWatchtowerFeed({ assignedToMe: true, overdueOnly: true, orderByDue: true });

    const input = variablesOf(0).input as Record<string, unknown>;
    expect(input.assignedToMe).toBe(true);
    expect(input.overdueOnly).toBe(true);
    expect(input.orderByDue).toBe(true);
  });
});
