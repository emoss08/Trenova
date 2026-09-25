import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import type { FieldFilter } from "@trenova/shared/types/data-table";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

type StubTableProps = {
  name: string;
  resource?: string;
  scopeFilters?: FieldFilter[];
  enableExport?: boolean;
  enableReadOnlyPanel?: boolean;
  TablePanel?: unknown;
};

const table = vi.hoisted(() => ({ props: [] as StubTableProps[] }));
const permissions = vi.hoisted(() => ({ allowed: new Set<string>() }));

// The data table has its own tests; here it only records what the trail
// hands it.
vi.mock("@/components/data-table/data-table", () => ({
  DataTable: (props: StubTableProps) => {
    table.props.push(props);
    return <section aria-label={`${props.name} table`} />;
  },
}));

vi.mock("../chain-status", () => ({ ChainStatusStrip: () => null }));
vi.mock("../audit-scope-bar", () => ({
  AuditScopeBar: ({ actions }: { actions?: React.ReactNode }) => <div>{actions}</div>,
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: number) => ({
    allowed: permissions.allowed.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

const { default: AuditTrailView } = await import("../audit-trail-view");
const { Operation, Resource } = await import("@trenova/shared/types/permission");

function renderView(searchParams: Record<string, string> = {}) {
  const client = new QueryClient();
  return render(
    <QueryClientProvider client={client}>
      <NuqsTestingAdapter searchParams={searchParams}>
        <AuditTrailView onOpenExports={() => {}} />
      </NuqsTestingAdapter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  table.props = [];
  permissions.allowed = new Set([`${Resource.AIAuditTrail}:${Operation.Read}`]);
});

afterEach(() => {
  cleanup();
});

describe("AuditTrailView", () => {
  // The trail's export is a signed server-side file; the browser's CSV of
  // the page would be an unsigned, unaudited copy, so the table never offers it.
  it("shows the standard table with the scope's filters and no browser export", () => {
    renderView();

    const props = table.props.at(-1);
    expect(props?.resource).toBe(Resource.AIAuditTrail);
    expect(props?.enableExport).toBe(false);
    expect(props?.enableReadOnlyPanel).toBe(true);
    expect(props?.TablePanel).toBeDefined();
    expect(props?.scopeFilters).toEqual([
      { field: "occurredAt", operator: "lastndays", value: 7 },
      { field: "purpose", operator: "eq", value: "Live" },
    ]);
  });

  it("narrows the table to the scope a shared address names", () => {
    renderView({
      trailScope: JSON.stringify({
        range: { kind: "between", from: 1_760_000_000, to: 1_760_604_799 },
        agent: { id: "agdef_1", name: "Billing desk" },
        personId: "usr_1",
        includeEvaluations: true,
      }),
    });

    expect(table.props.at(-1)?.scopeFilters).toEqual([
      { field: "occurredAt", operator: "gte", value: 1_760_000_000 },
      { field: "occurredAt", operator: "lte", value: 1_760_604_799 },
      { field: "agentDefinitionId", operator: "eq", value: "agdef_1" },
      { field: "personId", operator: "eq", value: "usr_1" },
    ]);
  });

  it("falls back to the default scope when the address cannot mean one", () => {
    renderView({ trailScope: JSON.stringify({ range: { kind: "last", days: 12 } }) });

    expect(table.props.at(-1)?.scopeFilters).toEqual([
      { field: "occurredAt", operator: "lastndays", value: 7 },
      { field: "purpose", operator: "eq", value: "Live" },
    ]);
  });

  it("offers the export only to people who may export the trail", () => {
    renderView();
    expect(screen.queryByRole("button", { name: /Export trail/ })).not.toBeInTheDocument();
    cleanup();

    permissions.allowed.add(`${Resource.AIAuditTrail}:${Operation.Export}`);
    renderView();
    expect(screen.getByRole("button", { name: /Export trail/ })).toBeInTheDocument();
  });
});
