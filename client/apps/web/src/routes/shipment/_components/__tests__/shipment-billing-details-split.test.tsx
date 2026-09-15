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
vi.mock("../why-this-rate", () => ({ WhyThisRate: () => null }));
vi.mock("../auto-rate-dialog", () => ({ AutoRateDialog: () => null }));
vi.mock("../additional-charges/fuel-surcharge-change-dialog", () => ({
  FuelSurchargeChangeDialog: () => null,
}));
vi.mock("@/components/formula-editor/receipt-view", () => ({ ReceiptView: () => null }));

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
  latestForm = form;
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } });
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

  it("shows the invoices a save would produce, one per payer", async () => {
    render(<Harness values={SPLIT_SHIPMENT} />);

    const card = await screen.findByTestId("billing-by-payer");
    expect(card).toHaveTextContent("AMD – AMD");
    expect(card).toHaveTextContent("NV – Nvidia");
    expect(card).toHaveTextContent("$1,000.00");
    expect(card).toHaveTextContent("$200.00");
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

  it("offers to split the freight charge from the charge summary", async () => {
    render(<Harness values={SPLIT_SHIPMENT} />);

    expect(await screen.findByTestId("freight-split-action")).toHaveTextContent("Split");
  });
});
