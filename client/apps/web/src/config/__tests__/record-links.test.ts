import {
  RECORD_LINKS,
  isRecordEntityType,
  recordAtLocation,
  recordPath,
  type RecordEntityType,
} from "@/config/record-links";
import { routes } from "@/router";
import type { RouteObject } from "react-router";
import { describe, expect, it } from "vitest";

function joinPath(parent: string, child: string): string {
  return child.startsWith("/") ? child : `${parent.replace(/\/$/, "")}/${child}`;
}

function registered(entries: RouteObject[], parent = "", found: string[] = []): string[] {
  for (const entry of entries) {
    const path = entry.path === undefined ? parent : joinPath(parent, entry.path);
    if (entry.path !== undefined) found.push(path);
    if (entry.children) registered(entry.children, path, found);
  }
  return found;
}

/** Whether a concrete pathname is served by a registered route, params included. */
function servedBy(pathname: string, routePaths: string[]): boolean {
  return routePaths.some((route) => {
    const pattern = new RegExp(
      `^${route
        .split("/")
        .map((segment) =>
          segment.startsWith(":") ? "[^/]+" : segment.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"),
        )
        .join("/")}/?$`,
    );
    return pattern.test(pathname);
  });
}

const entities = Object.keys(RECORD_LINKS) as RecordEntityType[];

/**
 * A link to a record is only as good as the route it names. The assistant's
 * table links pointed at `/shipments`, and worker links carried a parameter
 * the worker table never reads; both landed somewhere other than the record.
 */
describe("record links", () => {
  const routePaths = registered(routes);

  it.each(entities)("opens %s on a registered page", (entity) => {
    const url = new URL(recordPath(entity, "rec_1"), "http://localhost");

    expect(servedBy(url.pathname, routePaths)).toBe(true);
  });

  it.each(entities)("finds the %s it opened again from the address", (entity) => {
    const url = new URL(recordPath(entity, "rec_1"), "http://localhost");

    expect(recordAtLocation(url.pathname, url.search)).toEqual({
      entityType: entity,
      entityId: "rec_1",
    });
  });

  it("opens a list record in the table's edit panel", () => {
    const url = new URL(recordPath("worker", "wrk_1"), "http://localhost");

    expect(url.pathname).toBe("/hr/workers");
    expect(url.searchParams.get("panelType")).toBe("edit");
    expect(url.searchParams.get("panelEntityId")).toBe("wrk_1");
  });

  it("carries extra parameters such as the tab to open", () => {
    const url = new URL(recordPath("worker", "wrk_1", { tab: "hos" }), "http://localhost");

    expect(url.searchParams.get("tab")).toBe("hos");
    expect(url.searchParams.get("panelEntityId")).toBe("wrk_1");
  });

  it("encodes an id in a path segment and in a parameter", () => {
    expect(recordPath("report", "a/b")).toBe("/reports/explore/a%2Fb");
    expect(recordPath("invoice", "a&b=c")).toBe("/billing/invoices?item=a%26b%3Dc");
  });

  it("reads a page with no record open as the record's page", () => {
    expect(recordAtLocation("/billing/invoices", "")).toEqual({
      entityType: "invoice",
      entityId: "",
    });
  });

  it("tells apart records that open on the same page by the view they name", () => {
    expect(recordAtLocation("/admin/agent-control", "?tab=audit&audit=exports")).toEqual({
      entityType: "ai_audit_export",
      entityId: "",
    });
    expect(
      recordAtLocation(
        "/admin/agent-control",
        "?tab=audit&audit=trail&panelType=edit&panelEntityId=aiae_1",
      ),
    ).toEqual({ entityType: "ai_audit_event", entityId: "aiae_1" });
    expect(recordAtLocation("/admin/agent-control", "?tab=activity&activity=runs")).toEqual({
      entityType: "agent_run",
      entityId: "",
    });
  });

  it("reads a shared page naming no view as its first record", () => {
    expect(recordAtLocation("/admin/agent-control", "")).toEqual({
      entityType: "agent_run",
      entityId: "",
    });
  });

  it("does not claim a page records do not open on", () => {
    expect(recordAtLocation("/reports/ifta", "?item=x")).toBeNull();
    expect(recordAtLocation("/hr/workers/extra", "")).toBeNull();
  });

  it("knows its own entity types", () => {
    expect(isRecordEntityType("shipment")).toBe(true);
    expect(isRecordEntityType("toString")).toBe(false);
  });
});
