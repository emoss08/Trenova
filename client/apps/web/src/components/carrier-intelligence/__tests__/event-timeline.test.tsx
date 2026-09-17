import type { CarrierIntelEvent } from "@/lib/graphql/carrier-intelligence";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { EventTimeline } from "../event-timeline";
import { IntelTestProviders } from "./fixtures";

const mocks = vi.hoisted(() => ({
  acknowledge: vi.fn(),
  toastSuccess: vi.fn(),
}));

vi.mock("@/lib/graphql/carrier-intelligence", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/carrier-intelligence")>()),
  acknowledgeCarrierIntelEvents: mocks.acknowledge,
}));

vi.mock("sonner", () => ({ toast: { success: mocks.toastSuccess, error: vi.fn() } }));

afterEach(() => {
  vi.clearAllMocks();
});

const DETECTED_AT = 1_788_220_800;

function event(overrides: Partial<CarrierIntelEvent>): CarrierIntelEvent {
  return {
    id: "cie_1",
    subjectType: "Carrier",
    subjectId: "car_1",
    carrierId: "car_1",
    dotNumber: "1234567",
    subjectName: "Blue Ridge Freight",
    provider: "CarrierOK",
    source: "SnapshotDiff",
    category: "Insurance",
    fieldPath: "insurance.bipdOnFile",
    fieldLabel: "BIPD coverage on file",
    ruleCode: null,
    ruleLabel: null,
    severity: "High",
    action: null,
    priorValue: "750000",
    currentValue: "500000",
    summary: "BIPD coverage changed",
    vendorChangedAt: null,
    detectedAt: DETECTED_AT,
    status: "Open",
    acknowledgedById: null,
    acknowledgedBy: null,
    acknowledgedAt: null,
    resolvedById: null,
    resolvedBy: null,
    resolvedAt: null,
    resolution: null,
    resolutionNote: null,
    snapshotId: null,
    version: 1,
    createdAt: DETECTED_AT,
    updatedAt: DETECTED_AT,
    ...overrides,
  };
}

describe("EventTimeline", () => {
  it("titles changes with the API field label and human values, and rule events with the rule label", () => {
    render(
      <IntelTestProviders>
        <EventTimeline
          events={[
            event({ id: "cie_1" }),
            event({
              id: "cie_2",
              fieldPath: null,
              fieldLabel: null,
              ruleCode: "insurance.bipd_below_required",
              ruleLabel: "Liability below requirement",
              priorValue: null,
              currentValue: null,
              status: "Resolved",
              resolution: "NoActionRequired",
              resolvedAt: DETECTED_AT + 60,
              resolvedById: "usr_1",
              resolvedBy: { id: "usr_1", name: "Dana Whitfield" },
            }),
          ]}
          canUpdate
        />
      </IntelTestProviders>,
    );

    expect(
      screen.getByText("BIPD coverage on file: $750,000.00 → $500,000.00"),
    ).toBeInTheDocument();
    expect(screen.getByText("Liability below requirement")).toBeInTheDocument();
    expect(screen.queryByText("insurance.bipdOnFile")).toBeNull();

    const resolved = document.querySelector<HTMLElement>('[data-event-id="cie_2"]');
    expect(resolved).not.toBeNull();
    expect(within(resolved as HTMLElement).queryByRole("button")).toBeNull();
    expect(
      within(resolved as HTMLElement).getByText(
        /^Resolved as No action required by Dana Whitfield/,
      ),
    ).toBeInTheDocument();
  });

  it("humanizes the field path when an older event has no label", () => {
    render(
      <IntelTestProviders>
        <EventTimeline
          events={[
            event({
              fieldPath: "fleet.powerUnits",
              fieldLabel: null,
              category: "Fleet",
              priorValue: "12",
              currentValue: "9",
            }),
            event({
              id: "cie_2",
              fieldPath: null,
              fieldLabel: null,
              ruleCode: "safety.unsatisfactory_rating",
              ruleLabel: null,
              summary: "Safety rating is Unsatisfactory",
              priorValue: null,
              currentValue: null,
            }),
          ]}
          canUpdate={false}
        />
      </IntelTestProviders>,
    );

    expect(screen.getByText("Power units: 12 → 9")).toBeInTheDocument();
    expect(screen.getByText("Safety rating is Unsatisfactory")).toBeInTheDocument();
  });

  it("names who acknowledged a change", () => {
    render(
      <IntelTestProviders>
        <EventTimeline
          events={[
            event({
              status: "Acknowledged",
              acknowledgedAt: DETECTED_AT + 60,
              acknowledgedById: "usr_2",
              acknowledgedBy: { id: "usr_2", name: "Marcus Lee" },
            }),
          ]}
          canUpdate={false}
        />
      </IntelTestProviders>,
    );

    expect(screen.getByText(/^Acknowledged by Marcus Lee on/)).toBeInTheDocument();
  });

  it("acknowledges an open change", async () => {
    const user = userEvent.setup();
    const onChanged = vi.fn();
    mocks.acknowledge.mockResolvedValue(1);
    render(
      <IntelTestProviders>
        <EventTimeline events={[event({})]} canUpdate onChanged={onChanged} />
      </IntelTestProviders>,
    );

    await user.click(screen.getByRole("button", { name: "Acknowledge" }));

    await waitFor(() => expect(mocks.acknowledge).toHaveBeenCalledWith(["cie_1"]));
    await waitFor(() => expect(onChanged).toHaveBeenCalledTimes(1));
  });

  it("hides actions from users who cannot update", () => {
    render(
      <IntelTestProviders>
        <EventTimeline events={[event({})]} canUpdate={false} />
      </IntelTestProviders>,
    );

    expect(screen.queryByRole("button", { name: "Acknowledge" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Resolve" })).toBeNull();
  });
});
