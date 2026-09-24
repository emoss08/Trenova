import { describe, expect, it } from "vitest";
import { agentSubjectPath } from "../agent-subjects";

describe("agentSubjectPath", () => {
  it("opens a saved report an exception is about in the report explorer", () => {
    expect(agentSubjectPath("Report", "rd_01J")).toBe("/reports/explore/rd_01J");
  });

  it("opens a dashboard an exception is about on the dashboard", () => {
    expect(agentSubjectPath("Dashboard", "rdb_01J")).toBe("/reports/dashboards/rdb_01J");
  });

  it("opens a shipment on its own page", () => {
    expect(agentSubjectPath("Shipment", "shp_01J")).toBe(
      "/shipment-management/shipments?expanded=shp_01J&panelType=edit&panelEntityId=shp_01J",
    );
  });

  it("has no page for a subject that is not a record page", () => {
    expect(agentSubjectPath("Insight", "inst_01J")).toBeNull();
    expect(agentSubjectPath("Organization", "org_01J")).toBeNull();
  });

  it("has no page without an id or for an unknown kind", () => {
    expect(agentSubjectPath("Report", "")).toBeNull();
    expect(agentSubjectPath("Spreadsheet", "rd_01J")).toBeNull();
  });
});
