import type { CarrierIntelEventRow } from "@/lib/graphql/carrier-monitoring-table";
import type { CarrierIntelEventStatus } from "@trenova/graphql/generated/graphql";
import { act, renderHook } from "@testing-library/react";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { acknowledgeableEventIds, useEventInboxActions } from "../use-event-inbox-actions";

const mocks = vi.hoisted(() => ({
  acknowledgeCarrierIntelEvents: vi.fn(),
  handleMutationError: vi.fn(),
  toastSuccess: vi.fn(),
  toastInfo: vi.fn(),
  toastError: vi.fn(),
}));

vi.mock("@/lib/graphql/carrier-intelligence", () => ({
  acknowledgeCarrierIntelEvents: mocks.acknowledgeCarrierIntelEvents,
}));
vi.mock("@/hooks/use-api-mutation", () => ({
  handleMutationError: mocks.handleMutationError,
}));
vi.mock("sonner", () => ({
  toast: { success: mocks.toastSuccess, info: mocks.toastInfo, error: mocks.toastError },
}));

afterEach(() => {
  vi.clearAllMocks();
});

function eventRow(id: string, status: CarrierIntelEventStatus): CarrierIntelEventRow {
  return {
    id,
    subjectType: "Carrier",
    subjectId: `car_${id}`,
    carrierId: `car_${id}`,
    dotNumber: "1234567",
    subjectName: "Blue Ridge Freight",
    provider: "CarrierOK",
    source: "NativeChangeFeed",
    category: "Insurance",
    fieldPath: "insurance.bipdOnFile",
    ruleCode: "insurance.bipd_below_minimum",
    severity: "High",
    action: "Block",
    priorValue: "750000",
    currentValue: "0",
    summary: "BIPD coverage was cancelled",
    vendorChangedAt: null,
    detectedAt: 1_800_000_000,
    status,
    acknowledgedById: null,
    acknowledgedAt: null,
    resolvedById: null,
    resolvedAt: null,
    resolution: null,
    resolutionNote: null,
    snapshotId: null,
    version: 1,
    createdAt: 1_800_000_000,
    updatedAt: 1_800_000_000,
  };
}

function wrapper({ children }: { children: ReactNode }) {
  return <MemoryRouter>{children}</MemoryRouter>;
}

function renderActions(canUpdate: boolean) {
  const onChanged = vi.fn();
  const hook = renderHook(() => useEventInboxActions({ canUpdate, onChanged }), { wrapper });
  return { hook, onChanged };
}

describe("acknowledgeableEventIds", () => {
  it("keeps only open events from a selection", () => {
    expect(
      acknowledgeableEventIds([
        eventRow("1", "Open"),
        eventRow("2", "Acknowledged"),
        eventRow("3", "Resolved"),
        eventRow("4", "Open"),
        eventRow("5", "Dismissed"),
      ]),
    ).toEqual(["1", "4"]);
  });
});

describe("useEventInboxActions bulk acknowledge", () => {
  it("offers no bulk actions without update permission", () => {
    const { hook } = renderActions(false);

    expect(hook.result.current.dockActions).toEqual([]);
  });

  it("acknowledges only the open events in the selection and reports the skipped ones", async () => {
    mocks.acknowledgeCarrierIntelEvents.mockResolvedValue(2);
    const { hook, onChanged } = renderActions(true);
    const [action] = hook.result.current.dockActions;

    expect(action.id).toBe("acknowledge-events");
    expect(action.clearSelectionOnSuccess).toBe(true);

    await act(async () => {
      if (action.type === "select") {
        throw new Error("expected a simple dock action");
      }
      await action.onClick([
        eventRow("1", "Open"),
        eventRow("2", "Resolved"),
        eventRow("3", "Open"),
      ]);
    });

    expect(mocks.acknowledgeCarrierIntelEvents).toHaveBeenCalledWith(["1", "3"]);
    expect(mocks.toastSuccess).toHaveBeenCalledTimes(1);
    expect(mocks.toastSuccess.mock.calls[0][1]).toEqual({ description: expect.any(String) });
    expect(onChanged).toHaveBeenCalledTimes(1);
  });

  it("does not call the server when nothing selected is open", async () => {
    const { hook, onChanged } = renderActions(true);
    const [action] = hook.result.current.dockActions;

    await act(async () => {
      if (action.type === "select") {
        throw new Error("expected a simple dock action");
      }
      await action.onClick([eventRow("1", "Acknowledged"), eventRow("2", "Resolved")]);
    });

    expect(mocks.acknowledgeCarrierIntelEvents).not.toHaveBeenCalled();
    expect(mocks.toastInfo).toHaveBeenCalledTimes(1);
    expect(onChanged).not.toHaveBeenCalled();
  });

  it("surfaces the server error and rejects so the selection is kept", async () => {
    const failure = new Error("boom");
    mocks.acknowledgeCarrierIntelEvents.mockRejectedValue(failure);
    const { hook, onChanged } = renderActions(true);
    const [action] = hook.result.current.dockActions;

    await act(async () => {
      if (action.type === "select") {
        throw new Error("expected a simple dock action");
      }
      await expect(action.onClick([eventRow("1", "Open")])).rejects.toBe(failure);
    });

    expect(mocks.handleMutationError).toHaveBeenCalledWith({
      error: failure,
      resourceName: "Carrier intelligence event",
    });
    expect(mocks.toastSuccess).not.toHaveBeenCalled();
    expect(onChanged).not.toHaveBeenCalled();
  });

  it("hides acknowledge and resolve row actions for closed events", () => {
    const { hook } = renderActions(true);
    const actions = hook.result.current.contextMenuActions;
    const acknowledge = actions.find((entry) => entry.id === "acknowledge-event");
    const resolve = actions.find((entry) => entry.id === "resolve-event");
    const resolvedRow = { original: eventRow("1", "Resolved") } as Parameters<
      NonNullable<(typeof actions)[number]["hidden"]>
    >[0];
    const acknowledgedRow = { original: eventRow("2", "Acknowledged") } as typeof resolvedRow;

    expect(acknowledge?.hidden?.(resolvedRow)).toBe(true);
    expect(resolve?.hidden?.(resolvedRow)).toBe(true);
    expect(acknowledge?.hidden?.(acknowledgedRow)).toBe(true);
    expect(resolve?.hidden?.(acknowledgedRow)).toBe(false);
  });
});
