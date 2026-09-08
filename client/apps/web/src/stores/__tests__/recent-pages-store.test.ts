import { renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";
import { RECENT_PAGES_LIMIT, useRecentPages, useRecentPagesStore } from "../recent-pages-store";

const ORG_A = "org_a";
const ORG_B = "org_b";

function pathsFor(organizationId: string | undefined): string[] {
  const { result } = renderHook(() => useRecentPages(organizationId));
  return result.current.map((page) => page.path);
}

describe("recent pages store", () => {
  beforeEach(() => {
    useRecentPagesStore.setState({ pagesByOrganization: {} });
  });

  it("records a visit with the newest page first", () => {
    const { recordVisit } = useRecentPagesStore.getState();
    recordVisit(ORG_A, { path: "/billing/queue", title: "Billing Queue" });
    recordVisit(ORG_A, { path: "/hr/workers", title: "Workers" });

    expect(pathsFor(ORG_A)).toEqual(["/hr/workers", "/billing/queue"]);
  });

  it("keeps each organization's history apart", () => {
    const { recordVisit } = useRecentPagesStore.getState();
    recordVisit(ORG_A, { path: "/billing/queue", title: "Billing Queue" });
    recordVisit(ORG_B, { path: "/hr/workers", title: "Workers" });

    expect(pathsFor(ORG_A)).toEqual(["/billing/queue"]);
    expect(pathsFor(ORG_B)).toEqual(["/hr/workers"]);
  });

  it("moves a revisited page to the front instead of duplicating it", () => {
    const { recordVisit } = useRecentPagesStore.getState();
    recordVisit(ORG_A, { path: "/billing/queue", title: "Billing Queue" });
    recordVisit(ORG_A, { path: "/hr/workers", title: "Workers" });
    recordVisit(ORG_A, { path: "/billing/queue", title: "Billing Queue" });

    expect(pathsFor(ORG_A)).toEqual(["/billing/queue", "/hr/workers"]);
  });

  it("keeps the latest title for a page whose title changed", () => {
    const { recordVisit } = useRecentPagesStore.getState();
    recordVisit(ORG_A, { path: "/hr/workers/wrk_1", title: "Details" });
    recordVisit(ORG_A, { path: "/hr/workers/wrk_1", title: "Maria Alvarez" });

    const { result } = renderHook(() => useRecentPages(ORG_A));
    expect(result.current[0].title).toBe("Maria Alvarez");
  });

  it("ignores the home page, empty titles and visits with no organization", () => {
    const { recordVisit } = useRecentPagesStore.getState();
    recordVisit(ORG_A, { path: "/", title: "Home" });
    recordVisit(ORG_A, { path: "/hr/workers", title: "" });
    recordVisit(ORG_A, { path: "/hr/workers", title: "   " });
    recordVisit(undefined, { path: "/hr/workers", title: "Workers" });

    expect(pathsFor(ORG_A)).toEqual([]);
    expect(useRecentPagesStore.getState().pagesByOrganization).toEqual({});
  });

  it("returns nothing for an unknown or missing organization", () => {
    const { recordVisit } = useRecentPagesStore.getState();
    recordVisit(ORG_A, { path: "/billing/queue", title: "Billing Queue" });

    expect(pathsFor(ORG_B)).toEqual([]);
    expect(pathsFor(undefined)).toEqual([]);
  });

  it("caps each organization's list at the limit, dropping the oldest", () => {
    const { recordVisit } = useRecentPagesStore.getState();
    for (let index = 0; index <= RECENT_PAGES_LIMIT; index += 1) {
      recordVisit(ORG_A, { path: `/page/${index}`, title: `Page ${index}` });
    }

    const paths = pathsFor(ORG_A);
    expect(paths).toHaveLength(RECENT_PAGES_LIMIT);
    expect(paths[0]).toBe(`/page/${RECENT_PAGES_LIMIT}`);
    expect(paths).not.toContain("/page/0");
  });

  it("forgets a page for one organization only", () => {
    const { recordVisit, forget } = useRecentPagesStore.getState();
    recordVisit(ORG_A, { path: "/billing/queue", title: "Billing Queue" });
    recordVisit(ORG_B, { path: "/billing/queue", title: "Billing Queue" });
    forget(ORG_A, "/billing/queue");

    expect(pathsFor(ORG_A)).toEqual([]);
    expect(pathsFor(ORG_B)).toEqual(["/billing/queue"]);
  });
});
