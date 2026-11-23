import {
  RECORD_LINKS,
  isRecordEntityType,
  recordAtLocation,
  recordPath,
  type RecordEntityType,
} from "@/config/record-links";
import { servesPage } from "@/test/route-tree";
import { describe, expect, it } from "vitest";

const entities = Object.keys(RECORD_LINKS) as RecordEntityType[];

/**
 * A link to a record is only as good as the route it names. The assistant's
 * table links pointed at `/shipments`, and worker links carried a parameter
 * the worker table never reads; both landed somewhere other than the record.
 */
describe("record links", () => {
  it.each(entities)("opens %s on a registered page", (entity) => {
    const url = new URL(recordPath(entity, "rec_1"), "http://localhost");

    expect(servesPage(url.pathname)).toBe(true);
  });

  it.each(entities)("finds the %s it opened again from the address", (entity) => {
    const url = new URL(recordPath(entity, "rec_1"), "http://localhost");

    expect(recordAtLocation(url.pathname, url.search)).toEqual({
      entityType: entity,
      entityId: "rec_1",
    });
  });

  it("opens a worker on the worker's own page", () => {
    expect(recordPath("worker", "wrk_1")).toBe("/hr/workers/wrk_1");
  });

  it("opens a list record in the table's edit panel", () => {
    const url = new URL(recordPath("customer", "cus_1"), "http://localhost");

    expect(url.pathname).toBe("/billing/configuration-files/customers");
    expect(url.searchParams.get("panelType")).toBe("edit");
    expect(url.searchParams.get("panelEntityId")).toBe("cus_1");
  });

  it("carries extra parameters such as the tab to open", () => {
    expect(recordPath("worker", "wrk_1", { tab: "time", view: "hours-of-service" })).toBe(
      "/hr/workers/wrk_1?tab=time&view=hours-of-service",
    );
  });

  it("never reads a create page as a record named new", () => {
    expect(recordAtLocation("/hr/workers/new", "")).toBeNull();
    expect(recordAtLocation("/billing/configuration-files/factoring-companies/new", "")).toBeNull();
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

  it("opens an analysis in Activity's analyses, told apart from an agent run by its id", () => {
    expect(recordPath("analysis_run", "aanr_1")).toBe(
      "/admin/agent-control?tab=activity&activity=analyses&analysisRun=aanr_1",
    );
    expect(
      recordAtLocation(
        "/admin/agent-control",
        "?tab=activity&activity=analyses&analysisRun=aanr_1",
      ),
    ).toEqual({ entityType: "analysis_run", entityId: "aanr_1" });
  });

  it("reads a shared page naming no view as its first record", () => {
    expect(recordAtLocation("/admin/agent-control", "")).toEqual({
      entityType: "agent_run",
      entityId: "",
    });
  });

  it("does not claim a page records do not open on", () => {
    expect(recordAtLocation("/reports/ifta", "?item=x")).toBeNull();
    expect(recordAtLocation("/hr/workers/wrk_1/extra", "")).toBeNull();
    expect(recordAtLocation("/hr/workers", "")).toBeNull();
  });

  it("knows its own entity types", () => {
    expect(isRecordEntityType("shipment")).toBe(true);
    expect(isRecordEntityType("toString")).toBe(false);
  });
});
