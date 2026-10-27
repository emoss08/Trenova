import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { cleanup, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ReportSchedulesDialog } from "../report-schedules-dialog";

const mocks = vi.hoisted(() => ({
  schedules: vi.fn(),
  mutation: () => ({ mutate: vi.fn(), isPending: false }),
}));

vi.mock("@/hooks/use-reports", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/hooks/use-reports")>()),
  useReportSchedules: mocks.schedules,
  useCreateReportSchedule: mocks.mutation,
  useUpdateReportSchedule: mocks.mutation,
  useDeleteReportSchedule: mocks.mutation,
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}));

function schedule(id: string, runAsId: string, cronExpression: string) {
  return {
    id,
    definitionId: "rdef_1",
    cronExpression,
    timezone: "UTC",
    formats: ["csv"],
    emailRecipients: [],
    emailAttach: false,
    emailInline: false,
    notifyUserIds: [],
    alert: null,
    alertFiring: false,
    enabled: true,
    runAsId,
    lastRunId: null,
    nextRunAt: null,
    consecutiveFailures: 0,
    version: 1,
    createdAt: 1_700_000_000,
    updatedAt: 1_700_000_000,
  };
}

const definition = {
  id: "rdef_1",
  name: "Aging Detail",
  definition: { entity: "shipment", columns: [] },
} as never;

function renderDialog() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <ReportSchedulesDialog open onOpenChange={vi.fn()} definition={definition} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

function scheduleRows(): HTMLElement[] {
  return screen.getAllByRole("switch").map((toggle) => {
    const row = toggle.closest(".group");
    if (!(row instanceof HTMLElement)) throw new Error("a schedule switch sits outside its row");
    return row;
  });
}

beforeEach(() => {
  useAuthStore.setState({ user: { id: "usr_me" } } as never);
  mocks.schedules.mockReturnValue({
    data: [
      schedule("rsch_mine", "usr_me", "0 8 * * 1"),
      schedule("rsch_theirs", "usr_teammate", "0 9 1 * *"),
    ],
    isLoading: false,
  });
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("the report schedules dialog", () => {
  it("offers changes only on the schedules the person set up", () => {
    renderDialog();

    const rows = scheduleRows();
    expect(rows).toHaveLength(2);
    const [mine, theirs] = rows.map((row) => within(row));

    expect(mine.getByRole("button", { name: "Edit schedule" })).toBeInTheDocument();
    expect(mine.getByRole("button", { name: "Delete schedule" })).toBeInTheDocument();
    expect(mine.getByRole("switch")).not.toHaveAttribute("aria-disabled", "true");
    expect(mine.queryByText("Set up by a teammate")).toBeNull();

    expect(theirs.queryByRole("button", { name: "Edit schedule" })).toBeNull();
    expect(theirs.queryByRole("button", { name: "Delete schedule" })).toBeNull();
    expect(theirs.getByRole("switch")).toHaveAttribute("aria-disabled", "true");
    expect(theirs.getByText("Set up by a teammate")).toBeInTheDocument();
  });
});
