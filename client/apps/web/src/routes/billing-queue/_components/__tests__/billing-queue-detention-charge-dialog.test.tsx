import { TestRouter } from "@trenova/shared/testing/test-router";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import {
  detentionOccurrenceSchema,
  type DetentionOccurrence,
} from "@trenova/shared/types/detention";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueDetentionChargeDialog } from "../billing-queue-detention-charge-dialog";

const service = vi.hoisted(() => ({
  byShipment: vi.fn(),
  approve: vi.fn(),
  waive: vi.fn(),
  dispute: vi.fn(),
}));

vi.mock("@/services/detention", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/services/detention")>()),
  detentionService: service,
}));

afterEach(() => {
  cleanup();
  clearPermissions();
  vi.clearAllMocks();
});

/** An occurrence as the GraphQL contract sends it, parsed by the schema the service uses. */
function occurrence(overrides: Record<string, unknown>): DetentionOccurrence {
  return detentionOccurrenceSchema.parse({
    id: "dto_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    shipmentId: "shp_1",
    shipmentMoveId: "sm_1",
    stopId: "stp_1",
    customerId: "cus_1",
    locationId: "loc_1",
    stopType: "Pickup",
    scheduleType: "Open",
    clockStartAt: 1_793_800_000,
    freeTimeExpiresAt: 1_793_807_200,
    isOpen: false,
    arrivedLate: true,
    lateByMinutes: 0,
    freeMinutesGranted: 120,
    rawDwellMinutes: 1500,
    billableMinutes: 1380,
    roundedMinutes: 1380,
    billableUnits: "23",
    grossAmount: "1725",
    billableAmount: "1425",
    driverPayMinutes: 1380,
    driverPayAmount: null,
    netMargin: null,
    capApplied: "MaxChargePerDay",
    convertedToLayover: false,
    currency: "USD",
    status: "Pending",
    notificationStatus: "Missed",
    suppressedByGate: false,
    requiresApproval: true,
    waivedAmount: "0",
    collectabilityScore: 25,
    locationName: "Kestrel Valve Works Plant 1",
    version: 1,
    createdAt: 1_793_800_000,
    updatedAt: 1_793_800_000,
    ...overrides,
  });
}

function draw(occurrences: DetentionOccurrence[]) {
  service.byShipment.mockResolvedValue(occurrences);
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={queryClient}>
      <TestRouter>
        <BillingQueueDetentionChargeDialog shipmentId="shp_1" open onOpenChange={() => {}} />
      </TestRouter>
    </QueryClientProvider>,
  );
}

async function rowFor(name: string): Promise<HTMLElement> {
  return (await screen.findByText(name)).closest("li") as HTMLElement;
}

function buttonsIn(row: HTMLElement): string[] {
  return within(row)
    .queryAllByRole("button")
    .map((button) => button.textContent ?? "");
}

describe("BillingQueueDetentionChargeDialog", () => {
  it("lists each stop behind the line with what a biller may still do to it", async () => {
    seedPermissions({ [Resource.Detention]: Operation.Read | Operation.Update });
    draw([
      occurrence({}),
      occurrence({
        id: "dto_2",
        locationName: "Prairie Industrial Supply Birmingham",
        status: "Approved",
        requiresApproval: false,
        billableAmount: "1650",
      }),
      occurrence({
        id: "dto_3",
        locationName: "Midland Fluid Supply Detroit",
        status: "Waived",
        requiresApproval: false,
        billableAmount: "300",
        waivedAmount: "300",
      }),
      occurrence({
        id: "dto_4",
        locationName: "Madison",
        status: "NotBillable",
        billableAmount: "0",
      }),
    ]);

    const held = await rowFor("Kestrel Valve Works Plant 1");
    expect(held).toHaveTextContent("Waiting on approval");
    expect(held).toHaveTextContent("$1,425.00");
    expect(buttonsIn(held)).toEqual(["Approve", "Waive"]);

    const approved = await rowFor("Prairie Industrial Supply Birmingham");
    expect(approved).toHaveTextContent("Approved");
    expect(buttonsIn(approved)).toEqual(["Waive"]);

    const waived = await rowFor("Midland Fluid Supply Detroit");
    expect(waived).toHaveTextContent("Waived");
    expect(waived).toHaveTextContent("$300.00");
    expect(buttonsIn(waived)).toEqual([]);

    expect(screen.queryByText("Madison")).not.toBeInTheDocument();
    expect(service.byShipment).toHaveBeenCalledWith("shp_1", expect.anything());
  });

  it("shows the stops without a way to decide them to someone who cannot decide detention", async () => {
    seedPermissions({ [Resource.Detention]: Operation.Read });
    draw([occurrence({})]);

    const held = await rowFor("Kestrel Valve Works Plant 1");
    expect(buttonsIn(held)).toEqual([]);
  });

  it("approves a held stop from the line", async () => {
    seedPermissions({ [Resource.Detention]: Operation.Read | Operation.Update });
    service.approve.mockResolvedValue(occurrence({ status: "Approved" }));
    draw([occurrence({})]);

    const held = await rowFor("Kestrel Valve Works Plant 1");
    fireEvent.click(within(held).getByRole("button", { name: "Approve" }));

    await waitFor(() => expect(service.approve).toHaveBeenCalledWith("dto_1"));
  });

  it("says so when no stop on the shipment bills detention", async () => {
    seedPermissions({ [Resource.Detention]: Operation.Read | Operation.Update });
    draw([]);

    expect(
      await screen.findByText("No stop on this shipment bills detention."),
    ).toBeInTheDocument();
  });
});
