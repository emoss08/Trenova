import { describe, expect, it } from "vitest";
import type { Definition } from "@trenova/shared/generated/keel-analytics";
import {
  dashboardFilterFromKeel,
  dashboardFilterToKeel,
  fromKeelDefinition,
  toKeelDefinition,
} from "@/lib/report-definition";
import type { ReportIR } from "@/types/report";

const revenueByCustomer: Definition = {
  version: 2,
  entity: "shipment",
  columns: [
    { id: "customer", label: "Customer", dimension: { field: "customer.name" } },
    { id: "month", dimension: { field: "createdAt", bucket: "month" } },
    { id: "revenue", measure: { field: "totalChargeAmount", agg: "sum" } },
    {
      id: "late",
      measure: {
        agg: "count",
        filter: { and: [{ field: "status", op: "not_in", value: ["Completed"] }] },
      },
    },
    { id: "per_load", calc: "[revenue] / [late]", format: "money" },
    { id: "share", calc: "round([revenue] * 100, 1)" },
    { id: "running", window: { of: "revenue", fn: "running_total", over: "month" } },
  ],
  filters: {
    and: [
      { field: "status", op: "starts_with", value: "Comp" },
      { calc: "[revenue] > 0" },
      { or: [{ field: "createdAt", op: "last_n_days", param: "windowDays" }] },
    ],
  },
  having: { and: [{ column: "revenue", op: "gt", value: 1000 }] },
  groupings: { total: true, subtotals: ["customer"] },
  sort: [{ column: "revenue", desc: true }, { column: "customer" }],
  limit: { param: "rows" },
  pivot: { field: "status", values: ["New", "Completed"], columns: ["revenue"], other: true },
  parameters: [
    { name: "windowDays", type: "int", required: true, allowed: [7, 30] },
    { name: "rows", type: "int", default: 50 },
    { name: "customer", type: "ref", entity: "customer" },
  ],
  charts: [
    {
      id: "trend",
      type: "line",
      x: "month",
      series: ["revenue"],
      goal: { value: 5, label: "Goal" },
    },
  ],
};

describe("fromKeelDefinition", () => {
  it("reads every column kind the builder edits into its editing model", () => {
    const ir = fromKeelDefinition(revenueByCustomer);

    expect(ir?.entity).toBe("shipment");
    expect(ir?.columns[0]).toMatchObject({
      id: "customer",
      kind: "dimension",
      label: "Customer",
      ref: { path: ["customer"], field: "name" },
    });
    expect(ir?.columns[1]).toMatchObject({ kind: "dimension", bucket: "month" });
    expect(ir?.columns[2]).toMatchObject({
      kind: "measure",
      agg: "sum",
      ref: { field: "totalChargeAmount" },
    });
    expect(ir?.columns[3].filter?.filters?.[0]).toEqual({
      ref: { field: "status" },
      operator: "notin",
      value: ["Completed"],
    });
    expect(ir?.columns[4]).toMatchObject({
      kind: "computed",
      computed: { op: "divide", leftId: "revenue", rightId: "late", format: "money" },
    });
  });

  it("keeps what the builder cannot edit so a save loses nothing", () => {
    const ir = fromKeelDefinition(revenueByCustomer);

    expect(ir?.columns[5]).toMatchObject({ kind: "computed", calc: "round([revenue] * 100, 1)" });
    expect(ir?.columns[5].computed).toBeUndefined();
    expect(ir?.columns[6]).toMatchObject({ kind: "computed", window: { fn: "running_total" } });
    expect(ir?.extensions).toEqual({ subtotals: ["customer"], limitParam: "rows" });
    expect(toKeelDefinition(ir as ReportIR)).toEqual(revenueByCustomer);
  });

  it("reads a having condition on a measure column as the field and aggregation the panel edits", () => {
    const ir = fromKeelDefinition(revenueByCustomer);

    expect(ir?.having?.filters?.[0]).toEqual({
      ref: { field: "totalChargeAmount" },
      agg: "sum",
      operator: "gt",
      value: 1000,
    });
  });

  it("refuses something that is not a definition", () => {
    expect(fromKeelDefinition(null)).toBeNull();
    expect(fromKeelDefinition({ entity: "shipment" })).toBeNull();
    expect(fromKeelDefinition("shipment")).toBeNull();
  });
});

describe("toKeelDefinition", () => {
  it("adds a hidden measure for a having condition no column computes", () => {
    const ir: ReportIR = {
      entity: "shipment",
      columns: [{ id: "customer", kind: "dimension", ref: { path: ["customer"], field: "name" } }],
      having: {
        op: "and",
        filters: [{ ref: { field: "id" }, agg: "count", operator: "gte", value: 3 }],
      },
    };

    const definition = toKeelDefinition(ir);

    expect(definition.columns).toEqual([
      { id: "customer", dimension: { field: "customer.name" } },
      { id: "having_count_id", hidden: true, measure: { agg: "count", field: "id" } },
    ]);
    expect(definition.having).toEqual({
      and: [{ column: "having_count_id", op: "gte", value: 3 }],
    });
    expect(fromKeelDefinition(definition)?.columns).toHaveLength(1);
  });

  it("writes an operation on two operands as formula text", () => {
    const ir: ReportIR = {
      entity: "invoice",
      columns: [
        { id: "total", kind: "measure", agg: "sum", ref: { field: "totalAmount" } },
        {
          id: "half",
          kind: "computed",
          computed: { op: "multiply", leftId: "total", rightValue: -0.5, format: "none" },
        },
      ],
    };

    expect(toKeelDefinition(ir).columns[1]).toEqual({ id: "half", calc: "[total] * -0.5" });
  });

  it("types allowed parameter values and names the entity a record parameter picks", () => {
    const ir: ReportIR = {
      entity: "shipment",
      columns: [{ id: "n", kind: "measure", agg: "count" }],
      parameters: [
        { name: "days", type: "int", required: false, allowedValues: ["7", "x"] },
        { name: "late", type: "bool", required: false, allowedValues: ["true"] },
        { name: "customer", type: "ref", required: false, refEntity: "customer" },
        { name: "label", type: "string", required: false, refEntity: "customer" },
      ],
    };

    expect(toKeelDefinition(ir).parameters).toEqual([
      { name: "days", type: "int", allowed: [7, "x"] },
      { name: "late", type: "bool", allowed: [true] },
      { name: "customer", type: "ref", entity: "customer" },
      { name: "label", type: "string" },
    ]);
  });
});

describe("dashboard filters", () => {
  it("converts between the builder's reference and the keel path and operator", () => {
    const keel = {
      id: "f1",
      entity: "shipment",
      field: "customer.name",
      op: "is_not_null" as const,
    };

    const builder = dashboardFilterFromKeel(keel);

    expect(builder).toEqual({
      id: "f1",
      label: undefined,
      entity: "shipment",
      ref: { path: ["customer"], field: "name" },
      operator: "isnotnull",
      default: undefined,
    });
    expect(dashboardFilterToKeel(builder)).toEqual(keel);
  });
});
