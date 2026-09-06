import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { WorkerPTO } from "@trenova/shared/types/worker";
import { useController, type Control } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PTOFormDialog, normalizePTODates } from "../pto-form-dialog";

const { createWorkerPTO, updateWorkerPTO } = vi.hoisted(() => ({
  createWorkerPTO: vi.fn(),
  updateWorkerPTO: vi.fn(),
}));

vi.mock("@/lib/graphql/worker-mutations", () => ({
  createWorkerPTO,
  updateWorkerPTO,
}));

vi.mock("@/lib/graphql/pto-policy", () => ({
  fetchWorkerPtoAvailability: vi.fn().mockResolvedValue({
    tracked: false,
    enforced: false,
    allowed: true,
    days: "0",
    balanceDays: "0",
    pendingDays: "0",
    projectedAccrualDays: "0",
    projectedAvailableDays: "0",
    floorDays: "0",
    message: null,
  }),
}));

vi.mock("@trenova/shared/hooks/use-debounce", () => ({
  useDebounce: <T,>(value: T): T => value,
}));

vi.mock("@/lib/queries", () => ({
  queries: {
    worker: {
      listUpcomingPTO: { _def: ["worker", "list-upcoming-pto"] },
      ptoChartData: { _def: ["worker", "pto-chart-data"] },
    },
  },
}));

vi.mock("@/components/autocomplete-fields", () => ({
  WorkerAutocompleteField: ({
    control,
    name,
    disabled,
  }: {
    control: Control;
    name: string;
    disabled?: boolean;
  }) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label="Worker"
        disabled={disabled}
        value={(field.value as string) ?? ""}
        onChange={(event) => field.onChange(event.target.value)}
      />
    );
  },
}));

vi.mock("@/components/fields/date-field/date-field", () => ({
  AutoCompleteDateField: ({
    control,
    name,
    label,
  }: {
    control: Control;
    name: string;
    label: string;
  }) => {
    const { field } = useController({ control, name });
    return (
      <input
        aria-label={label}
        type="number"
        value={(field.value as number) || ""}
        onChange={(event) => field.onChange(Number(event.target.value))}
      />
    );
  },
}));

vi.mock("sonner", () => ({
  toast: { success: vi.fn(), error: vi.fn() },
}));

const DAY = 86_400;
const START = 1_767_225_600;
const END = START + DAY * 2;

function renderDialog(props: Partial<React.ComponentProps<typeof PTOFormDialog>> = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const onOpenChange = vi.fn();
  render(
    <QueryClientProvider client={queryClient}>
      <PTOFormDialog open onOpenChange={onOpenChange} {...props} />
    </QueryClientProvider>,
  );
  return { onOpenChange };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("normalizePTODates", () => {
  it("expands the chosen days to a full-day range so a single day is a valid request", () => {
    const normalized = normalizePTODates({
      workerId: "wrk_1",
      type: "Vacation",
      startDate: START,
      endDate: START,
      reason: "One day",
    });
    expect(normalized.endDate).toBeGreaterThan(normalized.startDate);
    expect(normalized.endDate - normalized.startDate).toBeLessThan(DAY);
    expect(normalized.startDate).toBeLessThanOrEqual(START);
  });
});

describe("PTOFormDialog", () => {
  it("requests PTO for the chosen worker with normalised dates", async () => {
    createWorkerPTO.mockResolvedValue({ id: "wrkpto_new" });
    const user = userEvent.setup();
    const { onOpenChange } = renderDialog();

    await user.type(screen.getByLabelText("Worker"), "wrk_1");
    await user.clear(screen.getByLabelText("First day"));
    await user.type(screen.getByLabelText("First day"), String(START));
    await user.clear(screen.getByLabelText("Last day"));
    await user.type(screen.getByLabelText("Last day"), String(END));
    await user.type(screen.getByLabelText("Reason"), "Family trip");

    expect(screen.getByTestId("pto-days-preview")).toHaveTextContent("3 days");

    await user.click(screen.getByRole("button", { name: /request pto/i }));

    await waitFor(() => expect(createWorkerPTO).toHaveBeenCalledTimes(1));
    const input = createWorkerPTO.mock.calls[0][0];
    expect(input.workerId).toBe("wrk_1");
    expect(input.type).toBe("Vacation");
    expect(input.reason).toBe("Family trip");
    expect(input.startDate).toBeLessThanOrEqual(START);
    expect(input.endDate).toBeGreaterThanOrEqual(END);
    expect(updateWorkerPTO).not.toHaveBeenCalled();
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
  });

  it("does not reserve space for the days preview until both dates are set", async () => {
    const user = userEvent.setup();
    renderDialog();

    const controls = () => document.querySelectorAll("[data-slot=form-control]").length;
    expect(screen.queryByTestId("pto-days-preview")).not.toBeInTheDocument();
    const before = controls();

    await user.clear(screen.getByLabelText("First day"));
    await user.type(screen.getByLabelText("First day"), String(START));
    expect(screen.queryByTestId("pto-days-preview")).not.toBeInTheDocument();
    expect(controls()).toBe(before);

    await user.clear(screen.getByLabelText("Last day"));
    await user.type(screen.getByLabelText("Last day"), String(END));
    expect(await screen.findByTestId("pto-days-preview")).toBeInTheDocument();
    expect(controls()).toBe(before + 1);
  });

  it("prefills the dates from a dragged calendar range", () => {
    renderDialog({ defaultRange: { start: START, end: END } });

    expect(screen.getByLabelText("First day")).toHaveValue(START);
    expect(screen.getByLabelText("Last day")).toHaveValue(END);
    expect(screen.getByTestId("pto-days-preview")).toHaveTextContent("3 days");
  });

  it("refuses to submit without a worker or reason", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByRole("button", { name: /request pto/i }));

    expect(await screen.findByText(/reason is required/i)).toBeInTheDocument();
    expect(createWorkerPTO).not.toHaveBeenCalled();
  });

  it("edits a requested PTO through the update mutation with its id and version", async () => {
    updateWorkerPTO.mockResolvedValue({ id: "wrkpto_1" });
    const user = userEvent.setup();
    const pto = {
      id: "wrkpto_1",
      workerId: "wrk_1",
      status: "Requested",
      type: "Sick",
      startDate: START,
      endDate: END,
      reason: "Flu",
      version: 4,
    } as WorkerPTO;
    renderDialog({ pto });

    expect(screen.getByLabelText("Worker")).toBeDisabled();
    await user.clear(screen.getByLabelText("Reason"));
    await user.type(screen.getByLabelText("Reason"), "Flu, extended");
    await user.click(screen.getByRole("button", { name: /save changes/i }));

    await waitFor(() => expect(updateWorkerPTO).toHaveBeenCalledTimes(1));
    const input = updateWorkerPTO.mock.calls[0][0];
    expect(input.id).toBe("wrkpto_1");
    expect(input.version).toBe(4);
    expect(input.reason).toBe("Flu, extended");
    expect(input).not.toHaveProperty("workerId");
    expect(createWorkerPTO).not.toHaveBeenCalled();
  });

  it("shows an approved request read-only instead of an editable form", () => {
    const pto = {
      id: "wrkpto_2",
      workerId: "wrk_1",
      status: "Approved",
      type: "Vacation",
      startDate: START,
      endDate: END,
      reason: "Beach",
      version: 2,
      approver: { id: "usr_1", name: "Dana Dispatcher" },
      worker: { firstName: "Ada", lastName: "Lovelace" },
    } as WorkerPTO;
    renderDialog({ pto });

    expect(screen.getByText(/only requested time off can be edited/i)).toBeInTheDocument();
    expect(screen.getByText("Dana Dispatcher")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /save changes/i })).not.toBeInTheDocument();
    expect(screen.queryByLabelText("Reason")).not.toBeInTheDocument();
  });
});
