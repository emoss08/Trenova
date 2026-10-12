import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const permissions = vi.hoisted(() => ({ allowed: new Set<string>() }));
const dialog = vi.hoisted(() => ({ props: [] as Record<string, unknown>[] }));

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: (props: { name: string; toolbar?: { trailing?: ReactNode } }) => (
    <section aria-label={`${props.name} table`}>{props.toolbar?.trailing}</section>
  ),
}));

vi.mock("../export-trail-dialog", () => ({
  ExportTrailDialog: (props: Record<string, unknown>) => {
    dialog.props.push(props);
    return props.open ? <p>Export dialog</p> : null;
  },
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: number) => ({
    allowed: permissions.allowed.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

const { default: AuditExportsView } = await import("../audit-exports-view");
const { Operation, Resource } = await import("@trenova/shared/types/permission");
const { DEFAULT_AUDIT_SCOPE, UNFILTERED_AUDIT_TABLE } = await import("../audit-model");

function renderView() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <NuqsTestingAdapter>
        <AuditExportsView />
      </NuqsTestingAdapter>
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  dialog.props = [];
  permissions.allowed = new Set([`${Resource.AIAuditTrail}:${Operation.Read}`]);
});

afterEach(cleanup);

describe("AuditExportsView", () => {
  it("offers an export of the whole default scope to people who may export", async () => {
    permissions.allowed.add(`${Resource.AIAuditTrail}:${Operation.Export}`);
    renderView();

    await userEvent.click(screen.getByRole("button", { name: "Export trail" }));

    expect(screen.getByText("Export dialog")).toBeInTheDocument();
    expect(dialog.props.at(-1)).toMatchObject({
      scope: DEFAULT_AUDIT_SCOPE,
      table: UNFILTERED_AUDIT_TABLE,
    });
  });

  it("offers no export without the right to export", () => {
    renderView();

    expect(screen.queryByRole("button", { name: "Export trail" })).not.toBeInTheDocument();
    expect(dialog.props).toHaveLength(0);
  });
});
