import { describe, expect, it } from "vitest";
import {
  appendReplyToPages,
  bumpReplyCountInPages,
  classifyCommentCacheKey,
  commentClientRef,
  insertCommentIntoPages,
  normalizeCommentFilter,
  patchCommentInPages,
  removeCommentFromPages,
  syncPinnedPage,
  type CommentInfiniteData,
  type CommentPage,
  type LocalShipmentComment,
} from "@/lib/shipment-comment-cache";

function makeComment(overrides: Partial<LocalShipmentComment> = {}): LocalShipmentComment {
  return {
    id: "shc_1",
    shipmentId: "shp_1",
    userId: "usr_1",
    parentCommentId: null,
    replyCount: 0,
    comment: "hello",
    body: null,
    type: "Internal",
    visibility: "Internal",
    priority: "Normal",
    source: "User",
    metadata: {},
    editedAt: null,
    pinnedAt: null,
    pinnedById: null,
    resolvedAt: null,
    resolvedById: null,
    requiresAcknowledgment: false,
    deletedAt: null,
    version: 0,
    createdAt: 100,
    updatedAt: 100,
    mentionedUserIds: [],
    user: null,
    pinnedBy: null,
    resolvedBy: null,
    mentionedUsers: null,
    acknowledgments: null,
    attachments: null,
    ...overrides,
  };
}

function makePage(results: LocalShipmentComment[], count = results.length): CommentPage {
  return {
    results,
    count,
    next: null,
    prev: null,
    pageInfo: { mode: "cursor", hasNextPage: false, endCursor: null, totalCount: count },
  };
}

function makeInfinite(pages: CommentPage[]): CommentInfiniteData {
  return { pages, pageParams: pages.map(() => null) };
}

describe("normalizeCommentFilter", () => {
  it("returns null for empty filters", () => {
    expect(normalizeCommentFilter(null)).toBeNull();
    expect(normalizeCommentFilter({})).toBeNull();
    expect(normalizeCommentFilter({ types: [], search: "  " })).toBeNull();
  });

  it("keeps active filters", () => {
    expect(normalizeCommentFilter({ unresolvedOnly: true })).toEqual({ unresolvedOnly: true });
    expect(normalizeCommentFilter({ search: "late" })).toEqual({ search: "late" });
  });
});

describe("insertCommentIntoPages", () => {
  it("prepends a new comment to the newest page and bumps counts", () => {
    const existing = makeComment({ id: "shc_old" });
    const data = makeInfinite([makePage([existing], 5)]);

    const result = insertCommentIntoPages(data, makeComment({ id: "shc_new" }));

    expect(result.inserted).toBe(true);
    expect(result.data.pages[0].results.map((r) => r.id)).toEqual(["shc_new", "shc_old"]);
    expect(result.data.pages[0].count).toBe(6);
    expect(result.data.pages[0].pageInfo?.totalCount).toBe(6);
  });

  it("is idempotent by id", () => {
    const existing = makeComment({ id: "shc_1" });
    const data = makeInfinite([makePage([existing], 3)]);

    const result = insertCommentIntoPages(data, makeComment({ id: "shc_1", comment: "updated" }));

    expect(result.inserted).toBe(false);
    expect(result.data.pages[0].results).toHaveLength(1);
    expect(result.data.pages[0].results[0].comment).toBe("updated");
    expect(result.data.pages[0].count).toBe(3);
  });

  it("replaces an optimistic row by clientRef (echo-before-response ordering)", () => {
    const optimistic = makeComment({
      id: "optimistic-abc",
      metadata: { clientRef: "abc" },
      pending: true,
    });
    const data = makeInfinite([makePage([optimistic], 1)]);

    const echo = makeComment({ id: "shc_real", metadata: { clientRef: "abc" } });
    const result = insertCommentIntoPages(data, echo);

    expect(result.inserted).toBe(false);
    expect(result.data.pages[0].results).toHaveLength(1);
    expect(result.data.pages[0].results[0].id).toBe("shc_real");
    expect(result.data.pages[0].count).toBe(1);
  });

  it("treats a later echo as a no-op after the response replaced the row (response-first ordering)", () => {
    const confirmed = makeComment({ id: "shc_real", metadata: { clientRef: "abc" } });
    const data = makeInfinite([makePage([confirmed], 1)]);

    const echo = makeComment({ id: "shc_real", metadata: { clientRef: "abc" } });
    const result = insertCommentIntoPages(data, echo);

    expect(result.inserted).toBe(false);
    expect(result.data.pages[0].results).toHaveLength(1);
  });

  it("creates a first page when the cache is empty", () => {
    const result = insertCommentIntoPages(makeInfinite([]), makeComment());
    expect(result.inserted).toBe(true);
    expect(result.data.pages[0].results).toHaveLength(1);
  });
});

describe("appendReplyToPages", () => {
  it("appends to the last page in ascending order", () => {
    const first = makeComment({ id: "shc_r1" });
    const data = makeInfinite([makePage([first], 1)]);

    const result = appendReplyToPages(data, makeComment({ id: "shc_r2" }));

    expect(result.inserted).toBe(true);
    expect(result.data.pages[0].results.map((r) => r.id)).toEqual(["shc_r1", "shc_r2"]);
  });

  it("dedupes replies by clientRef", () => {
    const optimistic = makeComment({
      id: "optimistic-xyz",
      metadata: { clientRef: "xyz" },
      pending: true,
    });
    const data = makeInfinite([makePage([optimistic], 1)]);

    const result = appendReplyToPages(
      data,
      makeComment({ id: "shc_real", metadata: { clientRef: "xyz" } }),
    );

    expect(result.inserted).toBe(false);
    expect(result.data.pages[0].results[0].id).toBe("shc_real");
  });
});

describe("patchCommentInPages / removeCommentFromPages", () => {
  it("patches a comment across pages", () => {
    const data = makeInfinite([
      makePage([makeComment({ id: "shc_a" })]),
      makePage([makeComment({ id: "shc_b" })]),
    ]);

    const result = patchCommentInPages(data, "shc_b", { resolvedAt: 500 });

    expect(result.patched).toBe(true);
    expect(result.data.pages[1].results[0].resolvedAt).toBe(500);
  });

  it("reports untouched data when the comment is absent", () => {
    const data = makeInfinite([makePage([makeComment({ id: "shc_a" })])]);
    const result = patchCommentInPages(data, "shc_missing", { resolvedAt: 500 });
    expect(result.patched).toBe(false);
    expect(result.data).toBe(data);
  });

  it("removes a comment and decrements counts", () => {
    const data = makeInfinite([
      makePage([makeComment({ id: "shc_a" }), makeComment({ id: "shc_b" })], 10),
    ]);

    const result = removeCommentFromPages(data, "shc_a");

    expect(result.removed).toBe(true);
    expect(result.data.pages[0].results.map((r) => r.id)).toEqual(["shc_b"]);
    expect(result.data.pages[0].count).toBe(9);
  });
});

describe("bumpReplyCountInPages", () => {
  it("bumps and floors at zero", () => {
    const data = makeInfinite([makePage([makeComment({ id: "shc_p", replyCount: 1 })])]);

    const up = bumpReplyCountInPages(data, "shc_p", 1);
    expect(up.data.pages[0].results[0].replyCount).toBe(2);

    const down = bumpReplyCountInPages(up.data, "shc_p", -5);
    expect(down.data.pages[0].results[0].replyCount).toBe(0);
  });
});

describe("syncPinnedPage", () => {
  it("inserts newly pinned comments at the front", () => {
    const page = makePage([makeComment({ id: "shc_a", pinnedAt: 100 })], 1);
    const next = syncPinnedPage(page, makeComment({ id: "shc_b", pinnedAt: 200 }));
    expect(next.results.map((r) => r.id)).toEqual(["shc_b", "shc_a"]);
    expect(next.count).toBe(2);
  });

  it("removes unpinned and tombstoned comments", () => {
    const page = makePage([makeComment({ id: "shc_a", pinnedAt: 100 })], 1);

    const unpinned = syncPinnedPage(page, makeComment({ id: "shc_a", pinnedAt: null }));
    expect(unpinned.results).toHaveLength(0);

    const tombstoned = syncPinnedPage(
      page,
      makeComment({ id: "shc_a", pinnedAt: 100, deletedAt: 900 }),
    );
    expect(tombstoned.results).toHaveLength(0);
  });

  it("updates an already-pinned comment in place", () => {
    const page = makePage([makeComment({ id: "shc_a", pinnedAt: 100 })], 1);
    const next = syncPinnedPage(
      page,
      makeComment({ id: "shc_a", pinnedAt: 100, comment: "edited" }),
    );
    expect(next.results[0].comment).toBe("edited");
    expect(next.count).toBe(1);
  });
});

describe("classifyCommentCacheKey", () => {
  it("classifies every cache shape", () => {
    expect(classifyCommentCacheKey(["shipment-comments", "shp_1", "list", null])).toEqual({
      kind: "list",
      filtered: false,
    });
    expect(
      classifyCommentCacheKey(["shipment-comments", "shp_1", "list", { search: "x" }]),
    ).toEqual({ kind: "list", filtered: true });
    expect(classifyCommentCacheKey(["shipment-comments", "shp_1", "replies", "shc_9"])).toEqual({
      kind: "replies",
      parentCommentId: "shc_9",
    });
    expect(classifyCommentCacheKey(["shipment-comments", "shp_1", "pinned"])).toEqual({
      kind: "pinned",
    });
    expect(classifyCommentCacheKey(["shipment-comments", "shp_1", "compact"])).toEqual({
      kind: "compact",
    });
    expect(classifyCommentCacheKey(["shipment-comments", "shp_1"])).toEqual({ kind: "unknown" });
  });
});

describe("commentClientRef", () => {
  it("reads the clientRef from metadata", () => {
    expect(commentClientRef(makeComment({ metadata: { clientRef: "abc" } }))).toBe("abc");
    expect(commentClientRef(makeComment({ metadata: {} }))).toBeNull();
    expect(commentClientRef(makeComment({ metadata: null }))).toBeNull();
  });
});
