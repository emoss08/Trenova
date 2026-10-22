import type { ArtifactRows } from "@/types/assistant";
import { describe, expect, it } from "vitest";
import {
  agentPages,
  agentShare,
  isPageSize,
  pageCount,
  pageSlice,
  positionAfter,
  railPages,
  RAIL_MOST,
} from "../table-pages";

function rows(page: Partial<ArtifactRows>): ArtifactRows {
  return {
    rows: [],
    hasMore: false,
    nextCursor: "",
    nextOffset: null,
    total: null,
    withheld: [],
    searchable: false,
    sortable: [],
    ...page,
  };
}

describe("positionAfter", () => {
  it("goes on after the cursor a page hands out", () => {
    expect(positionAfter(rows({ hasMore: true, nextCursor: "c-50" }))).toEqual({
      after: "c-50",
      first: false,
    });
  });

  it("goes on at the offset a page hands out", () => {
    expect(positionAfter(rows({ hasMore: true, nextOffset: 50 }))).toEqual({
      offset: 50,
      first: false,
    });
  });

  it("stops at the last page, whatever it says of what follows", () => {
    expect(positionAfter(rows({ hasMore: false, nextCursor: "c-50" }))).toBeNull();
  });

  it("stops when a page says there is more but not where", () => {
    expect(positionAfter(rows({ hasMore: true }))).toBeNull();
  });
});

describe("pageCount", () => {
  it("counts the pages of a list whose total is known", () => {
    expect(pageCount({ total: 128, read: 25, hasMore: true, size: 25 })).toEqual({
      pages: 6,
      exact: true,
    });
  });

  it("counts one page for an empty list", () => {
    expect(pageCount({ total: 0, read: 0, hasMore: false, size: 25 })).toEqual({
      pages: 1,
      exact: true,
    });
  });

  it("counts the rows read when nothing follows them", () => {
    expect(pageCount({ total: null, read: 51, hasMore: false, size: 25 })).toEqual({
      pages: 3,
      exact: true,
    });
  });

  it("counts a page more than the full pages read when more follow", () => {
    expect(pageCount({ total: null, read: 50, hasMore: true, size: 25 })).toEqual({
      pages: 3,
      exact: false,
    });
  });

  it("counts a part page read as the last known when more follow", () => {
    expect(pageCount({ total: null, read: 60, hasMore: true, size: 25 })).toEqual({
      pages: 3,
      exact: false,
    });
  });
});

describe("pageSlice", () => {
  it("takes the rows of a page", () => {
    const all = Array.from({ length: 60 }, (_, index) => index);

    expect(pageSlice(all, 2, 25)).toEqual(Array.from({ length: 10 }, (_, index) => 50 + index));
    expect(pageSlice(all, 3, 25)).toEqual([]);
  });
});

describe("agentShare", () => {
  it("says a page is all, part or none of the rows the agent read", () => {
    expect(agentShare(0, 25, 25, 25)).toBe("all");
    expect(agentShare(0, 50, 25, 50)).toBe("part");
    expect(agentShare(1, 25, 25, 25)).toBe("none");
    expect(agentShare(0, 25, 0, 25)).toBe("none");
    expect(agentShare(0, 25, 25, 0)).toBe("none");
  });

  it("counts a part page among the agent's", () => {
    expect(agentPages(30, 25)).toBe(2);
    expect(agentPages(0, 25)).toBe(0);
  });
});

describe("railPages", () => {
  it("marks the page open, the agent's and the ones read", () => {
    expect(
      railPages({
        pages: 4,
        current: 1,
        size: 25,
        agentRows: 25,
        readRows: 50,
        lockedFrom: null,
      }),
    ).toEqual([
      { index: 0, current: false, agent: true, read: true, locked: false },
      { index: 1, current: true, agent: false, read: true, locked: false },
      { index: 2, current: false, agent: false, read: false, locked: false },
      { index: 3, current: false, agent: false, read: false, locked: false },
    ]);
  });

  it("locks the pages past what the person may read", () => {
    const rail = railPages({
      pages: 3,
      current: 0,
      size: 25,
      agentRows: 25,
      readRows: 25,
      lockedFrom: 1,
    });

    expect(rail.map((segment) => segment.locked)).toEqual([false, true, true]);
  });

  it("shows the window around the page open in a long list", () => {
    const rail = railPages({
      pages: 40,
      current: 30,
      size: 25,
      agentRows: 25,
      readRows: 775,
      lockedFrom: null,
    });

    expect(rail).toHaveLength(RAIL_MOST);
    expect(rail[0].index).toBe(24);
    expect(rail.find((segment) => segment.current)?.index).toBe(30);
    expect(rail.at(-1)?.index).toBe(35);
  });

  it("keeps the window inside the list at its end", () => {
    const rail = railPages({
      pages: 20,
      current: 19,
      size: 25,
      agentRows: 0,
      readRows: 500,
      lockedFrom: null,
    });

    expect(rail[0].index).toBe(8);
    expect(rail.at(-1)?.index).toBe(19);
  });
});

describe("isPageSize", () => {
  it("accepts only the sizes offered", () => {
    expect(isPageSize(25)).toBe(true);
    expect(isPageSize(50)).toBe(true);
    expect(isPageSize(30)).toBe(false);
    expect(isPageSize("25")).toBe(false);
  });
});
