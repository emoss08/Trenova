import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  assignWatchtowerItem,
  fetchWatchtowerFeed,
  resolveWatchtowerItem,
  setWatchtowerItemDue,
} from "../watchtower";

const mocks = vi.hoisted(() => ({ requestGraphQL: vi.fn() }));

vi.mock("@trenova/shared/lib/graphql", () => ({ requestGraphQL: mocks.requestGraphQL }));

const variablesOf = (call: number) =>
  (mocks.requestGraphQL.mock.calls[call][0] as { variables: Record<string, unknown> }).variables;

describe("watchtower exception changes", () => {
  beforeEach(() => {
    mocks.requestGraphQL.mockReset();
    mocks.requestGraphQL.mockResolvedValue({
      assignWatchtowerItem: { id: "wt_1" },
      resolveWatchtowerItem: { id: "wt_1" },
      setWatchtowerItemDue: { id: "wt_1" },
    });
  });

  it("sends the version the person saw and no note when the note is blank", async () => {
    await assignWatchtowerItem({ id: "wt_1", version: 4, assigneeId: "usr_9", note: "   " });

    expect(variablesOf(0)).toEqual({
      id: "wt_1",
      input: { assigneeId: "usr_9", note: null, expectedVersion: 4 },
    });
  });

  it("leaves an exception with nobody when no owner is chosen", async () => {
    await assignWatchtowerItem({ id: "wt_1", version: 1, assigneeId: null });

    expect(variablesOf(0)).toEqual({
      id: "wt_1",
      input: { assigneeId: null, note: null, expectedVersion: 1 },
    });
  });

  it("clears a due time with an explicit null", async () => {
    await setWatchtowerItemDue({ id: "wt_1", version: 2, dueAt: null });

    expect(variablesOf(0)).toEqual({
      id: "wt_1",
      input: { dueAt: null, note: null, expectedVersion: 2 },
    });
  });

  it("trims the note it resolves with", async () => {
    await resolveWatchtowerItem({ id: "wt_1", version: 3, code: "Other", note: "  Phoned in " });

    expect(variablesOf(0)).toEqual({
      id: "wt_1",
      input: { code: "Other", note: "Phoned in", expectedVersion: 3 },
    });
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
