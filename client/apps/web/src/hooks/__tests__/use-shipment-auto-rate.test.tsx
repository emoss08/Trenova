import { act, renderHook, waitFor } from "@testing-library/react";
import type { ContractRate, Shipment } from "@trenova/shared/types/shipment";
import { useEffect, type ReactNode } from "react";
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
const CONTRACT_INPUTS = { rate: { $q: "2.15", u: "$/mi" }, minimumCharge: { $q: "450", u: "$" } };
const TYPED_INPUTS = { rate: { $q: "1.95", u: "$/mi" } };
const CONTRACT_FREIGHT = [
  {
    label: "Linehaul",
    quantity: 1240,
    quantityUnit: "mi",
    rate: 2.15,
    rateUnit: "$/mi",
    amount: 2666,
  },
];

function contractRate(overrides: Partial<ContractRate> = {}): ContractRate {
  return {
    applied: true,
    outcome: "Rated",
    agreementId: "ragr_01HTEST0000000000000000001",
    agreementName: "Acme TL 2026",
    ruleId: "ragrr_01HTEST000000000000000001",
    ruleLabel: "Dallas to chicago",
    formulaTemplateId: CONTRACT_TEMPLATE,
    formulaTemplateName: "Per mile",
    inputs: CONTRACT_INPUTS,
    refusal: "",
    freightLines: CONTRACT_FREIGHT,
    freightAmount: 2666,
    otherChargeAmount: 0,
    totalChargeAmount: 2666,
    previousFreightAmount: 0,
    charges: [],
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
    tariffInputs: {},
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
  useEffect(() => {
    formHandle = form;
  });
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
        charges: [
          {
            accessorialChargeId: "acc_01HTEST0000000000000000002",
            code: "XDROP",
            description: "Extra drop",
            owner: "Tariff",
            label: "Extra drops",
            amount: 50,
            quantity: 2,
            rate: 25,
            rateUnit: "$/stop",
          },
          {
            accessorialChargeId: "acc_01HTEST0000000000000000001",
            code: "TARP",
            description: "Tarp Fee",
            owner: "RateAgreement",
            label: "",
            amount: 25,
            quantity: null,
            rate: null,
            rateUnit: null,
          },
        ],
        otherChargeAmount: 75,
        totalChargeAmount: 2741,
      }),
    );

    renderAutoRate(
      ratableShipment({
        formulaTemplateId: "",
        tariffInputs: {},
        additionalCharges: [],
      } as unknown as Partial<Shipment>),
    );

    await waitFor(() => expect(previewContractRateMock).toHaveBeenCalledTimes(1));

    await waitFor(() => {
      expect(formHandle?.getValues("formulaTemplateId")).toBe(CONTRACT_TEMPLATE);
    });
    expect(formHandle?.getValues("tariffInputs")).toEqual(CONTRACT_INPUTS);
    expect(formHandle?.getValues("freightLines")).toEqual(CONTRACT_FREIGHT);
    expect(formHandle?.getValues("additionalCharges")).toEqual([
      {
        accessorialChargeId: "acc_01HTEST0000000000000000002",
        accessorialCharge: { code: "XDROP", description: "Extra drop" },
        isSystemGenerated: true,
        isTariff: true,
        label: "Extra drops",
        quantity: 2,
        rate: 25,
        rateUnit: "$/stop",
        method: "Flat",
        amount: 50,
        unit: 1,
        allocations: [],
      },
      {
        accessorialChargeId: "acc_01HTEST0000000000000000001",
        accessorialCharge: { code: "TARP", description: "Tarp Fee" },
        isSystemGenerated: true,
        isTariff: false,
        label: null,
        quantity: null,
        rate: null,
        rateUnit: null,
        method: "Flat",
        amount: 25,
        unit: 1,
        allocations: [],
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
    expect(formHandle?.getValues("tariffInputs")).toEqual(CONTRACT_INPUTS);
    expect(formHandle?.getValues("autoRated")).toBe(true);
    expect(formHandle?.getValues("rateAgreementId")).toBe("ragr_01HTEST0000000000000000001");
  });

  // Adopting a contract writes every input value it resolved, rule over
  // agreement over default, so a re-rate reapplies the whole contract and not
  // whatever the shipment happened to carry before.
  it("replaces the shipment's input values with the contract's resolved ones", async () => {
    previewContractRateMock.mockResolvedValue(contractRate());

    renderAutoRate(
      ratableShipment({
        tariffInputs: { rate: { $q: "3.50", u: "$/mi" }, extraDropFee: { $q: "75", u: "$" } },
      } as unknown as Partial<Shipment>),
    );

    await waitFor(() => expect(previewContractRateMock).toHaveBeenCalledTimes(1));
    await waitFor(() => {
      expect(formHandle?.getValues("formulaTemplateId")).toBe(CONTRACT_TEMPLATE);
    });

    expect(formHandle?.getValues("tariffInputs")).toEqual(CONTRACT_INPUTS);
  });

  // A contract prices a shipment once. Nothing about a shipment other than its
  // lane changes which agreement covers it, so a second lookup would return the
  // same contract and overwrite whatever the rater typed after the first.
  it("does not re-apply the contract after the rater edits the rate", async () => {
    renderAutoRate(ratableShipment());

    await waitFor(() => expect(previewContractRateMock).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(formHandle?.getValues("tariffInputs")).toEqual(CONTRACT_INPUTS));

    act(() => {
      formHandle?.setValue("tariffInputs", TYPED_INPUTS, { shouldDirty: true });
    });

    await act(async () => {
      await Promise.resolve();
    });

    expect(previewContractRateMock).toHaveBeenCalledTimes(1);
    expect(formHandle?.getValues("tariffInputs")).toEqual(TYPED_INPUTS);
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
