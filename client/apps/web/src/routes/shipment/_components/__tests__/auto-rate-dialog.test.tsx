import { useState } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {
  contractRateSchema,
  type ContractRate,
  type Shipment,
} from "@trenova/shared/types/shipment";
import { FormProvider, useForm } from "react-hook-form";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AutoRateDialog } from "../auto-rate-dialog";

const { autoRate, toastError } = vi.hoisted(() => ({ autoRate: vi.fn(), toastError: vi.fn() }));

vi.mock("@/services/api", () => ({
  apiService: { shipmentService: { autoRate } },
}));
vi.mock("@trenova/shared/components/ui/toast", () => ({
  toast: { error: toastError, success: vi.fn() },
}));

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

// The application as `POST /shipments/:id/auto-rate` returns it: the freight lines and the
// charges the contract's tariff billed, with decimals as Go marshals them.
const APPLIED: ContractRate = contractRateSchema.parse({
  applied: true,
  outcome: "Rated",
  agreementId: "ragr_1",
  agreementName: "Acme 2026",
  ruleId: "rarl_1",
  ruleLabel: "Dallas to Chicago",
  formulaTemplateId: "ft_dry_van",
  formulaTemplateName: "Dry van TL",
  templateVersion: 4,
  inputs: { rate: { $q: "2.35", u: "$/mi" } },
  freightLines: [
    {
      label: "Linehaul",
      quantity: "812",
      quantityUnit: "mi",
      rate: "2.35",
      rateUnit: "$/mi",
      amount: "1908.2",
    },
  ],
  freightAmount: "1908.2",
  otherChargeAmount: "125",
  totalChargeAmount: "2033.2",
  charges: [
    {
      accessorialChargeId: "acc_xdrop",
      code: "XDROP",
      description: "Extra drop",
      owner: "Tariff",
      label: "Extra drops",
      amount: "100",
      quantity: "2",
      rate: "50",
      rateUnit: "$/stop",
    },
    {
      accessorialChargeId: "acc_tarp",
      code: "TARP",
      description: "Tarp fee",
      owner: "RateAgreement",
      amount: "25",
      quantity: null,
      rate: null,
    },
  ],
  previousFreightAmount: "1800",
  explanation: "",
});

function shipmentAfter(): Shipment {
  return {
    id: "shp_1",
    formulaTemplateId: "ft_dry_van",
    tariffInputs: APPLIED.inputs,
    freightLines: APPLIED.freightLines,
    additionalCharges: [],
    ratingDetail: {
      bill: {
        currency: "CAD",
        lines: [],
        freightTotal: "1908.2",
        accessorialTotal: "125",
        total: "2033.2",
      },
    },
  } as unknown as Shipment;
}

function Harness() {
  const form = useForm<Shipment>({ defaultValues: { id: "shp_1" } as Shipment });
  const [client] = useState(() => new QueryClient());
  return (
    <QueryClientProvider client={client}>
      <FormProvider {...form}>
        <AutoRateDialog />
      </FormProvider>
    </QueryClientProvider>
  );
}

async function reapply() {
  const user = userEvent.setup();
  render(<Harness />);
  await user.click(screen.getByRole("button", { name: "Re-apply rate" }));
  await user.click(screen.getByRole("button", { name: "Re-rate" }));
}

describe("AutoRateDialog", () => {
  it("shows what the contract applied line by line, in the rating currency", async () => {
    autoRate.mockResolvedValue({ shipment: shipmentAfter(), contractRate: APPLIED });

    await reapply();

    const dialog = await screen.findByRole("dialog", { name: "Rate applied" });
    expect(
      within(dialog).getByText("Acme 2026 — Dallas to Chicago priced this shipment."),
    ).toBeTruthy();
    expect(within(dialog).getByText("Dry van TL")).toBeTruthy();
    expect(within(dialog).getByText("Linehaul")).toBeTruthy();
    expect(within(dialog).getByText("812 mi × CA$2.35/mi")).toBeTruthy();
    expect(within(dialog).getByText("Extra drops")).toBeTruthy();
    expect(within(dialog).getByText("XDROP")).toBeTruthy();
    expect(within(dialog).getByText("2 × CA$50.00/stop")).toBeTruthy();
    expect(within(dialog).getByText("Tarp fee")).toBeTruthy();
    expect(within(dialog).getByText("CA$2,033.20")).toBeTruthy();
    expect(within(dialog).getByText("+CA$108.20")).toBeTruthy();
  });

  it("changes nothing and says so when no contract covers the lane", async () => {
    autoRate.mockResolvedValue({
      shipment: shipmentAfter(),
      contractRate: { ...APPLIED, applied: false, outcome: "NoRateFound" },
    });

    await reapply();

    await vi.waitFor(() =>
      expect(toastError).toHaveBeenCalledWith("No contract covers this lane", expect.anything()),
    );
    expect(screen.queryByRole("dialog", { name: "Rate applied" })).toBeNull();
  });
});
