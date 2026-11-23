import { describe, expect, it } from "vitest";
import type {
  ReportCatalog,
  ReportCatalogEdge,
  ReportCatalogEntity,
  ReportCatalogField,
} from "@/lib/graphql/reports";
import { fromKeelDefinition } from "@/lib/report-definition";
import {
  dashboardParameterOptions,
  hasAllowedValues,
  parameterChoices,
  reportParameterOptions,
} from "@/lib/report-parameter-options";
import type { ReportIR } from "@/types/report";
import type { Definition, Parameter } from "@trenova/shared/generated/keel-analytics";

function field(overrides: Partial<ReportCatalogField> & { key: string }): ReportCatalogField {
  return {
    label: overrides.key,
    description: null,
    type: "string",
    format: null,
    measure: false,
    enumValues: [],
    aggregations: ["count"],
    operators: ["eq", "ne", "in", "not_in"],
    filterable: true,
    groupable: true,
    access: "Available",
    inReferenceView: false,
    protection: null,
    sensitivity: "internal",
    ...overrides,
  };
}

function edge(
  overrides: Partial<ReportCatalogEdge> & { name: string; target: string },
): ReportCatalogEdge {
  return {
    label: overrides.name,
    description: null,
    toMany: false,
    pick: false,
    access: "Available",
    inReferenceView: true,
    needsReadOf: null,
    ...overrides,
  };
}

function entity(overrides: Partial<ReportCatalogEntity> & { key: string }): ReportCatalogEntity {
  return {
    resource: overrides.key,
    label: overrides.key,
    pluralLabel: overrides.key,
    description: null,
    category: "Desk",
    ownScopeSupported: false,
    readable: true,
    exportable: true,
    fields: [],
    edges: [],
    ...overrides,
  };
}

const catalog: ReportCatalog = {
  version: "1",
  rules: null,
  withheldByPlan: [],
  entities: [
    entity({
      key: "daily_usage",
      fields: [
        field({
          key: "feature",
          type: "enum",
          enumValues: [
            { value: "AgentTurn", label: "Agent turn" },
            { value: "BillingQueueBrief", label: "Billing queue brief" },
            { value: "DocumentIntelligenceExtract", label: "Document extraction" },
          ],
        }),
        field({ key: "model" }),
        field({ key: "calls", type: "int", measure: true }),
      ],
      edges: [edge({ name: "user", target: "user" })],
    }),
    entity({
      key: "user",
      fields: [
        field({ key: "name" }),
        field({
          key: "status",
          type: "enum",
          enumValues: [
            { value: "Active", label: "Active" },
            { value: "Suspended", label: "Suspended" },
          ],
        }),
      ],
    }),
  ],
};

function report(definition: Definition): ReportIR {
  const ir = fromKeelDefinition(definition);
  if (!ir) throw new Error("fixture definition did not parse");
  return ir;
}

const features: Parameter = {
  name: "features",
  label: "Features",
  type: "string",
  multi: true,
  allowed: ["BillingQueueBrief", "AgentTurn", "DocumentIntelligenceExtract"],
};

describe("reportParameterOptions", () => {
  it("names a parameter bound through a top-level filter by the field's enum labels", () => {
    const ir = report({
      version: 2,
      entity: "daily_usage",
      columns: [{ id: "person", dimension: { field: "user.name" } }],
      filters: {
        and: [
          { field: "usageDate", op: "last_n_days", param: "windowDays" },
          { field: "feature", op: "in", param: "features" },
        ],
      },
      parameters: [{ name: "windowDays", type: "int", default: 30 }, features],
    });

    expect(reportParameterOptions(ir, catalog)).toEqual({
      features: [
        { value: "BillingQueueBrief", label: "Billing queue brief" },
        { value: "AgentTurn", label: "Agent turn" },
        { value: "DocumentIntelligenceExtract", label: "Document extraction" },
      ],
    });
  });

  it("finds the condition inside nested and/or groups", () => {
    const ir = report({
      version: 2,
      entity: "daily_usage",
      columns: [{ id: "calls", measure: { agg: "sum", field: "calls" } }],
      filters: {
        and: [
          { field: "model", op: "eq", value: "x" },
          {
            or: [
              { field: "model", op: "is_null" },
              { and: [{ field: "feature", op: "in", param: "features" }] },
            ],
          },
        ],
      },
      parameters: [features],
    });

    expect(reportParameterOptions(ir, catalog).features?.map((option) => option.label)).toEqual([
      "Billing queue brief",
      "Agent turn",
      "Document extraction",
    ]);
  });

  it("finds the condition in a measure's own filter", () => {
    const ir = report({
      version: 2,
      entity: "daily_usage",
      columns: [
        {
          id: "calls",
          measure: {
            agg: "sum",
            field: "calls",
            filter: { field: "feature", op: "in", param: "features" },
          },
        },
      ],
      parameters: [features],
    });

    expect(reportParameterOptions(ir, catalog).features?.[0]).toEqual({
      value: "BillingQueueBrief",
      label: "Billing queue brief",
    });
  });

  it("resolves a field path across an edge to the related entity's enum", () => {
    const ir = report({
      version: 2,
      entity: "daily_usage",
      columns: [{ id: "person", dimension: { field: "user.name" } }],
      filters: { field: "user.status", op: "eq", param: "status" },
      parameters: [{ name: "status", type: "string", allowed: ["Suspended", "Active"] }],
    });

    expect(reportParameterOptions(ir, catalog)).toEqual({
      status: [
        { value: "Suspended", label: "Suspended" },
        { value: "Active", label: "Active" },
      ],
    });
  });

  it("writes the codes in words for a parameter bound to no enum field", () => {
    const ir = report({
      version: 2,
      entity: "daily_usage",
      columns: [{ id: "person", dimension: { field: "user.name" } }],
      filters: { field: "model", op: "in", param: "models" },
      parameters: [{ name: "models", type: "string", multi: true, allowed: ["GPT5Mini", "POD"] }],
    });

    expect(reportParameterOptions(ir, catalog)).toEqual({
      models: [
        { value: "GPT5Mini", label: "GPT5 Mini" },
        { value: "POD", label: "POD" },
      ],
    });
  });

  it("writes a value the catalog does not list in words, beside the labelled ones", () => {
    const ir = report({
      version: 2,
      entity: "daily_usage",
      columns: [{ id: "person", dimension: { field: "user.name" } }],
      filters: { field: "feature", op: "in", param: "features" },
      parameters: [{ ...features, allowed: ["AgentTurn", "RateConfirmationDraft"] }],
    });

    expect(reportParameterOptions(ir, catalog).features).toEqual([
      { value: "AgentTurn", label: "Agent turn" },
      { value: "RateConfirmationDraft", label: "Rate Confirmation Draft" },
    ]);
  });

  it("writes every code in words while the catalog has not arrived", () => {
    const ir = report({
      version: 2,
      entity: "daily_usage",
      columns: [{ id: "person", dimension: { field: "user.name" } }],
      filters: { field: "feature", op: "in", param: "features" },
      parameters: [features],
    });

    expect(reportParameterOptions(ir, undefined).features?.map((option) => option.label)).toEqual([
      "Billing Queue Brief",
      "Agent Turn",
      "Document Intelligence Extract",
    ]);
  });

  it("leaves out parameters without allowed values", () => {
    const ir = report({
      version: 2,
      entity: "daily_usage",
      columns: [{ id: "person", dimension: { field: "user.name" } }],
      filters: { field: "usageDate", op: "last_n_days", param: "windowDays" },
      parameters: [{ name: "windowDays", type: "int" }],
    });

    expect(reportParameterOptions(ir, catalog)).toEqual({});
    expect(hasAllowedValues(ir.parameters)).toBe(false);
  });
});

describe("parameterChoices", () => {
  it("keeps the parameter's own order and words for every code without labels", () => {
    expect(
      parameterChoices({
        name: "kind",
        type: "string",
        required: false,
        allowedValues: ["AmountMismatch", "MissingBOL"],
      }),
    ).toEqual([
      { value: "AmountMismatch", label: "Amount Mismatch" },
      { value: "MissingBOL", label: "Missing BOL" },
    ]);
  });
});

describe("dashboardParameterOptions", () => {
  const tileReport = report({
    version: 2,
    entity: "daily_usage",
    columns: [{ id: "person", dimension: { field: "user.name" } }],
    filters: { field: "feature", op: "in", param: "features" },
    parameters: [features],
  });

  const dashboardParam = {
    name: "dashFeatures",
    type: "string",
    required: false,
    multi: true,
    allowedValues: ["AgentTurn", "BillingQueueBrief"],
  };

  it("names a dashboard parameter through a tile that binds its report parameter to it", () => {
    expect(
      dashboardParameterOptions(
        [dashboardParam],
        [{ ir: tileReport, bindings: { features: "dashFeatures" } }],
        catalog,
      ),
    ).toEqual({
      dashFeatures: [
        { value: "AgentTurn", label: "Agent turn" },
        { value: "BillingQueueBrief", label: "Billing queue brief" },
      ],
    });
  });

  it("links a report parameter to the dashboard parameter of the same name", () => {
    expect(
      dashboardParameterOptions(
        [{ ...dashboardParam, name: "features" }],
        [{ ir: null }, { ir: tileReport }],
        catalog,
      ).features?.[0],
    ).toEqual({ value: "AgentTurn", label: "Agent turn" });
  });

  it("writes the codes in words when no tile links to the parameter", () => {
    expect(dashboardParameterOptions([dashboardParam], [{ ir: tileReport }], catalog)).toEqual({
      dashFeatures: [
        { value: "AgentTurn", label: "Agent Turn" },
        { value: "BillingQueueBrief", label: "Billing Queue Brief" },
      ],
    });
  });
});
