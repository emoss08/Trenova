import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerTrainingTab from "../worker-training-tab";

const {
  fetchWorkerTrainingSummary,
  fetchWorkerTrainingRecords,
  assignRequiredWorkerTraining,
  cancelWorkerTraining,
  toast,
} = vi.hoisted(() => ({
  fetchWorkerTrainingSummary: vi.fn(),
  fetchWorkerTrainingRecords: vi.fn(),
  assignRequiredWorkerTraining: vi.fn(),
  cancelWorkerTraining: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}));

vi.mock("@/lib/graphql/worker-training", () => ({
  fetchWorkerTrainingSummary,
  fetchWorkerTrainingRecords,
  assignRequiredWorkerTraining,
  cancelWorkerTraining,
  fetchActiveTrainingCourses: vi.fn().mockResolvedValue([]),
  WORKER_TRAINING_KEY: "worker-training",
  WORKER_TRAINING_SUMMARY_KEY: "worker-training-summary",
  TRAINING_COURSES_KEY: "training-courses",
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({ toast }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("../waive-training-dialog", () => ({
  WaiveTrainingDialog: ({ open, record }: { open: boolean; record?: { id: string } | null }) =>
    open ? <div data-testid="waive-dialog">{record?.id}</div> : null,
}));

function course(id: string, name: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    businessUnitId: "bu_1",
    organizationId: "org_1",
    code: id.toUpperCase(),
    name,
    description: null,
    category: "Safety",
    status: "Active",
    delivery: "Classroom",
    contentUrl: null,
    durationMinutes: 60,
    passingScore: null,
    validityMonths: null,
    renewalWindowDays: 30,
    isRequired: true,
    requiredForDriverTypes: [],
    dueDaysAfterAssignment: 30,
    requiresAcknowledgement: true,
    sortOrder: 0,
    openRecordCount: 0,
    version: 1,
    createdAt: 1,
    updatedAt: 1,
    ...overrides,
  };
}

function record(id: string, courseId: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    businessUnitId: "bu_1",
    organizationId: "org_1",
    workerId: "wrk_1",
    courseId,
    course: null,
    status: "Assigned",
    assignedAt: 1_799_000_000,
    dueAt: null,
    startedAt: null,
    completedAt: null,
    expiresAt: null,
    score: null,
    passed: null,
    acknowledgedAt: null,
    documentId: null,
    document: null,
    assignedById: null,
    assignedBy: null,
    recordedById: null,
    recordedBy: null,
    notes: null,
    waivedReason: null,
    health: "Scheduled",
    daysUntilDue: null,
    daysUntilExpiry: null,
    version: 1,
    createdAt: 1,
    updatedAt: 1,
    ...overrides,
  };
}

const hazmat = course("trnc_hazmat", "Hazmat Awareness", { passingScore: "80.00" });
const orientation = course("trnc_orient", "New Driver Orientation");
const hos = course("trnc_hos", "Hours of Service Refresher", { delivery: "Document" });
const winter = course("trnc_winter", "Winter Driving", { isRequired: false, delivery: "Online" });

const hazmatRecord = record("wtrn_hazmat", "trnc_hazmat", {
  status: "InProgress",
  dueAt: 1_799_000_000,
  health: "Overdue",
  daysUntilDue: -4,
  acknowledgedAt: 1_798_500_000,
});
const orientationRecord = record("wtrn_orient", "trnc_orient", {
  status: "Completed",
  completedAt: 1_700_000_000,
  health: "Current",
});
const winterRecord = record("wtrn_winter", "trnc_winter", {
  status: "Assigned",
  dueAt: 1_801_000_000,
  health: "Scheduled",
  daysUntilDue: 20,
});
const oldFailure = record("wtrn_old", "trnc_hazmat", {
  status: "Failed",
  completedAt: 1_690_000_000,
  health: "Failed",
  score: "55.00",
});

const summary = {
  workerId: "wrk_1",
  compliant: false,
  requiredCount: 3,
  currentCount: 1,
  dueCount: 1,
  overdueCount: 1,
  expiringCount: 0,
  expiredCount: 0,
  missingCount: 1,
  items: [
    {
      course: orientation,
      record: orientationRecord,
      health: "Current",
      daysUntilDue: null,
      daysUntilExpiry: null,
      required: true,
    },
    {
      course: hazmat,
      record: hazmatRecord,
      health: "Overdue",
      daysUntilDue: -4,
      daysUntilExpiry: null,
      required: true,
    },
    {
      course: hos,
      record: null,
      health: "Missing",
      daysUntilDue: null,
      daysUntilExpiry: null,
      required: true,
    },
    {
      course: winter,
      record: winterRecord,
      health: "Scheduled",
      daysUntilDue: 20,
      daysUntilExpiry: null,
      required: false,
    },
  ],
};

const handlers = { onAssign: vi.fn(), onRecord: vi.fn(), onRead: vi.fn() };

function renderTab() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <WorkerTrainingTab workerId="wrk_1" {...handlers} />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

function loadAll() {
  fetchWorkerTrainingSummary.mockResolvedValue(summary);
  fetchWorkerTrainingRecords.mockResolvedValue([
    hazmatRecord,
    orientationRecord,
    winterRecord,
    oldFailure,
  ]);
}

describe("WorkerTrainingTab", () => {
  it("shows the standing, then one table grouped Required and Other with each row's action", async () => {
    loadAll();
    renderTab();

    expect(await screen.findByText("1 of 3")).toBeInTheDocument();
    expect(screen.getByText("Not qualified")).toBeInTheDocument();
    expect(screen.getByText("1 not assigned")).toBeInTheDocument();

    const user = userEvent.setup();
    const hazmatRow = within(screen.getByTestId("training-slot-trnc_hazmat"));
    expect(hazmatRow.getByText("Overdue by 4 days")).toBeInTheDocument();
    await user.click(hazmatRow.getByRole("button", { name: /^Actions for/ }));
    expect(await screen.findByRole("menuitem", { name: "Record result" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Waive" })).toBeInTheDocument();
    expect(screen.getByRole("menuitem", { name: "Cancel assignment" })).toBeInTheDocument();
    await user.keyboard("{Escape}");

    const hosRow = within(screen.getByTestId("training-slot-trnc_hos"));
    expect(hosRow.getAllByText("Not assigned").length).toBeGreaterThan(0);
    await user.click(hosRow.getByRole("button", { name: /^Actions for/ }));
    expect(await screen.findByRole("menuitem", { name: "Assign" })).toBeInTheDocument();
    await user.keyboard("{Escape}");

    const rows = screen.getAllByTestId(/^training-slot-/);
    expect(rows.map((row) => row.getAttribute("data-testid"))).toEqual([
      "training-slot-trnc_orient",
      "training-slot-trnc_hazmat",
      "training-slot-trnc_hos",
      "training-slot-trnc_winter",
    ]);
    expect(screen.getByText("Required")).toBeInTheDocument();
    expect(screen.getByText("Other")).toBeInTheDocument();
  });

  it("opens what each row needs next: the result for an open assignment, the course read otherwise", async () => {
    loadAll();
    renderTab();
    const user = userEvent.setup();

    await user.click(await screen.findByTestId("training-slot-trnc_hazmat"));
    expect(handlers.onRecord).toHaveBeenCalledWith(expect.objectContaining({ course: hazmat }));

    await user.click(screen.getByTestId("training-slot-trnc_orient"));
    expect(handlers.onRead).toHaveBeenCalledWith(orientationRecord);

    await user.click(screen.getByTestId("training-slot-trnc_hos"));
    expect(handlers.onAssign).toHaveBeenCalledWith("trnc_hos");
  });

  it("offers Assign required only while there are gaps, and assigns them in one request", async () => {
    loadAll();
    assignRequiredWorkerTraining.mockResolvedValue([record("wtrn_new", "trnc_hos")]);
    renderTab();

    const button = await screen.findByRole("button", { name: "Assign required" });
    fireEvent.click(button);
    fireEvent.click(button);
    await waitFor(() => expect(assignRequiredWorkerTraining).toHaveBeenCalledTimes(1));
    expect(assignRequiredWorkerTraining).toHaveBeenCalledWith("wrk_1");
  });

  it("hides Assign required when every required course is open or current", async () => {
    fetchWorkerTrainingSummary.mockResolvedValue({
      ...summary,
      missingCount: 0,
      items: summary.items.filter((item) => item.course.id !== "trnc_hos"),
    });
    fetchWorkerTrainingRecords.mockResolvedValue([]);
    renderTab();

    expect(await screen.findByRole("button", { name: "Assign course" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Assign required" })).not.toBeInTheDocument();
  });

  it("asks before cancelling an assignment, then cancels it with its version", async () => {
    loadAll();
    cancelWorkerTraining.mockResolvedValue(hazmatRecord);
    renderTab();
    const user = userEvent.setup();

    const hazmatRow = within(await screen.findByTestId("training-slot-trnc_hazmat"));
    await user.click(hazmatRow.getByRole("button", { name: /^Actions for/ }));
    await user.click(await screen.findByRole("menuitem", { name: "Cancel assignment" }));
    expect(cancelWorkerTraining).not.toHaveBeenCalled();
    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "Cancel assignment" }));
    await waitFor(() =>
      expect(cancelWorkerTraining).toHaveBeenCalledWith({ id: "wtrn_hazmat", version: 1 }),
    );
  });

  it("keeps the waiver a dialog and folds earlier records under the table", async () => {
    loadAll();
    renderTab();
    const user = userEvent.setup();

    const hazmatRow = within(await screen.findByTestId("training-slot-trnc_hazmat"));
    await user.click(hazmatRow.getByRole("button", { name: /^Actions for/ }));
    await user.click(await screen.findByRole("menuitem", { name: "Waive" }));
    expect(screen.getByTestId("waive-dialog")).toHaveTextContent("wtrn_hazmat");

    await user.click(screen.getByRole("button", { name: /History/ }));
    expect(screen.getByText(/55%/)).toBeInTheDocument();
  });

  it("says the read failed instead of drawing an empty table", async () => {
    fetchWorkerTrainingSummary.mockRejectedValue(new Error("boom"));
    fetchWorkerTrainingRecords.mockResolvedValue([]);
    renderTab();
    expect(await screen.findByText("Couldn't load the training")).toBeInTheDocument();
    expect(screen.queryByText("No courses on file")).not.toBeInTheDocument();
  });
});
