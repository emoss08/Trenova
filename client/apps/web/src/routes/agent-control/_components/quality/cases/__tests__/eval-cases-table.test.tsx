import type { AgentEvalCaseRow } from "@/lib/graphql/agent-eval-cases";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, waitFor } from "@testing-library/react";
import type { Row, RowAction } from "@trenova/shared/types/data-table";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { EvalCasesTable } from "../eval-cases-table";

const permissions = vi.hoisted(() => ({ denied: new Set<string>() }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: string) => ({
    allowed: !permissions.denied.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

const setAgentEvalCaseStatus = vi.hoisted(() => vi.fn());

vi.mock("@/lib/graphql/agent-eval-cases", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-eval-cases")>()),
  setAgentEvalCaseStatus,
}));

/** The data table has its own tests; here it only hands back the row actions it was given. */
const table = vi.hoisted(() => ({ actions: [] as RowAction<AgentEvalCaseRow>[] }));

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: (props: { contextMenuActions?: RowAction<AgentEvalCaseRow>[] }) => {
    table.actions = props.contextMenuActions ?? [];
    return null;
  },
}));

function row(overrides: Partial<AgentEvalCaseRow>): Row<AgentEvalCaseRow> {
  return {
    original: { id: "aec_1", status: "Active", automatic: true, ...overrides },
  } as Row<AgentEvalCaseRow>;
}

function shown(evalCase: Row<AgentEvalCaseRow>): string[] {
  return table.actions
    .filter((action) => !(typeof action.hidden === "function" && action.hidden(evalCase)))
    .map((action) => action.id);
}

function renderTable() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return render(<EvalCasesTable />, { wrapper });
}

afterEach(() => {
  cleanup();
  permissions.denied.clear();
  setAgentEvalCaseStatus.mockReset();
  table.actions = [];
});

describe("EvalCasesTable", () => {
  it("offers Keep only on an active case the system still keeps", () => {
    renderTable();

    expect(shown(row({ status: "Active", automatic: true }))).toContain("keep");
    expect(shown(row({ status: "Active", automatic: false }))).not.toContain("keep");
    expect(shown(row({ status: "Retired", automatic: true }))).not.toContain("keep");
    expect(shown(row({ status: "Candidate", automatic: false }))).not.toContain("keep");
  });

  it("does not offer Keep to someone who cannot change the golden set", () => {
    permissions.denied.add(`${Resource.AgentEvalSuite}:${Operation.Update}`);
    renderTable();

    expect(shown(row({ status: "Active", automatic: true }))).not.toContain("keep");
  });

  // Keeping is the status the case already has, asked for by a person: the
  // server takes that as the case becoming theirs.
  it("keeps a case by asking for the status it already has", async () => {
    setAgentEvalCaseStatus.mockResolvedValue({});
    renderTable();

    const keep = table.actions.find((action) => action.id === "keep");
    keep?.onClick?.(row({ id: "aec_9", status: "Active", automatic: true }));

    await waitFor(() => expect(setAgentEvalCaseStatus).toHaveBeenCalledTimes(1));
    expect(setAgentEvalCaseStatus).toHaveBeenCalledWith("aec_9", "Active");
  });
});
