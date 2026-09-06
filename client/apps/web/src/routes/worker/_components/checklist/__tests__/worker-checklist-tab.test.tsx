import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerChecklistTab from "../../worker-checklist-tab";

const {
  fetchWorkerChecklists,
  fetchActiveWorkerChecklistTemplates,
  completeWorkerChecklistItem,
  skipWorkerChecklistItem,
  reopenWorkerChecklistItem,
  startWorkerChecklist,
} = vi.hoisted(() => ({
  fetchWorkerChecklists: vi.fn(),
  fetchActiveWorkerChecklistTemplates: vi.fn(),
  completeWorkerChecklistItem: vi.fn(),
  skipWorkerChecklistItem: vi.fn(),
  reopenWorkerChecklistItem: vi.fn(),
  startWorkerChecklist: vi.fn(),
}));
const permissionState = vi.hoisted(() => ({ create: true, update: true, cancel: true }));

vi.mock("@/lib/graphql/worker-checklist", () => ({
  fetchWorkerChecklists,
  fetchActiveWorkerChecklistTemplates,
  completeWorkerChecklistItem,
  skipWorkerChecklistItem,
  markWorkerChecklistItemNotApplicable: vi.fn(),
  reopenWorkerChecklistItem,
  startWorkerChecklist,
  cancelWorkerChecklist: vi.fn(),
  WORKER_CHECKLISTS_KEY: "worker-checklists",
  WORKER_CHECKLIST_TEMPLATES_KEY: "worker-checklist-templates",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (_resource: string, operation: number) => {
    const map: Record<number, boolean> = {
      [1 << 1]: permissionState.create,
      [1 << 2]: permissionState.update,
      [1 << 15]: permissionState.cancel,
    };
    return { allowed: map[operation] ?? true, isLoading: false };
  },
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

const item = (
  id: string,
  label: string,
  overrides: Partial<Record<string, unknown>> = {},
): Record<string, unknown> => ({
  id,
  checklistId: "wcl_1",
  label,
  description: null,
  kind: "Task",
  required: true,
  owner: "HR",
  dueAt: null,
  overdue: false,
  credentialTypeId: null,
  documentTypeId: null,
  status: "Pending",
  completedById: null,
  completedAt: null,
  autoCompleted: false,
  note: null,
  evidenceDocumentId: null,
  evidenceCredentialId: null,
  sortOrder: 0,
  version: 1,
  completedBy: null,
  evidenceDocument: null,
  evidenceCredential: null,
  credentialType: null,
  ...overrides,
});

const checklist = {
  id: "wcl_1",
  workerId: "wrk_1",
  templateId: "wclt_1",
  name: "Driver onboarding",
  kind: "Onboarding",
  status: "Open",
  startedAt: 1_756_000_000,
  dueAt: 1_756_700_000,
  completedAt: null,
  cancelledAt: null,
  cancelReason: null,
  sourceEventId: null,
  startedById: "usr_1",
  version: 0,
  createdAt: 1,
  updatedAt: 1,
  startedBy: { id: "usr_1", name: "Ada" },
  progress: {
    total: 4,
    settled: 2,
    requiredTotal: 3,
    requiredDone: 1,
    overdue: 1,
    percent: 50,
    complete: false,
  },
  items: [
    item("wcli_cdl", "CDL on file", {
      kind: "Credential",
      owner: "Safety",
      status: "Done",
      autoCompleted: true,
      completedAt: 1_756_100_000,
      evidenceCredential: { id: "wcred_1", number: "D123", expiresAt: 1_900_000_000 },
    }),
    item("wcli_fuel", "Fuel card issued", {
      kind: "Equipment",
      owner: "Fleet",
      dueAt: 1_756_100_000,
      overdue: true,
    }),
    item("wcli_handbook", "Handbook acknowledged", {
      owner: "HR",
      status: "Skipped",
      note: "Signed on paper",
      completedBy: { id: "usr_2", name: "Grace" },
      completedAt: 1_756_200_000,
    }),
    item("wcli_dash", "Invited to Dash", {
      kind: "PortalAccess",
      owner: "Dispatch",
      required: false,
    }),
  ],
};

function renderTab() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <WorkerChecklistTab workerId="wrk_1" />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  permissionState.create = true;
  permissionState.update = true;
  permissionState.cancel = true;
});

describe("WorkerChecklistTab", () => {
  it("shows the progress ring, groups items by owner, and marks evidence and overdue items", async () => {
    fetchWorkerChecklists.mockResolvedValue([checklist]);
    fetchActiveWorkerChecklistTemplates.mockResolvedValue([]);
    renderTab();

    const card = await screen.findByTestId("checklist-wcl_1");
    expect(within(card).getByText("1/3 required")).toBeInTheDocument();
    expect(within(card).getByText("50%")).toBeInTheDocument();
    expect(within(card).getByText("1 overdue")).toBeInTheDocument();

    const groups = within(card).getAllByTestId(/^checklist-owner-/);
    expect(groups.map((group) => group.getAttribute("data-testid"))).toEqual([
      "checklist-owner-Safety",
      "checklist-owner-Fleet",
      "checklist-owner-HR",
      "checklist-owner-Dispatch",
    ]);

    const cdl = within(card).getByTestId("checklist-item-wcli_cdl");
    expect(cdl).toHaveAttribute("data-status", "Done");
    expect(within(cdl).getByText(/Satisfied by credential D123/)).toBeInTheDocument();

    const fuel = within(card).getByTestId("checklist-item-wcli_fuel");
    expect(fuel).toHaveAttribute("data-overdue", "true");
    expect(
      within(fuel).getByRole("button", { name: "Complete Fuel card issued" }),
    ).toBeInTheDocument();

    const handbook = within(card).getByTestId("checklist-item-wcli_handbook");
    expect(within(handbook).getByText("Skipped")).toBeInTheDocument();
    expect(within(handbook).getByText(/Signed on paper/)).toBeInTheDocument();
    expect(within(handbook).getByText(/Grace/)).toBeInTheDocument();

    const dash = within(card).getByTestId("checklist-item-wcli_dash");
    expect(within(dash).queryByRole("button", { name: /Complete/ })).toBeNull();
    expect(within(dash).getByText("Auto")).toBeInTheDocument();
  });

  it("completes an item straight from the row with the version", async () => {
    fetchWorkerChecklists.mockResolvedValue([checklist]);
    fetchActiveWorkerChecklistTemplates.mockResolvedValue([]);
    completeWorkerChecklistItem.mockResolvedValue(checklist);
    renderTab();

    const fuel = await screen.findByTestId("checklist-item-wcli_fuel");
    fireEvent.click(within(fuel).getByRole("button", { name: "Complete Fuel card issued" }));
    await waitFor(() =>
      expect(completeWorkerChecklistItem).toHaveBeenCalledExactlyOnceWith({
        id: "wcli_fuel",
        version: 1,
      }),
    );
    await waitFor(() => expect(fetchWorkerChecklists).toHaveBeenCalledTimes(2));
  });

  it("skips through a note dialog and refuses an empty note", async () => {
    fetchWorkerChecklists.mockResolvedValue([checklist]);
    fetchActiveWorkerChecklistTemplates.mockResolvedValue([]);
    skipWorkerChecklistItem.mockResolvedValue(checklist);
    renderTab();

    const fuel = await screen.findByTestId("checklist-item-wcli_fuel");
    fireEvent.click(within(fuel).getByRole("button", { name: "Skip Fuel card issued" }));
    const dialog = await screen.findByRole("dialog");
    fireEvent.click(within(dialog).getByRole("button", { name: "Skip item" }));
    expect(
      await within(dialog).findByText("Say why this item is being skipped"),
    ).toBeInTheDocument();
    expect(skipWorkerChecklistItem).not.toHaveBeenCalled();

    fireEvent.change(within(dialog).getByLabelText("Note"), {
      target: { value: "Driver brings own card" },
    });
    fireEvent.click(within(dialog).getByRole("button", { name: "Skip item" }));
    await waitFor(() =>
      expect(skipWorkerChecklistItem).toHaveBeenCalledExactlyOnceWith({
        id: "wcli_fuel",
        note: "Driver brings own card",
        version: 1,
      }),
    );
  });

  it("reopens a settled item", async () => {
    fetchWorkerChecklists.mockResolvedValue([checklist]);
    fetchActiveWorkerChecklistTemplates.mockResolvedValue([]);
    reopenWorkerChecklistItem.mockResolvedValue(checklist);
    renderTab();

    const handbook = await screen.findByTestId("checklist-item-wcli_handbook");
    fireEvent.click(within(handbook).getByRole("button", { name: "Reopen Handbook acknowledged" }));
    await waitFor(() =>
      expect(reopenWorkerChecklistItem).toHaveBeenCalledExactlyOnceWith("wcli_handbook", 1),
    );
  });

  it("starts a checklist from a template when there is none open", async () => {
    fetchWorkerChecklists.mockResolvedValue([]);
    fetchActiveWorkerChecklistTemplates.mockResolvedValue([
      { id: "wclt_on", name: "Driver onboarding", kind: "Onboarding", trigger: "Hired", items: [] },
      {
        id: "wclt_off",
        name: "Driver offboarding",
        kind: "Offboarding",
        trigger: "Terminated",
        items: [],
      },
    ]);
    startWorkerChecklist.mockResolvedValue(checklist);
    renderTab();

    expect(await screen.findByText("No checklists yet")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Start Driver offboarding" }));
    await waitFor(() =>
      expect(startWorkerChecklist).toHaveBeenCalledExactlyOnceWith({
        workerId: "wrk_1",
        templateId: "wclt_off",
      }),
    );
  });

  it("hides item actions without permission", async () => {
    permissionState.update = false;
    permissionState.create = false;
    fetchWorkerChecklists.mockResolvedValue([checklist]);
    fetchActiveWorkerChecklistTemplates.mockResolvedValue([]);
    renderTab();

    const fuel = await screen.findByTestId("checklist-item-wcli_fuel");
    expect(within(fuel).queryByRole("button", { name: /Complete/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /^Start / })).toBeNull();
  });
});
