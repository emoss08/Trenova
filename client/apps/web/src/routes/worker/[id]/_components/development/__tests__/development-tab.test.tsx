import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import {
  CLOSED_WORKER_OPEN_STATE,
  type WorkerOpenState,
  type WorkerRecord,
} from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useController, type Control } from "react-hook-form";
import { afterEach, beforeAll, describe, expect, it, vi } from "vitest";
import {
  AssignTrainingPane,
  CoursePane,
  EditReviewPane,
  RecordTrainingResultPane,
} from "../development-panes";
import DevelopmentTab from "../development-tab";

const mocks = vi.hoisted(() => ({
  fetchWorkerTrainingSummary: vi.fn(),
  fetchWorkerTrainingRecords: vi.fn(),
  fetchWorkerTrainingRecord: vi.fn(),
  completeWorkerTraining: vi.fn(),
  attachWorkerTrainingDocument: vi.fn(),
  fetchWorkerPerformanceReviews: vi.fn(),
  updatePerformanceReview: vi.fn(),
}));

vi.mock("@/lib/graphql/worker-training", () => ({
  fetchWorkerTrainingSummary: mocks.fetchWorkerTrainingSummary,
  fetchWorkerTrainingRecords: mocks.fetchWorkerTrainingRecords,
  fetchWorkerTrainingRecord: mocks.fetchWorkerTrainingRecord,
  completeWorkerTraining: mocks.completeWorkerTraining,
  attachWorkerTrainingDocument: mocks.attachWorkerTrainingDocument,
  assignWorkerTraining: vi.fn(),
  WORKER_TRAINING_KEY: "worker-training",
  WORKER_TRAINING_SUMMARY_KEY: "worker-training-summary",
}));

vi.mock("@/lib/graphql/performance-review", () => ({
  fetchWorkerPerformanceReviews: mocks.fetchWorkerPerformanceReviews,
  updatePerformanceReview: mocks.updatePerformanceReview,
  createPerformanceReview: vi.fn(),
  WORKER_REVIEWS_KEY: "worker-reviews",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

vi.mock("@/hooks/use-select-option", () => ({
  useSelectOption: () => ({ option: null, isLoading: false }),
  useFirstSelectOption: () => ({ option: null, isLoading: false }),
}));

vi.mock("../training/worker-training-tab", () => ({ default: () => null }));
vi.mock("../reviews/worker-reviews-tab", () => ({ default: () => null }));

vi.mock("@/components/autocomplete-fields", () => ({
  TrainingCourseAutocompleteField: ({
    control,
    name,
    label,
    disabled,
  }: {
    control: Control;
    name: string;
    label: string;
    disabled?: boolean;
  }) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        disabled={disabled}
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      />
    );
  },
  PerformanceReviewTemplateAutocompleteField: () => null,
}));

vi.mock("@/components/fields/date-field/date-field", () => ({
  AutoCompleteDateField: ({ label }: { label: string }) => <span>{label}</span>,
}));

vi.mock("@/components/documents/file-drop-field", () => ({
  FileDropField: ({ label }: { label: string }) => <p>{label}</p>,
}));

const hazmatCourse = {
  id: "trnc_hazmat",
  name: "Hazmat awareness refresher",
  category: "Compliance",
  delivery: "Online",
  durationMinutes: 45,
  passingScore: "80.00",
  isRequired: true,
};

const hazmatRecord = {
  id: "wtrn_hazmat",
  workerId: "wrk_1",
  courseId: "trnc_hazmat",
  course: hazmatCourse,
  status: "Assigned",
  assignedAt: 1_799_000_000,
  dueAt: 1_800_000_000,
  completedAt: null,
  expiresAt: null,
  score: null,
  document: null,
  assignedBy: null,
  recordedBy: null,
  notes: null,
  waivedReason: null,
  health: "DueSoon",
  daysUntilDue: 12,
  daysUntilExpiry: null,
  version: 3,
};

const summary = {
  workerId: "wrk_1",
  compliant: true,
  requiredCount: 1,
  currentCount: 0,
  dueCount: 1,
  overdueCount: 0,
  expiringCount: 0,
  expiredCount: 0,
  missingCount: 0,
  items: [
    {
      course: hazmatCourse,
      record: hazmatRecord,
      health: "DueSoon",
      daysUntilDue: 12,
      daysUntilExpiry: null,
      required: true,
    },
  ],
};

const draft = {
  id: "prv_2026",
  workerId: "wrk_1",
  template: { id: "rvt_1", code: "ANNUAL", name: "Driver annual", cadenceMonths: 12, items: [] },
  reviewer: { id: "usr_1", name: "Dana Whitfield" },
  status: "Draft",
  title: "2026 annual review",
  periodStart: 1_767_225_600,
  periodEnd: 1_798_675_200,
  ratings: [
    { key: "safety", label: "Safety and compliance", weight: 1, score: 4, comment: null },
    { key: "ontime", label: "On-time performance", weight: 1, score: null, comment: null },
  ],
  overallScore: "4.00",
  summary: null,
  strengths: null,
  improvements: null,
  goals: [],
  acknowledgedAt: null,
  workerComment: null,
  closedAt: null,
  nextReviewAt: null,
  version: 2,
};

const worker = { firstName: "Maria", wholeName: "Maria Alvarez" } as unknown as WorkerRecord;

function renderTab(open: Partial<WorkerOpenState>, view: "training" | "reviews" = "training") {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const onOpen = vi.fn();
  const view_ = render(
    <QueryClientProvider client={client}>
      <WorkPaneProvider>
        <WorkPaneHost>
          <DevelopmentTab
            workerId="wrk_1"
            worker={worker}
            view={view}
            open={{ ...CLOSED_WORKER_OPEN_STATE, ...open }}
            onOpen={onOpen}
            shown
          />
        </WorkPaneHost>
      </WorkPaneProvider>
    </QueryClientProvider>,
  );
  return { onOpen, client, ...view_ };
}

/** One page of the records connection, as the server answers `WorkerTrainingRecords`. */
function page(nodes: unknown[]) {
  return {
    edges: nodes.map((node, index) => ({ cursor: `c${index}`, node })),
    pageInfo: { hasNextPage: false, endCursor: null },
  };
}

// The panes load on demand; read their code once up front so the first test's wait is for the
// pane, not for the test runner transforming the module.
beforeAll(async () => {
  await Promise.all(
    [AssignTrainingPane, RecordTrainingResultPane, CoursePane, EditReviewPane].map((pane) =>
      pane.preload(),
    ),
  );
});

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("DevelopmentTab panes from the address", () => {
  it("opens the result pane for the assignment the address names, with its course locked", async () => {
    mocks.fetchWorkerTrainingSummary.mockResolvedValue(summary);
    mocks.fetchWorkerTrainingRecords.mockResolvedValue(page([hazmatRecord]));
    renderTab({ form: "record-training-result", training: "wtrn_hazmat" });

    const pane = await screen.findByRole("dialog", {
      name: "Record result: Hazmat awareness refresher",
    });
    expect(await within(pane).findByLabelText("Course")).toBeDisabled();
    expect(within(pane).getByText("Passes at 80%.")).toBeInTheDocument();
  });

  it("says a score below the mark fails before saving, and records the fail on the assignment", async () => {
    mocks.fetchWorkerTrainingSummary.mockResolvedValue(summary);
    mocks.fetchWorkerTrainingRecords.mockResolvedValue(page([hazmatRecord]));
    mocks.completeWorkerTraining.mockResolvedValue({ ...hazmatRecord, status: "Failed" });
    const { onOpen } = renderTab({ form: "record-training-result", training: "wtrn_hazmat" });
    const user = userEvent.setup();

    const pane = await screen.findByRole("dialog", {
      name: "Record result: Hazmat awareness refresher",
    });
    await user.type(await within(pane).findByLabelText(/Score/), "70");
    expect(within(pane).getByText("Passes at 80%. This would fail.")).toBeInTheDocument();
    await user.click(within(pane).getByRole("button", { name: "Record fail" }));

    await waitFor(() => expect(mocks.completeWorkerTraining).toHaveBeenCalledTimes(1));
    expect(mocks.completeWorkerTraining.mock.calls[0][0]).toMatchObject({
      id: "wtrn_hazmat",
      score: "70",
      version: 3,
      workerId: undefined,
      courseId: undefined,
    });
    expect(mocks.attachWorkerTrainingDocument).not.toHaveBeenCalled();
    await waitFor(() =>
      expect(onOpen).toHaveBeenCalledWith({ tab: "development", view: "training" }),
    );
  });

  it("presets the assign pane with the course the address names, so a reload keeps it", async () => {
    mocks.fetchWorkerTrainingSummary.mockResolvedValue(summary);
    renderTab({ form: "assign-training", course: "trnc_hazmat" });

    const pane = await screen.findByRole("dialog", { name: "Assign course" });
    expect(await within(pane).findByLabelText("Course")).toHaveValue("trnc_hazmat");
  });

  it("reads a course in the pane when the address names a record without a form", async () => {
    mocks.fetchWorkerTrainingSummary.mockResolvedValue(summary);
    mocks.fetchWorkerTrainingRecords.mockResolvedValue(page([hazmatRecord]));
    renderTab({ training: "wtrn_hazmat" });

    const pane = await screen.findByRole("dialog", { name: "Hazmat awareness refresher" });
    expect(await within(pane).findByText("None attached")).toBeInTheDocument();
  });

  it("opens the draft the address names in the edit pane, with the overall score live", async () => {
    mocks.fetchWorkerTrainingSummary.mockResolvedValue(summary);
    mocks.fetchWorkerPerformanceReviews.mockResolvedValue([draft]);
    renderTab({ form: "edit-review", review: "prv_2026" }, "reviews");
    const user = userEvent.setup();

    const pane = await screen.findByRole("dialog", { name: "Edit 2026 annual review" });
    const overall = await within(pane).findByTestId("review-overall");
    expect(overall).toHaveTextContent("4.0");
    expect(overall).toHaveTextContent("1 of 2 rated");

    await user.click(
      within(pane).getByRole("radio", { name: "On-time performance: 1, Needs work" }),
    );
    expect(overall).toHaveTextContent("2.5");
    expect(overall).toHaveTextContent("2 of 2 rated");
  });

  it("does not open the edit pane for a review that is no longer a draft", async () => {
    mocks.fetchWorkerTrainingSummary.mockResolvedValue(summary);
    mocks.fetchWorkerPerformanceReviews.mockResolvedValue([{ ...draft, status: "Submitted" }]);
    renderTab({ form: "edit-review", review: "prv_2026" }, "reviews");

    await waitFor(() => expect(mocks.fetchWorkerPerformanceReviews).toHaveBeenCalled());
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("reads an earlier record the address names from the records list when no course stands on it", async () => {
    const earlier = {
      ...hazmatRecord,
      id: "wtrn_earlier",
      status: "Completed",
      completedAt: 1_700_000_000,
      score: "91.00",
      health: "Current",
    };
    mocks.fetchWorkerTrainingSummary.mockResolvedValue(summary);
    mocks.fetchWorkerTrainingRecords.mockResolvedValue(page([hazmatRecord, earlier]));
    renderTab({ training: "wtrn_earlier" });

    const pane = await screen.findByRole("dialog", { name: "Hazmat awareness refresher" });
    expect(await within(pane).findByText("91%")).toBeInTheDocument();
  });

  it("does not read the training records for a record the summary already holds", async () => {
    mocks.fetchWorkerTrainingSummary.mockResolvedValue(summary);
    renderTab({ training: "wtrn_hazmat" });

    await screen.findByRole("dialog", { name: "Hazmat awareness refresher" });
    expect(mocks.fetchWorkerTrainingRecords).not.toHaveBeenCalled();
  });

  it("reads nothing for training while the reviews are showing and no pane is open", async () => {
    mocks.fetchWorkerPerformanceReviews.mockResolvedValue([]);
    renderTab({}, "reviews");

    await waitFor(() => expect(mocks.fetchWorkerTrainingSummary).not.toHaveBeenCalled());
    expect(mocks.fetchWorkerTrainingRecords).not.toHaveBeenCalled();
  });

  it("keeps what is typed in the edit pane when the draft is read again", async () => {
    mocks.fetchWorkerPerformanceReviews.mockResolvedValue([draft]);
    const { client } = renderTab({ form: "edit-review", review: "prv_2026" }, "reviews");
    const user = userEvent.setup();

    const pane = await screen.findByRole("dialog", { name: "Edit 2026 annual review" });
    const title = await within(pane).findByLabelText(/Title/);
    await user.clear(title);
    await user.type(title, "Annual review, revised");

    act(() => {
      client.setQueryData(["worker-reviews", "wrk_1"], [{ ...draft, version: 3 }]);
    });
    expect(within(pane).getByLabelText(/Title/)).toHaveValue("Annual review, revised");
  });

  it("sets a rating from the keyboard and clears it with Delete", async () => {
    mocks.fetchWorkerPerformanceReviews.mockResolvedValue([draft]);
    renderTab({ form: "edit-review", review: "prv_2026" }, "reviews");
    const user = userEvent.setup();

    const pane = await screen.findByRole("dialog", { name: "Edit 2026 annual review" });
    const safety = await within(pane).findByRole("radio", {
      name: "Safety and compliance: 4, Strong",
    });
    expect(safety).toHaveAttribute("aria-checked", "true");
    safety.focus();
    await user.keyboard("{ArrowRight}");
    expect(within(pane).getByRole("radio", { name: /^Safety and compliance: 5,/ })).toHaveAttribute(
      "aria-checked",
      "true",
    );
    await user.keyboard("{Delete}");
    expect(within(pane).getByTestId("review-overall")).toHaveTextContent("Not rated yet");
  });
});
