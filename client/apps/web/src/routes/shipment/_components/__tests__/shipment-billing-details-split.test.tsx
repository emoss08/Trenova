import { useEffect, useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { Shipment } from "@trenova/shared/types/shipment";
import { MemoryRouter } from "react-router";
import {
  FormProvider,
  useController,
  useForm,
  type Control,
  type FieldValues,
  type UseFormReturn,
} from "react-hook-form";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import ShipmentBillingDetails from "../shipment-billing-details";

const { getBillingProfile, getUIPolicy } = vi.hoisted(() => ({
  getBillingProfile: vi.fn(),
  getUIPolicy: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: {
    shipmentService: { getUIPolicy },
    customerService: { getBillingProfile },
  },
}));

const CUSTOMERS = [
  { id: "cus_intel", label: "Intel", description: null, meta: { code: "INTEL" } },
  { id: "cus_amd", label: "AMD", description: null, meta: { code: "AMD" } },
  { id: "cus_nvidia", label: "Nvidia", description: null, meta: { code: "NV" } },
];

function CustomerPickerStub({
  control,
  name,
  label,
  onOptionChange,
}: {
  control: Control<FieldValues>;
  name: string;
  label?: string;
  onOptionChange?: (option: (typeof CUSTOMERS)[number] | null) => void;
}) {
  const { field } = useController({ control, name });
  return (
    <select
      aria-label={label ?? name}
      value={field.value ?? ""}
      onChange={(event) => {
        const option = CUSTOMERS.find((c) => c.id === event.target.value) ?? null;
        field.onChange(option?.id ?? null);
        onOptionChange?.(option);
      }}
    >
      <option value="">none</option>
      {CUSTOMERS.map((c) => (
        <option key={c.id} value={c.id}>
          {c.label}
        </option>
      ))}
    </select>
  );
}

vi.mock("@/components/autocomplete-fields", () => ({
  CustomerAutocompleteField: CustomerPickerStub,
  FormulaTemplateAutocompleteField: () => null,
  OrderAutocompleteField: () => null,
}));
vi.mock("@/hooks/use-shipment-auto-rate", () => ({
  useShipmentAutoRate: () => ({ appliedRate: null, dismissAppliedRate: vi.fn() }),
}));
vi.mock("@/hooks/use-shipment-totals-preview", () => ({
  useShipmentTotalsPreview: () => ({
    isCalculating: false,
    error: null,
    fuelSurchargeChange: null,
    resolveFuelSurchargeChange: vi.fn(),
  }),
}));
vi.mock("../profitability/profitability-summary", () => ({ ProfitabilitySummary: () => null }));
vi.mock("../previous-rates-dialog", () => ({ PreviousRatesButton: () => null }));
vi.mock("../auto-rate-dialog", () => ({ AutoRateDialog: () => null }));
vi.mock("../additional-charges/fuel-surcharge-change-dialog", () => ({
  FuelSurchargeChangeDialog: () => null,
}));
vi.mock("../shipment-tariff-inputs", () => ({ ShipmentTariffInputs: () => null }));

afterEach(cleanup);

beforeEach(() => {
  vi.clearAllMocks();
  getUIPolicy.mockResolvedValue(undefined);
  getBillingProfile.mockImplementation(async (customerId: string) => ({
    customerId,
    creditStatus: "Hold",
    creditHoldReason: `${customerId} is on hold`,
  }));
});

let latestForm: UseFormReturn<Shipment> | null = null;

function Harness({ values }: { values: Partial<Shipment> }) {
  const form = useForm<Shipment>({ defaultValues: values as Shipment });
  useEffect(() => {
    latestForm = form;
  }, [form]);
  const [client] = useState(
    () => new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } }),
  );
  return (
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <FormProvider {...form}>
          <ShipmentBillingDetails />
        </FormProvider>
      </MemoryRouter>
    </QueryClientProvider>
  );
}

const SPLIT_SHIPMENT: Partial<Shipment> = {
  customerId: "cus_intel",
  customer: { id: "cus_intel", name: "Intel", code: "INTEL" } as Shipment["customer"],
  billToCustomerId: "cus_amd",
  billToCustomer: { id: "cus_amd", name: "AMD", code: "AMD" } as Shipment["billToCustomer"],
  freightTerms: "ThirdParty",
  freightChargeAmount: 1000,
  otherChargeAmount: 200,
  totalChargeAmount: 1200,
  freightAllocations: [],
  additionalCharges: [
    {
      accessorialChargeId: "acc_1",
      method: "Flat",
      amount: 200,
      unit: 1,
      isSystemGenerated: false,
      allocations: [
        {
          billToCustomerId: "cus_nvidia",
          method: "Percent",
          percent: 100,
          billToCustomer: { id: "cus_nvidia", name: "Nvidia", code: "NV" },
        },
      ],
    },
  ] as Shipment["additionalCharges"],
};

describe("ShipmentBillingDetails split billing", () => {
  // Three customers will each receive an invoice, and a hold on any of them
  // blocks that invoice, so every payer is checked — not only the customer
  // who ordered the shipment.
  it("raises a credit alert for every payer on the shipment", async () => {
    render(<Harness values={SPLIT_SHIPMENT} />);

    await waitFor(() => {
      expect(screen.getByTestId("credit-alert-cus_amd")).toBeInTheDocument();
      expect(screen.getByTestId("credit-alert-cus_nvidia")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("credit-alert-cus_intel")).not.toBeInTheDocument();
    expect(getBillingProfile).toHaveBeenCalledWith("cus_amd");
    expect(getBillingProfile).toHaveBeenCalledWith("cus_nvidia");
    expect(getBillingProfile).not.toHaveBeenCalledWith("cus_intel");
  });

  it("says an automatic hold stops new shipments but not billing", async () => {
    getBillingProfile.mockImplementation(async (customerId: string) => ({
      customerId,
      creditStatus: customerId === "cus_amd" ? "Hold" : "Active",
      enforceCreditLimit: true,
      creditHoldAutomatic: true,
      creditHoldReason:
        "Open receivables of 15000.00 USD are over the credit limit of 10000.00 USD",
    }));
    render(<Harness values={SPLIT_SHIPMENT} />);

    const alert = await screen.findByTestId("credit-alert-cus_amd");
    expect(alert).toHaveTextContent(
      "Open receivables of 15000.00 USD are over the credit limit of 10000.00 USD. No new shipments can be booked for them until the balance is paid back under the limit. Shipments already booked are still billed.",
    );
  });

  it("says a hold a person placed also holds the payer's invoices", async () => {
    getBillingProfile.mockImplementation(async (customerId: string) => ({
      customerId,
      creditStatus: customerId === "cus_amd" ? "Suspended" : "Active",
      enforceCreditLimit: true,
      creditHoldAutomatic: false,
      creditHoldReason: "Bankruptcy filing",
    }));
    render(<Harness values={SPLIT_SHIPMENT} />);

    const alert = await screen.findByTestId("credit-alert-cus_amd");
    expect(alert).toHaveTextContent(
      "Bankruptcy filing. No new shipments can be booked for them, and their invoices wait until accounts receivable lifts the hold.",
    );
  });

  it("shows the invoices a save would produce, one per payer", async () => {
    render(<Harness values={SPLIT_SHIPMENT} />);

    const card = await screen.findByTestId("billing-by-payer");
    expect(card).toHaveTextContent("AMD – AMD");
    expect(card).toHaveTextContent("NV – Nvidia");
    expect(card).toHaveTextContent("$1,000.00");
    expect(card).toHaveTextContent("$200.00");
  });

  // The server rounds every share but the last half to even and gives the last the rest, so a
  // 60/40 split of $1,000.01 bills $600.01 and $400.00 (testdata/allocation.json). The old
  // float split rounded the other way on half cents.
  it("divides the freight to the cent the way the server bills it", async () => {
    render(
      <Harness
        values={{
          ...SPLIT_SHIPMENT,
          billToCustomerId: "cus_intel",
          billToCustomer: SPLIT_SHIPMENT.customer as Shipment["billToCustomer"],
          freightChargeAmount: 1000.01,
          additionalCharges: [],
          freightAllocations: [
            {
              billToCustomerId: "cus_amd",
              method: "Percent",
              percent: 40,
              sequence: 1,
              billToCustomer: { id: "cus_amd", name: "AMD", code: "AMD" },
            },
            { billToCustomerId: "cus_intel", method: "Percent", percent: 60, sequence: 0 },
          ] as Shipment["chargeAllocations"],
        }}
      />,
    );

    const card = await screen.findByTestId("billing-by-payer");
    const rows = card.querySelectorAll(".divide-y > div");
    expect([...rows].map((row) => row.textContent)).toEqual([
      "INTEL – IntelPrimaryFreight $600.01Accessorials $0.00$600.01",
      "AMD – AMDFreight $400.00Accessorials $0.00$400.00",
    ]);
  });

  it("waits for a split that mixes percents and amounts before naming what anyone owes", async () => {
    render(
      <Harness
        values={{
          ...SPLIT_SHIPMENT,
          freightAllocations: [
            { billToCustomerId: "cus_intel", method: "Percent", percent: 60, sequence: 0 },
            { billToCustomerId: "cus_amd", method: "Amount", amount: 400, sequence: 1 },
          ] as Shipment["chargeAllocations"],
        }}
      />,
    );

    const card = await screen.findByTestId("billing-by-payer");
    expect(card).toHaveTextContent(
      "Billing by payerEach payer's share shows once every split adds up.",
    );
  });

  // A bill-to was chosen for Intel's shipment. Once the shipment belongs to
  // someone else that choice, and every split naming a payer, is stale.
  it("clears the bill-to and every split when the customer changes", async () => {
    const user = userEvent.setup();
    render(<Harness values={SPLIT_SHIPMENT} />);

    await user.selectOptions(screen.getByRole("combobox", { name: "Customer" }), "cus_nvidia");

    await waitFor(() => {
      expect(latestForm?.getValues("billToCustomerId")).toBeNull();
    });
    expect(latestForm?.getValues("freightAllocations")).toEqual([]);
    expect(latestForm?.getValues("additionalCharges.0.allocations")).toEqual([]);
  });

  it("keeps the bill-to when the shipment is first given a customer", async () => {
    const user = userEvent.setup();
    render(
      <Harness
        values={{
          customerId: "",
          billToCustomerId: "cus_amd",
          freightAllocations: [],
          additionalCharges: [],
        }}
      />,
    );

    await user.selectOptions(screen.getByRole("combobox", { name: "Customer" }), "cus_intel");

    await waitFor(() => {
      expect(latestForm?.getValues("customerId")).toBe("cus_intel");
    });
    expect(latestForm?.getValues("billToCustomerId")).toBe("cus_amd");
  });

  // Who pays the freight is chosen on the freight row itself, the same way as
  // for every accessorial; sharing it is one step further in.
  it("names the freight's payer on the charge summary and opens the split from there", async () => {
    const user = userEvent.setup();
    render(<Harness values={SPLIT_SHIPMENT} />);

    const control = await screen.findByTestId("charge-payer-control");
    expect(control).toHaveTextContent("Bill to: Same as shipment");

    await user.click(control);
    await user.click(await screen.findByRole("button", { name: "Split this charge…" }));

    expect(await screen.findByText("Split freight charge")).toBeInTheDocument();
  });
});
