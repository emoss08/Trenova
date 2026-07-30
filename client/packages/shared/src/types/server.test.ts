import { describe, expect, it } from "vitest";
import { z } from "zod";
import { createLimitOffsetResponse } from "@trenova/shared/types/server";

describe("createLimitOffsetResponse", () => {
  it("accepts null limit-offset cursors from REST and GraphQL list responses", () => {
    const schema = createLimitOffsetResponse(z.object({ id: z.string() }));

    expect(
      schema.parse({
        results: [{ id: "row_1" }],
        count: 1,
        next: null,
        prev: null,
      }),
    ).toEqual({
      results: [{ id: "row_1" }],
      count: 1,
      next: null,
      prev: null,
    });
  });

  it("defaults omitted limit-offset cursors to null", () => {
    const schema = createLimitOffsetResponse(z.object({ id: z.string() }));

    expect(
      schema.parse({
        results: [{ id: "row_1" }],
        count: 1,
      }),
    ).toEqual({
      results: [{ id: "row_1" }],
      count: 1,
      next: null,
      prev: null,
    });
  });

  it("preserves GraphQL cursor pageInfo when present", () => {
    const schema = createLimitOffsetResponse(z.object({ id: z.string() }));

    expect(
      schema.parse({
        results: [{ id: "row_1" }],
        count: 25,
        next: "20",
        prev: null,
        pageInfo: {
          mode: "cursor",
          hasNextPage: true,
          endCursor: "cursor_1",
          totalCount: 25,
        },
      }),
    ).toEqual({
      results: [{ id: "row_1" }],
      count: 25,
      next: "20",
      prev: null,
      pageInfo: {
        mode: "cursor",
        hasNextPage: true,
        endCursor: "cursor_1",
        totalCount: 25,
      },
    });
  });

  it("preserves REST cursor totalCount when present", () => {
    const schema = createLimitOffsetResponse(z.object({ id: z.string() }));

    expect(
      schema.parse({
        results: [{ id: "row_1" }],
        count: 25,
        totalCount: 25,
        next: null,
        prev: null,
      }),
    ).toEqual({
      results: [{ id: "row_1" }],
      count: 25,
      totalCount: 25,
      next: null,
      prev: null,
    });
  });

  it("preserves offset-backed GraphQL pageInfo when present", () => {
    const schema = createLimitOffsetResponse(z.object({ id: z.string() }));

    expect(
      schema.parse({
        results: [{ id: "row_1" }],
        count: 25,
        next: "20",
        prev: null,
        pageInfo: {
          mode: "offset",
          hasNextPage: true,
          endCursor: "cursor_1",
          totalCount: 25,
        },
      }),
    ).toEqual({
      results: [{ id: "row_1" }],
      count: 25,
      next: "20",
      prev: null,
      pageInfo: {
        mode: "offset",
        hasNextPage: true,
        endCursor: "cursor_1",
        totalCount: 25,
      },
    });
  });
});

describe("createLimitOffsetResponse collection shapes", () => {
  const schema = createLimitOffsetResponse(z.object({ id: z.string() }));

  it("accepts an empty page", () => {
    expect(schema.parse({ results: [], count: 0, next: null, prev: null })).toEqual({
      results: [],
      count: 0,
      next: null,
      prev: null,
    });
  });

  it("accepts an empty page that still reports a non-zero total", () => {
    const parsed = schema.parse({ results: [], count: 250, next: "20", prev: null });

    expect(parsed.results).toEqual([]);
    expect(parsed.count).toBe(250);
  });

  it("accepts many rows and preserves their order", () => {
    const parsed = schema.parse({
      results: [{ id: "row_1" }, { id: "row_2" }, { id: "row_3" }],
      count: 3,
      next: null,
      prev: null,
    });

    expect(parsed.results.map((row) => row.id)).toEqual(["row_1", "row_2", "row_3"]);
  });

  it("does not require results.length to agree with count", () => {
    const parsed = schema.parse({
      results: [{ id: "row_1" }, { id: "row_2" }],
      count: 137,
      next: "20",
      prev: null,
    });

    expect(parsed.results).toHaveLength(2);
    expect(parsed.count).toBe(137);
  });

  it("accepts a last page whose cursor is null", () => {
    const parsed = schema.parse({
      results: [{ id: "row_1" }],
      count: 1,
      next: null,
      prev: "0",
      pageInfo: { mode: "cursor", hasNextPage: false, endCursor: null, totalCount: 1 },
    });

    expect(parsed.pageInfo).toEqual({
      mode: "cursor",
      hasNextPage: false,
      endCursor: null,
      totalCount: 1,
    });
  });

  it("rejects a pageInfo that omits endCursor", () => {
    expect(() =>
      schema.parse({
        results: [{ id: "row_1" }],
        count: 1,
        pageInfo: { mode: "cursor", hasNextPage: false, totalCount: 1 },
      }),
    ).toThrow();
  });

  it("rejects a pageInfo that omits totalCount", () => {
    expect(() =>
      schema.parse({
        results: [{ id: "row_1" }],
        count: 1,
        pageInfo: { mode: "cursor", hasNextPage: false, endCursor: null },
      }),
    ).toThrow();
  });

  it("rejects a pageInfo that omits mode", () => {
    expect(() =>
      schema.parse({
        results: [{ id: "row_1" }],
        count: 1,
        pageInfo: { hasNextPage: false, endCursor: null, totalCount: 1 },
      }),
    ).toThrow();
  });
});
