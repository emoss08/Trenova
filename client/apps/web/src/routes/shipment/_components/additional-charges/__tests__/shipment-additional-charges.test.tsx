import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import type { DetentionOccurrence } from "@trenova/shared/types/detention";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { ReactNode } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import AdditionalChargesSection from "../shipment-additional-charges";

const CHARGE_ID = "ac_01DETENTION0000000000000001";
const SHIPMENT_ID = "shp_01SHIPMENT000000000000000001";

const occurrences: DetentionOccurrence[] = [];

vi.mock("@/lib/queries", () => ({
  queries: {
    detention: {
      byShipment: (shipmentId: string) => ({
        queryKey: ["byShipment", shipmentId],
        queryFn: async () => occurrences,
      }),
    },
  },
}));
vi.mock("@/routes/detention-desk/_components/occurrence-detail-sheet", () => ({
  OccurrenceDetailSheet: () => null,
}));
vi.mock("../shipment-additional-charges-dialog", () => ({
  AdditionalChargeDialog: () => null,
}));
vi.mock("@/components/autocomplete-fields", () => ({
  CustomerAutocompleteField: () => null,
}));

afterEach(() => {
  cleanup();
  occurrences.length = 0;
});

function occurrence(
  overrides: Partial<DetentionOccurrence> & Pick<DetentionOccurrence, "id">,
): DetentionOccurrence {
  return {
    organizationId: "org_1",
    businessUnitId: "bu_1",
    shipmentId: SHIPMENT_ID,
    shipmentMoveId: "smv_1",
    stopId: "stp_1",
    customerId: "cus_1",
    locationId: "loc_1",
    stopType: "Pickup",
    scheduleType: "Appointment",
    arrivedAt: 1_700_000_000,
    departedAt: 1_700_010_000,
    isOpen: false,
    arrivedLate: false,
    lateByMinutes: 0,
    freeMinutesGranted: 120,
    rawDwellMinutes: 210,
    billableMinutes: 90,
    roundedMinutes: 90,
    billableUnits: 1.5,
    grossAmount: 112.5,
    billableAmount: 112.5,
    waivedAmount: 0,
    capApplied: "None",
    convertedToLayover: false,
    currency: "USD",
    status: "Approved",
    notificationStatus: "NotRequired",
    suppressedByGate: false,
    requiresApproval: false,
    collectabilityScore: 80,
    version: 1,
    createdAt: 1_700_000_000,
    updatedAt: 1_700_000_000,
    additionalChargeId: CHARGE_ID,
    locationName: "Acme DC",
    customerName: "Acme",
    ...overrides,
  } as DetentionOccurrence;
}

function Harness({ values, children }: { values: Partial<Shipment>; children: ReactNode }) {
  const form = useForm<Shipment>({ defaultValues: values as Shipment });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={client}>
      <FormProvider {...form}>{children}</FormProvider>
    </QueryClientProvider>
  );
}

function detentionCharge(): Shipment["additionalCharges"][number] {
  return {
    id: CHARGE_ID,
    organizationId: "org_1",
    businessUnitId: "bu_1",
    accessorialChargeId: "acc_1",
    isSystemGenerated: true,
    isDetention: true,
    method: "Flat",
    amount: 562.5,
    unit: 1,
    version: 0,
    allocations: [],
    accessorialCharge: { code: "DET", description: "Detention Fee" } as never,
  } as Shipment["additionalCharges"][number];
}

describe("AdditionalChargesSection detention rows", () => {
  it("shows one detention charge covering every detained stop", async () => {
    occurrences.push(
      occurrence({
        id: "dto_pickup",
        stopType: "Pickup",
        roundedMinutes: 90,
        billableAmount: 112.5,
      }),
      occurrence({
        id: "dto_delivery",
        stopType: "Delivery",
        locationName: "Cold Storage PHX",
        arrivedAt: 1_700_100_000,
        roundedMinutes: 360,
        rawDwellMinutes: 480,
        billableAmount: 450,
      }),
    );

    render(
      <Harness values={{ id: SHIPMENT_ID, version: 2, additionalCharges: [detentionCharge()] }}>
        <AdditionalChargesSection />
      </Harness>,
    );

    expect(await screen.findByText("2 stops")).toBeInTheDocument();
    expect(screen.getByText("7h 30m")).toBeInTheDocument();
    expect(screen.getByText("$562.50")).toBeInTheDocument();
    expect(screen.getAllByText("DET")).toHaveLength(1);
  });

  it("marks a detention charge before its occurrences have loaded", () => {
    render(
      <Harness values={{ id: SHIPMENT_ID, version: 2, additionalCharges: [detentionCharge()] }}>
        <AdditionalChargesSection />
      </Harness>,
    );

    expect(screen.getByText("Detention")).toBeInTheDocument();
    expect(screen.queryByText("2 stops")).not.toBeInTheDocument();
    expect(screen.getByText("1")).toBeInTheDocument();
  });

  it("does not treat a manual charge as detention", () => {
    render(
      <Harness
        values={{
          id: SHIPMENT_ID,
          version: 2,
          additionalCharges: [{ ...detentionCharge(), isSystemGenerated: false }],
        }}
      >
        <AdditionalChargesSection />
      </Harness>,
    );

    expect(screen.queryByText("Detention")).not.toBeInTheDocument();
  });
});

describe("AdditionalChargesSection payers", () => {
  // Detention and fuel surcharge rows cannot be edited, but who pays them can
  // change, so every row carries the payer control, not only manual charges.
  it("names who pays each charge, including system-generated ones", () => {
    render(
      <Harness
        values={{
          id: SHIPMENT_ID,
          version: 2,
          customerId: "cus_acme",
          customer: { id: "cus_acme", name: "Acme Manufacturing", code: "ACME" } as never,
          additionalCharges: [
            {
              ...detentionCharge(),
              allocations: [
                {
                  billToCustomerId: "cus_peak",
                  method: "Percent",
                  percent: 100,
                  amount: null,
                  sequence: 0,
                  billToCustomer: { id: "cus_peak", name: "Peak Distributing", code: "PEAK" },
                },
              ],
            },
            {
              ...detentionCharge(),
              id: "ac_manual",
              isSystemGenerated: false,
              isDetention: false,
              allocations: [],
            },
          ] as Shipment["additionalCharges"],
        }}
      >
        <AdditionalChargesSection />
      </Harness>,
    );

    const controls = screen.getAllByTestId("charge-payer-control");
    expect(controls).toHaveLength(2);
    expect(controls[0]).toHaveTextContent("Bill to: PEAK – Peak Distributing");
    expect(controls[1]).toHaveTextContent("Bill to: Same as shipment");
  });
});
