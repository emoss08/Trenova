import type { ExtractionEvalRefreezeProgress } from "@/lib/graphql/extraction-eval";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { Operation, Resource } from "@trenova/shared/types/permission";
import type { ReactNode } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const permissions = vi.hoisted(() => ({ denied: new Set<string>() }));
const refreeze = vi.fn();
const fetchProgress = vi.fn<() => Promise<ExtractionEvalRefreezeProgress | null>>();

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string, operation: string) => ({
    allowed: !permissions.denied.has(`${resource}:${operation}`),
    isLoading: false,
  }),
}));

vi.mock("@/components/data-table/data-table", () => ({
  DataTable: ({ toolbar }: { toolbar?: { end?: ReactNode } }) => (
    <div data-testid="cases-table">{toolbar?.end}</div>
  ),
}));

vi.mock("@/lib/graphql/extraction-eval", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/graphql/extraction-eval")>();
  return {
    ...actual,
    refreezeExtractionEvalCases: (input: unknown) => refreeze(input),
    fetchExtractionEvalRefreeze: () => fetchProgress(),
  };
});

const { CasesTable } = await import("../cases-table");

beforeEach(() => {
  permissions.denied.clear();
  refreeze.mockReset();
  fetchProgress.mockReset();
  fetchProgress.mockResolvedValue(null);
  refreeze.mockResolvedValue({
    queued: 3,
    workflowId: "extraction-eval-refreeze:org:bu",
    alreadyRunning: false,
  });
});

afterEach(() => cleanup());

function renderTable() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={client}>
      <CasesTable />
    </QueryClientProvider>,
  );
}

describe("CasesTable re-conversion", () => {
  it("re-converts every active case with Docling and shows how far it got", async () => {
    renderTable();
    fetchProgress.mockResolvedValue({
      workflowId: "extraction-eval-refreeze:org:bu",
      status: "Running",
      total: 3,
      frozen: 1,
      skipped: 0,
      failed: 0,
    });

    await userEvent.click(
      await screen.findByRole("button", { name: "Re-convert cases with Docling" }),
    );

    await waitFor(() => expect(refreeze).toHaveBeenCalledWith({}));
    expect(
      await screen.findByText("Re-converting cases with Docling: 1 of 3 cases"),
    ).toBeInTheDocument();
    expect(screen.getByText("1 re-converted, 0 skipped, 0 failed")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Re-convert cases with Docling" })).toBeDisabled();
  });

  it("reports the result counts of the last re-conversion", async () => {
    fetchProgress.mockResolvedValue({
      workflowId: "extraction-eval-refreeze:org:bu",
      status: "Completed",
      total: 5,
      frozen: 3,
      skipped: 1,
      failed: 1,
    });
    renderTable();

    expect(
      await screen.findByText("The last re-conversion with Docling finished"),
    ).toBeInTheDocument();
    expect(
      screen.getByText("5 cases asked for: 3 re-converted, 1 skipped, 1 failed"),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Re-convert cases with Docling" })).toBeEnabled();
  });

  it("offers no re-conversion without permission to update the evaluation suite", async () => {
    permissions.denied.add(`${Resource.AgentEvalSuite}:${Operation.Update}`);
    renderTable();

    await screen.findByTestId("cases-table");
    expect(
      screen.queryByRole("button", { name: "Re-convert cases with Docling" }),
    ).not.toBeInTheDocument();
  });
});
