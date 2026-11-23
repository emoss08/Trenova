import { WorkPaneHost, WorkPaneProvider } from "@/components/work-pane";
import type { WorkerRecord } from "@/routes/worker/use-worker-state";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { getTodayDate } from "@trenova/shared/lib/date";
import { useController, type Control } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AmendEventPane } from "../amend-event-pane";
import { RecordEventPane } from "../record-event-pane";

const mocks = vi.hoisted(() => ({
  recordWorkerEmploymentEvent: vi.fn(),
  amendWorkerEmploymentEvent: vi.fn(),
  fetchWorkerEmploymentEventOptions: vi.fn(),
  previewWorkerEmploymentEvent: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn(), info: vi.fn() },
}));

vi.mock("@/lib/graphql/worker-employment", () => ({
  recordWorkerEmploymentEvent: mocks.recordWorkerEmploymentEvent,
  amendWorkerEmploymentEvent: mocks.amendWorkerEmploymentEvent,
  fetchWorkerEmploymentEventOptions: mocks.fetchWorkerEmploymentEventOptions,
  previewWorkerEmploymentEvent: mocks.previewWorkerEmploymentEvent,
  fetchWorkerEmploymentEvents: vi.fn(),
  WORKER_EMPLOYMENT_EVENTS_KEY: "worker-employment-events",
  WORKER_EMPLOYMENT_OPTIONS_KEY: "worker-employment-event-options",
  WORKER_EMPLOYMENT_PREVIEW_KEY: "worker-employment-event-preview",
}));

vi.mock("@trenova/shared/components/ui/toast", () => ({ toast: mocks.toast }));

function pickerMock(label: string) {
  return ({ control, name }: { control: Control; name: string }) => {
    const { field, fieldState } = useController({ control, name });
    return (
      <>
        <input
          aria-label={label}
          value={(field.value as string) ?? ""}
          onChange={(event) => field.onChange(event.target.value || null)}
        />
        {fieldState.error?.message ? <p>{fieldState.error.message}</p> : null}
      </>
    );
  };
}

vi.mock("@/components/autocomplete-fields", () => ({
  FleetCodeAutocompleteField: pickerMock("Fleet code"),
  UserAutocompleteField: pickerMock("Manager"),
}));

vi.mock("@/components/fields/select-field", () => ({
  SelectField: ({
    control,
    name,
    label,
    options,
  }: {
    control: Control;
    name: string;
    label: string;
    options: { value: string; label: string }[];
  }) => {
    const { field, fieldState } = useController({ control, name });
    return (
      <>
        <select
          aria-label={label}
          value={(field.value as string) ?? ""}
          onChange={(event) => field.onChange(event.target.value || null)}
        >
          <option value="">—</option>
          {options.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
        {fieldState.error?.message ? <p>{fieldState.error.message}</p> : null}
      </>
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

const worker = {
  wholeName: "Maria Alvarez",
  firstName: "Maria",
  driverType: "OTR",
  type: "Employee",
  fleetCode: { id: "fc_1", code: "MW-OTR", color: null },
  manager: { id: "usr_9", name: "Marcus Bell" },
} as unknown as WorkerRecord;

const activeOptions = {
  recordableKinds: [
    "ProbationEnded",
    "Promoted",
    "Transferred",
    "RateChanged",
    "LeaveStarted",
    "Suspended",
    "Terminated",
  ],
  reasonRequiredKinds: ["Terminated", "Suspended", "LeaveStarted", "RateChanged"],
};

const terminationEffects = [
  {
    kind: "PtoPolicyEnded",
    name: "Driver PTO",
    count: null,
    days: null,
    ptoType: null,
    startDate: null,
    endDate: null,
    failed: false,
  },
  {
    kind: "PtoPaidOut",
    name: null,
    count: null,
    days: "9.50",
    ptoType: "Vacation",
    startDate: null,
    endDate: null,
    failed: false,
  },
  {
    kind: "PayAssignmentEnded",
    name: "OTR mileage",
    count: null,
    days: null,
    ptoType: null,
    startDate: null,
    endDate: null,
    failed: false,
  },
  {
    kind: "PortalAccessRevoked",
    name: null,
    count: null,
    days: null,
    ptoType: null,
    startDate: null,
    endDate: null,
    failed: false,
  },
  {
    kind: "ChecklistStarted",
    name: "Offboarding",
    count: null,
    days: null,
    ptoType: null,
    startDate: null,
    endDate: null,
    failed: false,
  },
];

function inputById(name: string): HTMLInputElement | HTMLTextAreaElement {
  const element =
    document.getElementById(`input-${name}`) ??
    document.getElementById(`textarea-${name}`) ??
    document.getElementById(name);
  if (!(element instanceof HTMLInputElement) && !(element instanceof HTMLTextAreaElement)) {
    throw new Error(`${name} not rendered`);
  }
  return element;
}

function wrap(children: React.ReactNode) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return (
    <QueryClientProvider client={queryClient}>
      <WorkPaneProvider>
        <WorkPaneHost>{children}</WorkPaneHost>
      </WorkPaneProvider>
    </QueryClientProvider>
  );
}

function renderPane(presetKind: string | null = null) {
  const onClose = vi.fn();
  render(
    wrap(
      <RecordEventPane
        open
        presetKind={presetKind}
        workerId="wrk_1"
        worker={worker}
        onClose={onClose}
      />,
    ),
  );
  return { onClose };
}

async function pane() {
  return within(await screen.findByRole("dialog", { name: "Record employment event" }));
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
  vi.useRealTimers();
});

describe("RecordEventPane", () => {
  it("offers only the kinds the server says the worker can move through", async () => {
    mocks.fetchWorkerEmploymentEventOptions.mockResolvedValue({
      recordableKinds: ["Rehired"],
      reasonRequiredKinds: [],
    });
    renderPane();

    const select = (await (await pane()).findByLabelText("Event")) as HTMLSelectElement;
    const options = Array.from(select.options).map((option) => option.value);
    expect(options).toEqual(["", "Rehired"]);
    expect(mocks.fetchWorkerEmploymentEventOptions).toHaveBeenCalledWith("wrk_1", {
      signal: expect.any(AbortSignal),
    });
  });

  it("presets the kind from the address when the worker may record it", async () => {
    mocks.fetchWorkerEmploymentEventOptions.mockResolvedValue(activeOptions);
    mocks.previewWorkerEmploymentEvent.mockResolvedValue({
      kind: "Terminated",
      effectiveAt: getTodayDate(),
      fromValues: [],
      toValues: [],
      effects: terminationEffects,
    });
    renderPane("Terminated");

    const view = await pane();
    expect(await view.findByLabelText("Event")).toHaveValue("Terminated");
    expect(view.getByText("This ends employment.")).toBeInTheDocument();
    expect(view.getByRole("button", { name: "End employment" })).toBeInTheDocument();
  });

  it("ignores a preset the worker may not record", async () => {
    mocks.fetchWorkerEmploymentEventOptions.mockResolvedValue({
      recordableKinds: ["Rehired"],
      reasonRequiredKinds: [],
    });
    renderPane("Terminated");

    expect(await (await pane()).findByLabelText("Event")).toHaveValue("");
  });

  it("lists what a termination will do, in words, from the server's preview", async () => {
    mocks.fetchWorkerEmploymentEventOptions.mockResolvedValue(activeOptions);
    mocks.previewWorkerEmploymentEvent.mockResolvedValue({
      kind: "Terminated",
      effectiveAt: getTodayDate(),
      fromValues: [{ key: "status", value: "Active" }],
      toValues: [{ key: "status", value: "Inactive" }],
      effects: terminationEffects,
    });
    renderPane();

    const view = await pane();
    fireEvent.change(await view.findByLabelText("Event"), { target: { value: "Terminated" } });

    expect(
      await view.findByText("Pays out 9.5 days of time off.", {}, { timeout: 2000 }),
    ).toBeInTheDocument();
    expect(view.getByText("Ends the time off policy.")).toBeInTheDocument();
    expect(view.getByText("Driver PTO")).toBeInTheDocument();
    expect(view.getByText("Vacation")).toBeInTheDocument();
    expect(view.getByText("Ends the pay assignment.")).toBeInTheDocument();
    expect(view.getByText("Revokes Dash access.")).toBeInTheDocument();
    expect(view.getByText("Starts the Offboarding checklist.")).toBeInTheDocument();
    expect(mocks.previewWorkerEmploymentEvent).toHaveBeenLastCalledWith(
      expect.objectContaining({
        workerId: "wrk_1",
        kind: "Terminated",
        effectiveAt: getTodayDate(),
      }),
      { signal: expect.any(AbortSignal) },
    );
  });

  it("waits for the kind's own fields before reading the preview", async () => {
    mocks.fetchWorkerEmploymentEventOptions.mockResolvedValue(activeOptions);
    renderPane();

    const view = await pane();
    fireEvent.change(await view.findByLabelText("Event"), { target: { value: "RateChanged" } });
    expect(
      view.getByText("Fill in the event's details to see what it changes."),
    ).toBeInTheDocument();
    await act(() => new Promise((resolve) => setTimeout(resolve, 400)));
    expect(mocks.previewWorkerEmploymentEvent).not.toHaveBeenCalled();
  });

  it("says when the preview could not be read, and saving still works", async () => {
    mocks.fetchWorkerEmploymentEventOptions.mockResolvedValue(activeOptions);
    mocks.previewWorkerEmploymentEvent.mockRejectedValue(new Error("down"));
    renderPane();

    const view = await pane();
    fireEvent.change(await view.findByLabelText("Event"), { target: { value: "ProbationEnded" } });
    expect(
      await view.findByText(
        "Could not read what this will change. Saving still works.",
        {},
        { timeout: 2000 },
      ),
    ).toBeInTheDocument();
  });

  it("requires a reason for the kinds the server names, then records with today's date", async () => {
    mocks.fetchWorkerEmploymentEventOptions.mockResolvedValue(activeOptions);
    mocks.previewWorkerEmploymentEvent.mockResolvedValue({
      kind: "Suspended",
      effectiveAt: getTodayDate(),
      fromValues: [],
      toValues: [],
      effects: [],
    });
    mocks.recordWorkerEmploymentEvent.mockResolvedValue({ event: { id: "wee_1" }, effects: [] });
    const { onClose } = renderPane();

    const view = await pane();
    fireEvent.change(await view.findByLabelText("Event"), { target: { value: "Suspended" } });
    fireEvent.click(view.getByRole("button", { name: "Record event" }));
    expect(await view.findByText("A reason is required to record “Suspended”")).toBeInTheDocument();
    expect(mocks.recordWorkerEmploymentEvent).not.toHaveBeenCalled();

    fireEvent.change(inputById("reason"), { target: { value: "Safety review" } });
    fireEvent.click(view.getByRole("button", { name: "Record event" }));
    await waitFor(() => expect(mocks.recordWorkerEmploymentEvent).toHaveBeenCalledTimes(1));
    expect(mocks.recordWorkerEmploymentEvent).toHaveBeenCalledWith({
      workerId: "wrk_1",
      kind: "Suspended",
      effectiveAt: getTodayDate(),
      reason: "Safety review",
      notes: null,
      documentId: null,
      fleetCodeId: undefined,
      managerId: undefined,
      driverType: undefined,
      workerType: undefined,
      rate: undefined,
      rateUnit: undefined,
      leaveType: undefined,
    });
    await waitFor(() => expect(onClose).toHaveBeenCalled());
    // Nothing happened out of sight, and the event itself shows on the history.
    expect(mocks.toast.success).not.toHaveBeenCalled();
  });

  it("asks a transfer for a fleet code or a manager, and takes the manager alone", async () => {
    mocks.fetchWorkerEmploymentEventOptions.mockResolvedValue(activeOptions);
    mocks.previewWorkerEmploymentEvent.mockResolvedValue({
      kind: "Transferred",
      effectiveAt: getTodayDate(),
      fromValues: [],
      toValues: [],
      effects: [],
    });
    mocks.recordWorkerEmploymentEvent.mockResolvedValue({ event: { id: "wee_2" }, effects: [] });
    renderPane();

    const view = await pane();
    fireEvent.change(await view.findByLabelText("Event"), { target: { value: "Transferred" } });
    fireEvent.click(view.getByRole("button", { name: "Record event" }));
    expect(await view.findAllByText("Choose a fleet code or a manager")).toHaveLength(2);
    expect(mocks.recordWorkerEmploymentEvent).not.toHaveBeenCalled();

    fireEvent.change(view.getByLabelText("Manager"), { target: { value: "usr_12" } });
    fireEvent.click(view.getByRole("button", { name: "Record event" }));
    await waitFor(() => expect(mocks.recordWorkerEmploymentEvent).toHaveBeenCalledTimes(1));
    expect(mocks.recordWorkerEmploymentEvent.mock.calls[0][0]).toMatchObject({
      kind: "Transferred",
      managerId: "usr_12",
      fleetCodeId: undefined,
    });
  });

  it("tells the recorder what the save did, and each effect that failed", async () => {
    mocks.fetchWorkerEmploymentEventOptions.mockResolvedValue(activeOptions);
    mocks.previewWorkerEmploymentEvent.mockResolvedValue({
      kind: "Terminated",
      effectiveAt: getTodayDate(),
      fromValues: [],
      toValues: [],
      effects: terminationEffects,
    });
    mocks.recordWorkerEmploymentEvent.mockResolvedValue({
      event: { id: "wee_3" },
      effects: [terminationEffects[1], { ...terminationEffects[3], failed: true }],
    });
    renderPane("Terminated");

    const view = await pane();
    await view.findByRole("button", { name: "End employment" });
    fireEvent.change(inputById("reason"), { target: { value: "Resigned" } });
    fireEvent.click(view.getByRole("button", { name: "End employment" }));

    await waitFor(() => expect(mocks.toast.success).toHaveBeenCalledTimes(1));
    expect(mocks.toast.success).toHaveBeenCalledWith("Terminated recorded", {
      description: "Paid out 9.5 days of time off.",
    });
    expect(mocks.toast.warning).toHaveBeenCalledWith(
      "Dash access could not be shut off. Revoke it by hand.",
      expect.objectContaining({ description: expect.any(String) }),
    );
  });
});

describe("AmendEventPane", () => {
  it("amends the narrative with a note and the version", async () => {
    mocks.amendWorkerEmploymentEvent.mockResolvedValue({ id: "wee_3" });
    const onClose = vi.fn();
    render(
      wrap(
        <AmendEventPane
          workerId="wrk_1"
          onClose={onClose}
          event={
            {
              id: "wee_3",
              kind: "Suspended",
              effectiveAt: 1_750_000_000,
              reason: "Safety review",
              notes: null,
              version: 2,
              fromValues: [],
              toValues: [],
            } as never
          }
        />,
      ),
    );

    const view = within(await screen.findByRole("dialog", { name: "Amend event" }));
    expect(view.getByText("Suspended")).toBeInTheDocument();
    fireEvent.change(inputById("amendmentNote"), { target: { value: "Wrong date" } });
    fireEvent.change(view.getByLabelText("Effective"), { target: { value: "1750086400" } });
    fireEvent.click(view.getByRole("button", { name: "Save amendment" }));

    await waitFor(() => expect(mocks.amendWorkerEmploymentEvent).toHaveBeenCalledTimes(1));
    expect(mocks.amendWorkerEmploymentEvent).toHaveBeenCalledWith({
      id: "wee_3",
      effectiveAt: 1_750_086_400,
      reason: "Safety review",
      notes: null,
      amendmentNote: "Wrong date",
      version: 2,
    });
    await waitFor(() => expect(onClose).toHaveBeenCalled());
  });
});
