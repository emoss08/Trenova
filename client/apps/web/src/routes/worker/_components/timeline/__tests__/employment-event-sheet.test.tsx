import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { getTodayDate } from "@trenova/shared/lib/date";
import { useController, type Control } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { EmploymentEventSheet } from "../employment-event-dialog";

const { recordWorkerEmploymentEvent, amendWorkerEmploymentEvent, toast } = vi.hoisted(() => ({
  recordWorkerEmploymentEvent: vi.fn(),
  amendWorkerEmploymentEvent: vi.fn(),
  toast: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
}));

vi.mock("@/lib/graphql/worker-employment", () => ({
  recordWorkerEmploymentEvent,
  amendWorkerEmploymentEvent,
  WORKER_EMPLOYMENT_EVENTS_KEY: "worker-employment-events",
}));

vi.mock("sonner", () => ({ toast }));

vi.mock("@/components/autocomplete-fields", () => ({
  FleetCodeAutocompleteField: ({ control, name }: { control: Control; name: string }) => {
    const { field, fieldState } = useController({ control, name });
    return (
      <>
        <input
          aria-label="Fleet code"
          value={(field.value as string) ?? ""}
          onChange={(event) => field.onChange(event.target.value)}
        />
        {fieldState.error?.message ? <p>{fieldState.error.message}</p> : null}
      </>
    );
  },
}));

vi.mock("@/components/fields/select-field", () => ({
  SelectField: ({
    control,
    name,
    label,
    options,
    isReadOnly,
  }: {
    control: Control;
    name: string;
    label: string;
    options: { value: string; label: string }[];
    isReadOnly?: boolean;
  }) => {
    const { field, fieldState } = useController({ control, name });
    return (
      <>
        <select
          aria-label={label}
          disabled={isReadOnly}
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

const worker = { fleetCodeId: "fc_1", driverType: "Local", type: "Employee", status: "Active" };

function inputById(name: string): HTMLInputElement | HTMLTextAreaElement {
  const element = document.getElementById(`input-${name}`) ?? document.getElementById(name);
  if (!(element instanceof HTMLInputElement) && !(element instanceof HTMLTextAreaElement)) {
    throw new Error(`${name} not rendered`);
  }
  return element;
}

function renderSheet(props: Partial<React.ComponentProps<typeof EmploymentEventSheet>> = {}) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const onOpenChange = vi.fn();
  render(
    <QueryClientProvider client={queryClient}>
      <EmploymentEventSheet
        open
        onOpenChange={onOpenChange}
        workerId="wrk_1"
        worker={worker as never}
        mode="record"
        {...props}
      />
    </QueryClientProvider>,
  );
  return { onOpenChange };
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("EmploymentEventSheet", () => {
  it("requires a reason for a termination and then records it with today's date", async () => {
    recordWorkerEmploymentEvent.mockResolvedValue({
      event: { id: "wee_1" },
      cascade: {
        ptoAssignmentEnded: true,
        payAssignmentEnded: false,
        upcomingPtoCancelled: 2,
        defaultPolicyApplied: false,
        checklistStarted: false,
      },
    });
    const today = getTodayDate();
    const { onOpenChange } = renderSheet();

    fireEvent.change(screen.getByLabelText("Event"), { target: { value: "Terminated" } });
    fireEvent.click(screen.getByRole("button", { name: "Record Terminated" }));
    expect(
      await screen.findByText("A reason is required when recording terminated"),
    ).toBeInTheDocument();
    expect(recordWorkerEmploymentEvent).not.toHaveBeenCalled();

    fireEvent.change(inputById("reason"), { target: { value: "Resigned" } });
    fireEvent.click(screen.getByRole("button", { name: "Record Terminated" }));

    await waitFor(() => expect(recordWorkerEmploymentEvent).toHaveBeenCalledTimes(1));
    expect(recordWorkerEmploymentEvent).toHaveBeenCalledWith({
      workerId: "wrk_1",
      kind: "Terminated",
      effectiveAt: today,
      reason: "Resigned",
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
    await waitFor(() => expect(onOpenChange).toHaveBeenCalledWith(false));
    expect(toast.success).toHaveBeenCalledWith(
      "Terminated recorded",
      expect.objectContaining({ description: expect.stringContaining("2 upcoming PTO") }),
    );
  });

  it("asks for the destination fleet on a transfer", async () => {
    recordWorkerEmploymentEvent.mockResolvedValue({
      event: { id: "wee_2" },
      cascade: {
        ptoAssignmentEnded: false,
        payAssignmentEnded: false,
        upcomingPtoCancelled: 0,
        defaultPolicyApplied: false,
        checklistStarted: false,
      },
    });
    renderSheet();

    fireEvent.change(screen.getByLabelText("Event"), { target: { value: "Transferred" } });
    fireEvent.click(screen.getByRole("button", { name: "Record Transferred" }));
    expect(await screen.findByText("Choose the fleet the worker is moving to")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Fleet code"), { target: { value: "fc_9" } });
    fireEvent.click(screen.getByRole("button", { name: "Record Transferred" }));
    await waitFor(() => expect(recordWorkerEmploymentEvent).toHaveBeenCalledTimes(1));
    expect(recordWorkerEmploymentEvent.mock.calls[0][0]).toMatchObject({
      kind: "Transferred",
      fleetCodeId: "fc_9",
    });
  });

  it("asks what kind of leave is starting and sends it with the event", async () => {
    recordWorkerEmploymentEvent.mockResolvedValue({
      event: { id: "wee_3" },
      cascade: {
        ptoAssignmentEnded: false,
        payAssignmentEnded: false,
        upcomingPtoCancelled: 0,
        defaultPolicyApplied: false,
        checklistStarted: false,
        ptoPaidOutDays: "0.00",
        ptoForfeitedDays: "0.00",
      },
    });
    renderSheet();

    fireEvent.change(screen.getByLabelText("Event"), { target: { value: "LeaveStarted" } });
    expect(screen.getByLabelText("Leave type")).toBeInTheDocument();
    fireEvent.change(inputById("reason"), { target: { value: "Surgery" } });
    fireEvent.click(screen.getByRole("button", { name: "Record Leave started" }));
    expect(await screen.findByText("Choose the kind of leave")).toBeInTheDocument();
    expect(recordWorkerEmploymentEvent).not.toHaveBeenCalled();

    fireEvent.change(screen.getByLabelText("Leave type"), { target: { value: "Medical" } });
    fireEvent.click(screen.getByRole("button", { name: "Record Leave started" }));
    await waitFor(() => expect(recordWorkerEmploymentEvent).toHaveBeenCalledTimes(1));
    expect(recordWorkerEmploymentEvent.mock.calls[0][0]).toMatchObject({
      kind: "LeaveStarted",
      leaveType: "Medical",
      reason: "Surgery",
    });
  });

  it("tells the recorder how a termination settled the PTO balances", async () => {
    recordWorkerEmploymentEvent.mockResolvedValue({
      event: { id: "wee_4" },
      cascade: {
        ptoAssignmentEnded: true,
        payAssignmentEnded: true,
        upcomingPtoCancelled: 0,
        defaultPolicyApplied: false,
        checklistStarted: true,
        ptoPaidOutDays: "6.50",
        ptoForfeitedDays: "2.00",
      },
    });
    renderSheet();

    fireEvent.change(screen.getByLabelText("Event"), { target: { value: "Terminated" } });
    fireEvent.change(inputById("reason"), { target: { value: "Resigned" } });
    fireEvent.click(screen.getByRole("button", { name: "Record Terminated" }));
    await waitFor(() => expect(toast.success).toHaveBeenCalledTimes(1));
    expect(toast.success).toHaveBeenCalledWith(
      "Terminated recorded",
      expect.objectContaining({
        description: expect.stringMatching(/Paid out 6\.5 PTO days.*Forfeited 2 PTO days/),
      }),
    );
  });

  it("tells the recorder that the termination shut off the worker's Dash sign-in", async () => {
    recordWorkerEmploymentEvent.mockResolvedValue({
      event: { id: "wee_5" },
      cascade: {
        ptoAssignmentEnded: true,
        payAssignmentEnded: true,
        upcomingPtoCancelled: 0,
        defaultPolicyApplied: false,
        checklistStarted: true,
        ptoPaidOutDays: "0.00",
        ptoForfeitedDays: "0.00",
        trainingAssigned: 0,
        portalAccessRevoked: true,
        portalRevocationError: "",
      },
    });
    renderSheet();

    fireEvent.change(screen.getByLabelText("Event"), { target: { value: "Terminated" } });
    fireEvent.change(inputById("reason"), { target: { value: "Resigned" } });
    fireEvent.click(screen.getByRole("button", { name: "Record Terminated" }));
    await waitFor(() => expect(toast.success).toHaveBeenCalledTimes(1));
    expect(toast.success).toHaveBeenCalledWith(
      "Terminated recorded",
      expect.objectContaining({
        description: expect.stringContaining("Revoked the driver portal sign-in"),
      }),
    );
  });

  // The termination has already taken effect, so a portal failure is reported
  // rather than raised. The recorder has to be told to close the account by
  // hand, otherwise a departed driver keeps a working sign-in.
  it("warns the recorder when the portal sign-in could not be shut off", async () => {
    recordWorkerEmploymentEvent.mockResolvedValue({
      event: { id: "wee_6" },
      cascade: {
        ptoAssignmentEnded: true,
        payAssignmentEnded: false,
        upcomingPtoCancelled: 0,
        defaultPolicyApplied: false,
        checklistStarted: true,
        ptoPaidOutDays: "0.00",
        ptoForfeitedDays: "0.00",
        trainingAssigned: 0,
        portalAccessRevoked: false,
        portalRevocationError: "portal is unreachable",
      },
    });
    renderSheet();

    fireEvent.change(screen.getByLabelText("Event"), { target: { value: "Terminated" } });
    fireEvent.change(inputById("reason"), { target: { value: "Resigned" } });
    fireEvent.click(screen.getByRole("button", { name: "Record Terminated" }));
    await waitFor(() => expect(toast.warning).toHaveBeenCalledTimes(1));
    expect(toast.warning).toHaveBeenCalledWith(
      "Revoke the driver portal sign-in by hand",
      expect.objectContaining({
        description: expect.stringContaining("portal is unreachable"),
      }),
    );
  });

  // The cascade already reports courses opened by a hire or rehire, but the
  // summary dropped them on the floor.
  it("tells the recorder how many required courses a rehire opened", async () => {
    recordWorkerEmploymentEvent.mockResolvedValue({
      event: { id: "wee_7" },
      cascade: {
        ptoAssignmentEnded: false,
        payAssignmentEnded: false,
        upcomingPtoCancelled: 0,
        defaultPolicyApplied: true,
        checklistStarted: true,
        ptoPaidOutDays: "0.00",
        ptoForfeitedDays: "0.00",
        trainingAssigned: 3,
        portalAccessRevoked: false,
        portalRevocationError: "",
      },
    });
    renderSheet({ worker: { ...worker, status: "Inactive" } as never });

    fireEvent.change(screen.getByLabelText("Event"), { target: { value: "Rehired" } });
    fireEvent.change(inputById("reason"), { target: { value: "Back on" } });
    fireEvent.click(screen.getByRole("button", { name: "Record Rehired" }));
    await waitFor(() => expect(toast.success).toHaveBeenCalledTimes(1));
    expect(toast.success).toHaveBeenCalledWith(
      "Rehired recorded",
      expect.objectContaining({
        description: expect.stringContaining("Opened 3 required courses"),
      }),
    );
  });

  it("offers only the kinds the worker's state allows", async () => {
    renderSheet({ worker: { ...worker, status: "Inactive" } as never });
    const select = screen.getByLabelText("Event") as HTMLSelectElement;
    const options = Array.from(select.options).map((option) => option.value);
    expect(options).toContain("Rehired");
    expect(options).not.toContain("Terminated");
    expect(options).not.toContain("Promoted");
  });

  it("amends the narrative with a note and the version", async () => {
    amendWorkerEmploymentEvent.mockResolvedValue({ id: "wee_3" });
    renderSheet({
      mode: "amend",
      event: {
        id: "wee_3",
        kind: "Suspended",
        effectiveAt: 1_750_000_000,
        reason: "Safety review",
        notes: null,
        version: 2,
      } as never,
    });

    fireEvent.change(inputById("amendmentNote"), { target: { value: "Wrong date" } });
    fireEvent.change(screen.getByLabelText("Effective"), { target: { value: "1750086400" } });
    fireEvent.click(screen.getByRole("button", { name: "Save amendment" }));

    await waitFor(() => expect(amendWorkerEmploymentEvent).toHaveBeenCalledTimes(1));
    expect(amendWorkerEmploymentEvent).toHaveBeenCalledWith({
      id: "wee_3",
      effectiveAt: 1_750_086_400,
      reason: "Safety review",
      notes: null,
      amendmentNote: "Wrong date",
      version: 2,
    });
    expect(recordWorkerEmploymentEvent).not.toHaveBeenCalled();
  });
});
