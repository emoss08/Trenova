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
