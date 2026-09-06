import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import WorkerTrainingTab from "../../worker-training-tab";

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

vi.mock("sonner", () => ({ toast }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("../assign-training-dialog", () => ({
  AssignTrainingDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="assign-dialog" /> : null,
}));
vi.mock("../complete-training-dialog", () => ({
  CompleteTrainingDialog: ({ open, courseId }: { open: boolean; courseId?: string | null }) =>
    open ? <div data-testid="complete-dialog">{courseId}</div> : null,
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

function renderTab() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <WorkerTrainingTab workerId="wrk_1" />
    </QueryClientProvider>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("WorkerTrainingTab", () => {
  it("shows the matrix with progress, required slots first, and the right action per slot", async () => {
    fetchWorkerTrainingSummary.mockResolvedValue(summary);
    fetchWorkerTrainingRecords.mockResolvedValue([
      hazmatRecord,
      orientationRecord,
      winterRecord,
      oldFailure,
    ]);
    renderTab();

    expect(await screen.findByText("1/3")).toBeInTheDocument();
    expect(screen.getByText("Not qualified")).toBeInTheDocument();

    const hazmatCard = within(screen.getByTestId("training-slot-trnc_hazmat"));
    expect(hazmatCard.getByText("Overdue by 4 days")).toBeInTheDocument();
    expect(hazmatCard.getByText(/Acknowledged by the driver/)).toBeInTheDocument();
    expect(hazmatCard.getByRole("button", { name: "Record result" })).toBeInTheDocument();
    expect(hazmatCard.getByRole("button", { name: "Waive" })).toBeInTheDocument();

    const hosCard = within(screen.getByTestId("training-slot-trnc_hos"));
    expect(hosCard.getByText("Not assigned")).toBeInTheDocument();
    expect(hosCard.getByRole("button", { name: "Assign" })).toBeInTheDocument();

    const orientationCard = within(screen.getByTestId("training-slot-trnc_orient"));
    expect(orientationCard.getByText("Does not expire")).toBeInTheDocument();
    expect(
      orientationCard.queryByRole("button", { name: "Record result" }),
    ).not.toBeInTheDocument();

    const slots = screen.getAllByTestId(/^training-slot-/);
    expect(slots[0]).toHaveAttribute("data-testid", "training-slot-trnc_orient");
    expect(slots[slots.length - 1]).toHaveAttribute("data-testid", "training-slot-trnc_winter");
  });

  it("assigns every missing required course in one click", async () => {
    fetchWorkerTrainingSummary.mockResolvedValue(summary);
    fetchWorkerTrainingRecords.mockResolvedValue([]);
    assignRequiredWorkerTraining.mockResolvedValue([record("wtrn_new", "trnc_hos")]);
    renderTab();

    fireEvent.click(await screen.findByRole("button", { name: "Assign required" }));
    await waitFor(() => expect(assignRequiredWorkerTraining).toHaveBeenCalledWith("wrk_1"));
    expect(toast.success).toHaveBeenCalledWith(
      "1 course assigned",
      expect.objectContaining({ description: expect.stringContaining("Hours of Service") }),
    );
  });

  it("opens the result and waiver dialogs for an open course and keeps closed records in history", async () => {
    fetchWorkerTrainingSummary.mockResolvedValue(summary);
    fetchWorkerTrainingRecords.mockResolvedValue([
      hazmatRecord,
      orientationRecord,
      winterRecord,
      oldFailure,
    ]);
    renderTab();

    const hazmatCard = within(await screen.findByTestId("training-slot-trnc_hazmat"));
    fireEvent.click(hazmatCard.getByRole("button", { name: "Record result" }));
    expect(screen.getByTestId("complete-dialog")).toHaveTextContent("trnc_hazmat");

    fireEvent.click(hazmatCard.getByRole("button", { name: "Waive" }));
    expect(screen.getByTestId("waive-dialog")).toHaveTextContent("wtrn_hazmat");

    fireEvent.click(screen.getByRole("button", { name: /History \(2\)/ }));
    expect(screen.getByText("55.00%")).toBeInTheDocument();
  });
});
