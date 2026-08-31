import { act, renderHook, waitFor } from "@testing-library/react";
import type { ContractRate, Shipment } from "@trenova/shared/types/shipment";
import type { ReactNode } from "react";
import { FormProvider, useForm, useFormContext } from "react-hook-form";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { useShipmentAutoRate } from "../use-shipment-auto-rate";

const previewContractRateMock = vi.hoisted(() => vi.fn());

vi.mock("@/services/api", () => ({
  apiService: {
    shipmentService: {
      previewContractRate: previewContractRateMock,
    },
  },
}));

vi.mock("@trenova/shared/hooks/use-debounce", () => ({
  useDebounce: <T,>(value: T) => value,
}));

const CONTRACT_TEMPLATE = "fmt_01HTEST0000000000000000001";

function contractRate(overrides: Partial<ContractRate> = {}): ContractRate {
  return {
    applied: true,
    outcome: "Rated",
    agreementId: "ragr_01HTEST0000000000000000001",
    agreementName: "Acme TL 2026",
    ruleId: "ragrr_01HTEST000000000000000001",
    ruleLabel: "Dallas to Chicago",
    formulaTemplateId: CONTRACT_TEMPLATE,
    formulaTemplateName: "Per Mile",
    baseRate: "2.15",
    linehaulAmount: "2666",
    otherChargeAmount: "0",
    totalChargeAmount: "2666",
    previousLinehaulAmount: "0",
    accessorials: [],
    explanation: "Acme TL 2026 priced this lane",
    ...overrides,
  } as ContractRate;
}

/** A shipment the panel would consider ratable: every required id, both ends. */
function ratableShipment(overrides: Partial<Shipment> = {}): Shipment {
  return {
    customerId: "cus_01HTEST0000000000000000001",
    serviceTypeId: "svt_01HTEST0000000000000000001",
    shipmentTypeId: "spt_01HTEST0000000000000000001",
    formulaTemplateId: "fmt_01HTEST0000000000000000009",
    baseRate: 1,
    ratingUnit: 1,
    fuelSurchargeLocked: false,
    additionalCharges: [],
    commodities: [],
    moves: [
      {
        sequence: 0,
        loaded: true,
        stops: [
          { sequence: 0, type: "Pickup", locationId: "loc_01HTEST0000000000000000001" },
          { sequence: 1, type: "Delivery", locationId: "loc_01HTEST0000000000000000002" },
        ],
      },
    ],
    ...overrides,
  } as unknown as Shipment;
}

let formHandle: ReturnType<typeof useForm<Shipment>> | null = null;

function Harness({ defaultValues, children }: { defaultValues: Shipment; children: ReactNode }) {
  const form = useForm<Shipment>({ defaultValues });
  formHandle = form;
  return <FormProvider {...form}>{children}</FormProvider>;
}

function renderAutoRate(defaultValues: Shipment, enabled = true) {
  return renderHook(
    () => {
      const form = useFormContext<Shipment>();
      const autoRate = useShipmentAutoRate({ enabled });
      return { form, autoRate };
    },
    {
      wrapper: ({ children }: { children: ReactNode }) => (
        <Harness defaultValues={defaultValues}>{children}</Harness>
      ),
    },
  );
}

describe("useShipmentAutoRate", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    formHandle = null;
    previewContractRateMock.mockResolvedValue(contractRate());
  });

  // The contract that covers a load is resolved from the customer and both ends
  // of the lane. Asking from half a lane resolves the wrong contract, so nothing
  // is asked until all of it is known.
  it("asks for no contract rate until the shipment can be rated", async () => {
    renderAutoRate(
      ratableShipment({
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

    await act(async () => {
      await Promise.resolve();
    });

    expect(previewContractRateMock).not.toHaveBeenCalled();
  });

  // The state every new shipment is actually in: the rater has named the
  // customer and the lane and nothing else, because the rating method and the
  // base rate are what they are waiting for the contract to tell them. Gating
  // the lookup on a rating method waits for the answer before asking the
  // question, and no shipment created through the panel is ever auto-rated.
  it("rates a new shipment that has no rating method yet", async () => {
    previewContractRateMock.mockResolvedValue(
      contractRate({
        accessorials: [
          {
            accessorialChargeId: "acc_01HTEST0000000000000000001",
            description: "Tarp Fee",
            method: "Flat",
            amount: 75,
            unit: 1,
          },
        ],
        otherChargeAmount: 75,
        totalChargeAmount: 2741,
      }),
    );

    renderAutoRate(
      ratableShipment({
        formulaTemplateId: "",
        baseRate: 0,
        additionalCharges: [],
      } as unknown as Partial<Shipment>),
    );

    await waitFor(() => expect(previewContractRateMock).toHaveBeenCalledTimes(1));

    await waitFor(() => {
      expect(formHandle?.getValues("formulaTemplateId")).toBe(CONTRACT_TEMPLATE);
    });
    expect(formHandle?.getValues("baseRate")).toBe(2.15);
    expect(formHandle?.getValues("additionalCharges")).toEqual([
      {
        accessorialChargeId: "acc_01HTEST0000000000000000001",
        accessorialCharge: {
          description: "Tarp Fee",
        },
        isSystemGenerated: true,
        method: "Flat",
        amount: 75,
        unit: 1,
      },
    ]);
    expect(formHandle?.getValues("autoRated")).toBe(true);
  });

  // What the contract charges is written into the shipment's own fields, where
  // the rater can see it and change it. That is the whole model.
  it("seats the contract's rating method and base rate on the form", async () => {
    renderAutoRate(ratableShipment());

    await waitFor(() => expect(previewContractRateMock).toHaveBeenCalledTimes(1));

    await waitFor(() => {
      expect(formHandle?.getValues("formulaTemplateId")).toBe(CONTRACT_TEMPLATE);
    });
    expect(formHandle?.getValues("baseRate")).toBe(2.15);
    expect(formHandle?.getValues("autoRated")).toBe(true);
    expect(formHandle?.getValues("rateAgreementId")).toBe("ragr_01HTEST0000000000000000001");
  });

  // A rule that binds no rate of its own prices through whatever the shipment
  // already carries, so there is nothing to seat.
  it("keeps the typed base rate when the contract binds none", async () => {
    previewContractRateMock.mockResolvedValue(contractRate({ baseRate: null }));

    renderAutoRate(ratableShipment({ baseRate: 3.5 } as unknown as Partial<Shipment>));

    await waitFor(() => expect(previewContractRateMock).toHaveBeenCalledTimes(1));
    await waitFor(() => {
      expect(formHandle?.getValues("formulaTemplateId")).toBe(CONTRACT_TEMPLATE);
    });

    expect(formHandle?.getValues("baseRate")).toBe(3.5);
  });

  // A contract prices a shipment once. Nothing about a shipment other than its
  // lane changes which agreement covers it, so a second lookup would return the
  // same contract and overwrite whatever the rater typed after the first.
  it("does not re-apply the contract after the rater edits the rate", async () => {
    renderAutoRate(ratableShipment());

    await waitFor(() => expect(previewContractRateMock).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(formHandle?.getValues("baseRate")).toBe(2.15));

    act(() => {
      formHandle?.setValue("baseRate", 1.95, { shouldDirty: true });
    });

    await act(async () => {
      await Promise.resolve();
    });

    expect(previewContractRateMock).toHaveBeenCalledTimes(1);
    expect(formHandle?.getValues("baseRate")).toBe(1.95);
  });

  // An existing shipment already carries a rate somebody may have negotiated.
  it("does nothing on a shipment that already exists", async () => {
    renderAutoRate(ratableShipment(), false);

    await act(async () => {
      await Promise.resolve();
    });

    expect(previewContractRateMock).not.toHaveBeenCalled();
  });

  // An organization that has not written its agreements yet rates every load
  // this way, and interrupting somebody typing a shipment over it is noise.
  it("leaves the form alone when no contract covers the lane", async () => {
    previewContractRateMock.mockResolvedValue(
      contractRate({ applied: false, outcome: "NoRateFound" }),
    );

    renderAutoRate(ratableShipment());

    await waitFor(() => expect(previewContractRateMock).toHaveBeenCalledTimes(1));
    await act(async () => {
      await Promise.resolve();
    });

    expect(formHandle?.getValues("formulaTemplateId")).toBe("fmt_01HTEST0000000000000000009");
    expect(formHandle?.getValues("autoRated")).toBeFalsy();
  });
});
