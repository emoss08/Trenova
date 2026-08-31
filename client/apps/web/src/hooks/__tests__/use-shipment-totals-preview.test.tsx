import { act, renderHook, waitFor } from "@testing-library/react";
import type { Shipment } from "@trenova/shared/types/shipment";
import type { ReactNode } from "react";
import { FormProvider, useForm } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useShipmentTotalsPreview } from "../use-shipment-totals-preview";

const calculateTotalsMock = vi.hoisted(() => vi.fn());

vi.mock("@/services/api", () => ({
  apiService: {
    shipmentService: {
      calculateTotals: calculateTotalsMock,
    },
  },
}));

// The debounce the hook puts in front of every request.
vi.mock("@trenova/shared/hooks/use-debounce", () => ({
  useDebounce: <T,>(value: T) => value,
}));

const totals = {
  freightChargeAmount: 100,
  otherChargeAmount: 0,
  totalChargeAmount: 100,
  fuelSurcharge: null,
};

/**
 * A form seated the way the shipment panel seats a new shipment: the required
 * ids blank, and one move carrying the two empty stop rows the panel adds so
 * the user has somewhere to pick an origin and a destination.
 */
function baseShipment(overrides: Partial<Shipment> = {}): Shipment {
  return {
    customerId: "",
    serviceTypeId: "",
    shipmentTypeId: "",
    formulaTemplateId: "",
    baseRate: 0,
    ratingUnit: 1,
    fuelSurchargeLocked: false,
    additionalCharges: [],
    commodities: [],
    moves: [
      {
        sequence: 0,
        loaded: true,
        stops: [
          { sequence: 0, type: "Pickup", locationId: "" },
          { sequence: 1, type: "Delivery", locationId: "" },
        ],
      },
    ],
    ...overrides,
  } as unknown as Shipment;
}

function Harness({ defaultValues, children }: { defaultValues: Shipment; children: ReactNode }) {
  const form = useForm<Shipment>({ defaultValues });
  return <FormProvider {...form}>{children}</FormProvider>;
}

function renderPreview(defaultValues: Shipment) {
  return renderHook(() => useShipmentTotalsPreview(), {
    wrapper: ({ children }: { children: ReactNode }) => (
      <Harness defaultValues={defaultValues}>{children}</Harness>
    ),
  });
}

const ratable = {
  customerId: "cus_01HTEST0000000000000000001",
  serviceTypeId: "svt_01HTEST0000000000000000001",
  shipmentTypeId: "spt_01HTEST0000000000000000001",
  formulaTemplateId: "ft_01HTEST00000000000000000001",
};

describe("useShipmentTotalsPreview", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    calculateTotalsMock.mockResolvedValue(totals);
  });

  // shipmentFromInput parses serviceTypeId, shipmentTypeId, customerId and
  // formulaTemplateId with a non-null parse. An empty string on any of them
  // fails the whole mutation, so the preview has to wait for all four rather
  // than for the rating method alone.
  it.each([
    ["serviceTypeId", { ...ratable, serviceTypeId: "" }],
    ["shipmentTypeId", { ...ratable, shipmentTypeId: "" }],
    ["customerId", { ...ratable, customerId: "" }],
    ["formulaTemplateId", { ...ratable, formulaTemplateId: "" }],
  ])("does not request totals while %s is blank", async (_field, ids) => {
    renderPreview(baseShipment(ids));

    await act(async () => {
      await Promise.resolve();
    });

    expect(calculateTotalsMock).not.toHaveBeenCalled();
  });

  // Every stop's locationId is parsed non-null too. The panel seats blank stop
  // rows before the user picks anything, and sending one took the preview down
  // with "pulid: invalid length".
  it("never sends a stop without a location", async () => {
    renderPreview(
      baseShipment({
        ...ratable,
        moves: [
          {
            sequence: 0,
            loaded: true,
            stops: [
              { sequence: 0, type: "Pickup", locationId: "loc_01HTEST0000000000000000001" },
              { sequence: 1, type: "Delivery", locationId: "" },
            ],
          },
        ],
      } as unknown as Partial<Shipment>),
    );

    await waitFor(() => expect(calculateTotalsMock).toHaveBeenCalled());

    const payload = calculateTotalsMock.mock.calls[0][0] as Shipment;
    const sentStops = (payload.moves ?? []).flatMap((move) => move.stops ?? []);
    expect(sentStops.every((stop) => !!stop.locationId)).toBe(true);
  });

  // accessorialChargeId and commodityId are parsed the same way. The charge
  // rows were filtered out of the change hash but not out of the payload, so a
  // half-filled accessorial row still reached the server.
  it("never sends a charge or commodity row without its id", async () => {
    renderPreview(
      baseShipment({
        ...ratable,
        moves: [],
        additionalCharges: [
          { accessorialChargeId: "", amount: 0, method: "Flat", unit: 1 },
          {
            accessorialChargeId: "acc_01HTEST0000000000000000001",
            amount: 50,
            method: "Flat",
            unit: 1,
          },
        ],
        commodities: [
          { commodityId: "", pieces: 1, weight: 100 },
          { commodityId: "com_01HTEST0000000000000000001", pieces: 2, weight: 200 },
        ],
      } as unknown as Partial<Shipment>),
    );

    await waitFor(() => expect(calculateTotalsMock).toHaveBeenCalled());

    const payload = calculateTotalsMock.mock.calls[0][0] as Shipment;
    expect(payload.additionalCharges).toHaveLength(1);
    expect(payload.commodities).toHaveLength(1);
    expect((payload.additionalCharges ?? []).every((c) => !!c.accessorialChargeId)).toBe(true);
    expect((payload.commodities ?? []).every((c) => !!c.commodityId)).toBe(true);
  });
});
