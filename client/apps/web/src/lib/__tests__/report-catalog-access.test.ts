import { describe, expect, it } from "vitest";
import {
  listedFields,
  rootEntities,
  unavailableRefs,
  withoutRefs,
} from "@/lib/report-catalog-access";
import { buildCatalogIndex } from "@/routes/reports/builder/_components/builder-state";
import {
  catalog,
  customer,
} from "@/routes/reports/builder/_components/__tests__/catalog-access-fixture";
import type { ReportIR } from "@/types/report";

const index = buildCatalogIndex(catalog);

describe("report catalog access", () => {
  it("starts a report only from a kind the caller reads", () => {
    expect(rootEntities(catalog).map((entity) => entity.key)).toEqual(["shipment"]);
  });

  it("lists only the reference view of a kind reached through a join", () => {
    expect(listedFields(customer).map((field) => field.key)).toEqual(["name", "code"]);
  });

  it("names every field a saved report uses that the caller may not, once each", () => {
    const ir: ReportIR = {
      entity: "shipment",
      columns: [
        { id: "c1", kind: "dimension", ref: { field: "proNumber" } },
        { id: "c2", kind: "dimension", ref: { path: ["customer"], field: "name" } },
        { id: "c3", kind: "dimension", ref: { path: ["customer"], field: "dotNumber" } },
        { id: "c4", kind: "dimension", ref: { field: "marginFloorPercent" } },
        { id: "c5", kind: "dimension", ref: { path: ["driverProfile"], field: "cdlClass" } },
      ],
      filters: {
        op: "and",
        filters: [{ ref: { field: "status" }, operator: "eq", value: "New" }],
        groups: [
          {
            op: "or",
            filters: [{ ref: { path: ["customer"], field: "dotNumber" }, operator: "eq" }],
          },
        ],
      },
      pivot: { ref: { field: "gone" }, values: [], measureIds: [] },
    };

    expect(unavailableRefs(index, ir)).toEqual([
      {
        ref: { path: ["customer"], field: "dotNumber" },
        label: "DOT Number",
        entityLabel: "Customer",
        access: "NeedsKindRead",
        kindLabel: "Customers",
        group: undefined,
      },
      {
        ref: { field: "marginFloorPercent" },
        label: "Margin Floor",
        entityLabel: "Shipment",
        access: "NeedsFieldGroup",
        kindLabel: undefined,
        group: "Margins",
      },
      {
        ref: { path: ["driverProfile"], field: "cdlClass" },
        label: "cdlClass",
        entityLabel: "Shipment",
        access: "NeedsKindRead",
        kindLabel: "Workers",
      },
      {
        ref: { field: "status" },
        label: "Status",
        entityLabel: "Shipment",
        access: "WithheldByCeiling",
        kindLabel: undefined,
        group: undefined,
      },
      { ref: { field: "gone" }, label: "gone", entityLabel: "Shipment", access: "Missing" },
    ]);
  });

  it("takes out the unavailable columns, filters and pivot and keeps the rest", () => {
    const ir: ReportIR = {
      entity: "shipment",
      columns: [
        { id: "c1", kind: "dimension", ref: { field: "proNumber" } },
        { id: "c2", kind: "dimension", ref: { path: ["customer"], field: "dotNumber" } },
      ],
      filters: {
        op: "and",
        filters: [
          { ref: { field: "status" }, operator: "eq", value: "New" },
          { ref: { field: "proNumber" }, operator: "eq", value: "P1" },
        ],
      },
      having: { op: "and", filters: [{ ref: { field: "status" }, operator: "eq" }] },
      pivot: { ref: { field: "status" }, values: [], measureIds: [] },
    };

    const next = withoutRefs(ir, unavailableRefs(index, ir));

    expect(next.columns.map((column) => column.id)).toEqual(["c1"]);
    expect(next.filters).toEqual({
      op: "and",
      filters: [{ ref: { field: "proNumber" }, operator: "eq", value: "P1" }],
      groups: [],
    });
    expect(next.having).toBeNull();
    expect(next.pivot).toBeNull();
    expect(unavailableRefs(index, next)).toEqual([]);
  });
});
