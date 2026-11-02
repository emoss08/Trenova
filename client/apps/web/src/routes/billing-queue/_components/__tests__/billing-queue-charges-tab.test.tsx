import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ApiRequestError } from "@trenova/shared/lib/api";
import type { BillingIssue, BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { MemoryRouter } from "react-router";
import { clearPermissions, seedPermissions } from "@/test/permissions";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { BillingQueueChargesTab } from "../billing-queue-charges-tab";
import { billingQueue } from "@/lib/queries/billing-queue";
import { useQuery } from "@tanstack/react-query";
import {
  ACME_SHARE_JSON,
  DETENTION_ID,
  LUMPER_ID,
  acmeItemWithLumper,
  acmeItemWithoutDetention,
  queueItem,
} from "./split-shipment-fixture";

const { updateCharges, reassignCharge, resolveIssue, getById } = vi.hoisted(() => ({
  updateCharges: vi.fn(),
  reassignCharge: vi.fn(),
  resolveIssue: vi.fn(),
  getById: vi.fn(),
}));

vi.mock("@/services/api", () => ({
  apiService: { billingQueueService: { updateCharges, reassignCharge, resolveIssue, getById } },
}));
vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));
const { fetchGraphQLSelectOptions } = vi.hoisted(() => ({
  fetchGraphQLSelectOptions: vi.fn(),
}));
vi.mock("@/lib/graphql/select-options", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/select-options")>()),
  fetchGraphQLSelectOptions,
}));
const { getShipmentProfitabilityGraphQL } = vi.hoisted(() => ({
  getShipmentProfitabilityGraphQL: vi.fn(),
}));
vi.mock("@/lib/graphql/shipment", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/shipment")>()),
  getShipmentProfitabilityGraphQL,
}));
vi.mock("../billing-queue-rerate-dialog", () => ({ BillingQueueRerateDialog: () => null }));
vi.mock("../billing-queue-reassign-charge-dialog", () => ({
  BillingQueueReassignChargeDialog: ({
    open,
    line,
  }: {
    open: boolean;
    line: { description: string } | null;
  }) => (open && line ? <p>Reassigning {line.description}</p> : null),
}));

afterEach(cleanup);
beforeEach(() => {
  // A test that fails before using its queued responses must not hand them to the next.
  vi.resetAllMocks();
  fetchGraphQLSelectOptions.mockResolvedValue({ count: 0, results: [] });
  getShipmentProfitabilityGraphQL.mockResolvedValue(null);
});

/**
 * The tab as the detail pane mounts it: reading the item from its query, which already
 * holds `item`. The refetch a save triggers never answers, so whatever the sheet shows
 * after a save came from the save's own response.
 */
function renderFromQuery(item: BillingQueueItem) {
  getById.mockReturnValue(new Promise(() => {}));
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(billingQueue.get(item.id).queryKey, item);
  function FromQuery() {
    const { data } = useQuery(billingQueue.get(item.id));
    return data ? <BillingQueueChargesTab item={data} /> : null;
  }
  return render(
    <MemoryRouter>
      <NuqsTestingAdapter>
        <QueryClientProvider client={client}>
          <FromQuery />
        </QueryClientProvider>
      </NuqsTestingAdapter>
    </MemoryRouter>,
  );
}

/** `shipmentProfitability` as the schema returns it: Decimals arrive as strings. */
function profitability(revenue: string, profit: string, marginPercent: string) {
  return {
    shipmentId: "shp_1",
    loadedMiles: 412,
    deadheadMiles: 38,
    totalMiles: 450,
    revenue,
    estimatedCost: "1800.00",
    profit,
    marginPercent,
    revenuePerLoadedMile: null,
    breakEvenRpm: null,
    missingDistance: false,
    breakdown: [],
    profile: { targetMarginPercent: "15" },
  };
}

function renderTab(item: BillingQueueItem) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter>
      <NuqsTestingAdapter>
        <QueryClientProvider client={client}>
          <BillingQueueChargesTab item={item} />
        </QueryClientProvider>
      </NuqsTestingAdapter>
    </MemoryRouter>,
  );
}

describe("BillingQueueChargesTab payer bill", () => {
  // Acme pays $1,350 of the freight and all of the detention. The tab is
  // Acme's invoice, not the shipment's charge sheet.
  it("shows what the payer owes, not the shipment total", () => {
    renderTab(queueItem({ payer: "acme" }));

    expect(screen.getByTestId("payer-bill-total")).toHaveTextContent("$2,504.38");
    expect(screen.queryByText(/shipment total/)).not.toBeInTheDocument();

    const freight = screen.getByTestId("payer-line-freight");
    expect(freight).toHaveTextContent("$1,350.00");
    expect(freight).toHaveTextContent("of $2,850.00");
    expect(freight).not.toHaveTextContent("Amount split");

    const detention = screen.getByTestId(`payer-line-${DETENTION_ID}`);
    expect(detention).toHaveTextContent("Detention Fee");
    expect(detention).toHaveTextContent("$1,154.38");
    expect(detention).not.toHaveTextContent("of $");

    expect(screen.queryByText(/Billed to other payers/)).not.toBeInTheDocument();
  });

  it("moves charges another payer owes out of the bill", async () => {
    const user = userEvent.setup();
    renderTab(queueItem({ payer: "peak" }));

    expect(screen.getByTestId("payer-bill-total")).toHaveTextContent("$1,500.00");
    expect(screen.getByTestId("payer-line-freight")).toHaveTextContent("$1,500.00");
    expect(screen.queryByTestId(`payer-line-${DETENTION_ID}`)).not.toBeInTheDocument();
    expect(
      within(screen.getByRole("table", { name: "Charge lines" }))
        .getAllByRole("row")
        .map((row) => row.dataset.testid ?? null),
    ).toEqual([null, "payer-line-freight"]);

    await user.click(screen.getByText("Billed to other payers (1)"));
    const other = await screen.findByTestId(`other-payer-line-${DETENTION_ID}`);
    expect(other).toHaveTextContent("Detention Fee");
    expect(other).toHaveTextContent("Acme Manufacturing");
    expect(other).toHaveTextContent("$1,154.38");
  });

  it("warns that charges are shared while they can still be changed", () => {
    renderTab(queueItem({ payer: "peak" }));

    expect(
      screen.getByText(
        "Charges belong to the shipment, so changing one here also changes the other payers' bills.",
      ),
    ).toBeInTheDocument();
  });

  it("drops the shared-charges warning once nothing on the item can change", () => {
    renderTab(queueItem({ payer: "peak", status: "Approved" }));

    expect(screen.queryByText(/Charges belong to the shipment/)).not.toBeInTheDocument();
  });

  it("falls back to the shipment's charges when the split cannot be resolved", () => {
    renderTab(
      queueItem({
        payer: "acme",
        share: {
          ...ACME_SHARE_JSON,
          lines: [],
          payers: [],
          freightAmount: "0",
          accessorialAmount: "0",
          totalAmount: "0",
          resolutionError: "Amount allocations for this charge must add up to 3000.00",
        },
      }),
    );

    expect(
      screen.getByText("Amount allocations for this charge must add up to 3000.00"),
    ).toBeInTheDocument();
    expect(screen.queryByTestId("payer-bill-total")).not.toBeInTheDocument();
    expect(screen.getByText("$4,004.38")).toBeInTheDocument();
  });
});

// A shipment one tariff rated, as `GET /billing-queue/:id/` returns it with no split to
// resolve: two freight lines from the bill, a charge the tariff billed itself with its label,
// quantity and rate, and the tariff's input values the shipment carries.
function tariffRatedItem(): BillingQueueItem {
  const item = queueItem({
    payer: "acme",
    share: {
      ...ACME_SHARE_JSON,
      lines: [],
      payers: [],
      freightAmount: "0",
      accessorialAmount: "0",
      totalAmount: "0",
      resolutionError: "",
    },
  });
  const shipment = item.shipment!;
  return {
    ...item,
    payerShare: null,
    shipment: {
      ...shipment,
      formulaTemplateId: "ft_ltl",
      formulaTemplate: { id: "ft_ltl", name: "LTL class rates" } as never,
      tariffInputs: {
        minimumCharge: { $q: "95", u: "$" },
        fuelPeg: { $q: "1.2", u: "$/gal" },
      },
      freightChargeAmount: 2850,
      freightLines: [
        {
          label: "Linehaul",
          quantity: 12.4,
          quantityUnit: "cwt",
          rate: 210,
          rateUnit: "$/cwt",
          amount: 2604,
        },
        {
          label: "Deficit weight",
          quantity: null,
          quantityUnit: null,
          rate: null,
          rateUnit: null,
          amount: 246,
        },
      ],
      additionalCharges: [
        {
          id: "ac_lift",
          accessorialChargeId: "acc_lift",
          method: "Flat",
          amount: 150,
          unit: 1,
          isSystemGenerated: true,
          isTariff: true,
          label: "Liftgate, 2 stops",
          quantity: 2,
          rate: 75,
          rateUnit: "$",
          accessorialCharge: { id: "acc_lift", code: "LIFT", description: "Liftgate" },
        },
      ] as never,
      otherChargeAmount: 150,
      totalChargeAmount: 3000,
    },
  };
}

describe("BillingQueueChargesTab on a tariff's bill", () => {
  it("lists each freight line of the bill with its quantity and rate", () => {
    renderTab(tariffRatedItem());

    expect(screen.getByText("Linehaul")).toBeInTheDocument();
    expect(screen.getByText("12.4 cwt × $210.00/cwt")).toBeInTheDocument();
    expect(screen.getByText("Deficit weight")).toBeInTheDocument();
    const lineHaul = screen.getByText("Line haul", { selector: "dt" });
    expect(lineHaul.nextElementSibling).toHaveTextContent("$2,850.00");
    expect(screen.queryByText("Base rate")).not.toBeInTheDocument();
  });

  it("names a charge the tariff billed by its label and keeps it out of hand edits", () => {
    renderTab(tariffRatedItem());

    expect(screen.getByText("Liftgate, 2 stops")).toBeInTheDocument();
    expect(screen.getByText("2 × $75.00")).toBeInTheDocument();
    expect(screen.getByText("Liftgate, 2 stops").nextElementSibling).toHaveTextContent("Tariff");
    expect(
      screen.queryByRole("button", { name: "Edit Liftgate, 2 stops" }),
    ).not.toBeInTheDocument();
  });

  it("shows the tariff's terms the shipment rated with, with a way to adjust them", () => {
    renderTab(tariffRatedItem());

    const terms = screen.getByRole("list", { name: "Terms" });
    expect(within(terms).getByText("Minimum charge")).toBeInTheDocument();
    expect(within(terms).getByText("$95.00")).toBeInTheDocument();
    expect(within(terms).getByText("$1.20/gal")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Re-rate" })).toBeEnabled();
  });
});

describe("BillingQueueChargesTab safe edits", () => {
  // Deleting a charge the freight split does not touch still re-resolves every
  // split; an amount split that stops adding up is refused with a named field,
  // and the biller can convert it to percentages and try again.
  it("offers to convert a stale amount split and retries with the flag", async () => {
    const user = userEvent.setup();
    updateCharges
      .mockRejectedValueOnce(
        new ApiRequestError(422, {
          type: "https://api.trenova.app/problems/validation-error",
          title: "Validation failed",
          status: 422,
          errors: [
            {
              field: "convertAmountSplitsToPercent",
              code: "INVALID_OPERATION",
              message:
                "The freight split no longer adds up: the payer amounts total 2850.00 but the charge is now 3000.00. Convert the split to percentages to keep each payer's proportion, or change the split on the shipment.",
            },
          ],
        }),
      )
      .mockResolvedValueOnce(queueItem({ payer: "acme" }));
    renderTab(queueItem({ payer: "acme" }));

    await user.click(screen.getByRole("button", { name: "Edit Detention Fee" }));
    await user.click(screen.getByRole("button", { name: "Remove line" }));

    const dialog = await screen.findByRole("alertdialog");
    expect(within(dialog).getByText(/The freight split no longer adds up/)).toBeInTheDocument();
    expect(updateCharges).toHaveBeenCalledTimes(1);
    const firstPayload = updateCharges.mock.calls[0][1];
    expect(firstPayload.convertAmountSplitsToPercent).toBeUndefined();

    await user.click(within(dialog).getByRole("button", { name: "Convert to percentages" }));

    await waitFor(() => expect(updateCharges).toHaveBeenCalledTimes(2));
    expect(updateCharges.mock.calls[1]).toEqual([
      "bqi_acme",
      { ...firstPayload, convertAmountSplitsToPercent: true },
    ]);
  });

  it("does not offer charge-payer changes once the item is approved", () => {
    renderTab(queueItem({ payer: "acme", status: "Approved" }));

    expect(screen.queryByRole("button", { name: /Change payer/ })).not.toBeInTheDocument();
  });

  it("opens the reassignment for a charge another payer owes", async () => {
    const user = userEvent.setup();
    renderTab(queueItem({ payer: "peak", status: "ReadyForReview" }));

    await user.click(screen.getByText("Billed to other payers (1)"));
    await user.click(await screen.findByRole("button", { name: "Change payer for Detention Fee" }));

    expect(await screen.findByText("Reassigning Detention Fee")).toBeInTheDocument();
  });
});

describe("BillingQueueChargesTab editing a line in place", () => {
  it("says nothing changed until the rate does, then what the change does to the line", async () => {
    const user = userEvent.setup();
    renderTab(queueItem({ payer: "acme" }));

    await user.click(screen.getByRole("button", { name: "Edit Detention Fee" }));

    expect(screen.getByText("No change yet")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Save line" })).toBeDisabled();

    const rate = screen.getByRole("textbox", { name: "Rate" });
    await user.clear(rate);
    await user.type(rate, "1200");

    expect(
      screen.getByText("+$45.62 against the saved line; the invoice total updates when you save"),
    ).toBeInTheDocument();

    await user.clear(rate);
    await user.type(rate, "1000");
    expect(
      screen.getByText("−$154.38 against the saved line; the invoice total updates when you save"),
    ).toBeInTheDocument();
  });

  it("saves the line with its charge, quantity and rate", async () => {
    const user = userEvent.setup();
    updateCharges.mockResolvedValueOnce(queueItem({ payer: "acme" }));
    renderTab(queueItem({ payer: "acme" }));

    await user.click(screen.getByRole("button", { name: "Edit Detention Fee" }));
    const quantity = screen.getByRole("textbox", { name: "Quantity" });
    await user.clear(quantity);
    await user.type(quantity, "2");
    await user.click(screen.getByRole("button", { name: "Save line" }));

    await waitFor(() => expect(updateCharges).toHaveBeenCalledTimes(1));
    expect(updateCharges.mock.calls[0]).toEqual([
      "bqi_acme",
      {
        additionalCharges: [
          {
            id: DETENTION_ID,
            accessorialChargeId: "acc_det",
            method: "Flat",
            amount: 1154.38,
            unit: 2,
          },
        ],
        expandShipmentDetails: true,
      },
    ]);
  });

  it("saves a quantity past three digits, up to what a charge's unit column holds", async () => {
    const user = userEvent.setup();
    updateCharges.mockResolvedValueOnce(queueItem({ payer: "acme" }));
    renderTab(queueItem({ payer: "acme" }));

    await user.click(screen.getByRole("button", { name: "Edit Detention Fee" }));
    const quantity = screen.getByRole("textbox", { name: "Quantity" });
    await user.clear(quantity);
    await user.type(quantity, "32768");
    expect(quantity).toHaveValue("3276");

    await user.clear(quantity);
    await user.type(quantity, "32767");
    expect(quantity).toHaveValue("32767");
    await user.click(screen.getByRole("button", { name: "Save line" }));

    await waitFor(() => expect(updateCharges).toHaveBeenCalledTimes(1));
    expect(updateCharges.mock.calls[0]![1].additionalCharges[0].unit).toBe(32_767);
  });

  it("saves a rate well past $100,000", async () => {
    const user = userEvent.setup();
    updateCharges.mockResolvedValueOnce(queueItem({ payer: "acme" }));
    renderTab(queueItem({ payer: "acme" }));

    await user.click(screen.getByRole("button", { name: "Edit Detention Fee" }));
    const rate = screen.getByRole("textbox", { name: "Rate" });
    await user.clear(rate);
    await user.type(rate, "250000");
    await user.click(screen.getByRole("button", { name: "Save line" }));

    await waitFor(() => expect(updateCharges).toHaveBeenCalledTimes(1));
    expect(updateCharges.mock.calls[0]![1].additionalCharges[0].amount).toBe(250_000);
  });

  it("refuses a quantity under one without saving", async () => {
    const user = userEvent.setup();
    renderTab(queueItem({ payer: "acme" }));

    await user.click(screen.getByRole("button", { name: "Edit Detention Fee" }));
    const quantity = screen.getByRole("textbox", { name: "Quantity" });
    await user.clear(quantity);
    await user.type(quantity, "0");
    await user.click(screen.getByRole("button", { name: "Save line" }));

    expect(await screen.findByText("Must be at least 1")).toBeInTheDocument();
    expect(updateCharges).not.toHaveBeenCalled();
  });

  it("opens the split between payers from the row, on a shipment with more than one", async () => {
    const user = userEvent.setup();
    renderTab(queueItem({ payer: "acme" }));

    await user.click(screen.getByRole("button", { name: "Edit Detention Fee" }));
    const editor = screen.getByRole("button", { name: "Save line" }).closest("form")!;
    await user.click(within(editor).getByRole("button", { name: "Split between payers" }));

    expect(await screen.findByText("Reassigning Detention Fee")).toBeInTheDocument();
  });

  it("moves the whole line to the payer picked, after saving nothing else", async () => {
    const user = userEvent.setup();
    fetchGraphQLSelectOptions.mockResolvedValue({
      count: 2,
      results: [
        { id: "cus_acme", label: "Acme Foods", description: null, meta: null },
        { id: "cus_peak", label: "Peak Logistics", description: null, meta: null },
      ],
    });
    reassignCharge.mockResolvedValueOnce({
      item: queueItem({ payer: "acme" }),
      items: [],
      createdItemIds: [],
      canceledItemIds: [],
    });
    renderTab(queueItem({ payer: "acme" }));

    await user.click(screen.getByRole("button", { name: "Edit Detention Fee" }));
    const [, payer] = screen.getAllByRole("combobox");
    await user.click(payer!);
    await user.click(await screen.findByRole("option", { name: /Peak Logistics/u }));

    expect(screen.getByText("Moves the line to Peak Logistics when you save")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Save line" }));

    await waitFor(() => expect(reassignCharge).toHaveBeenCalledTimes(1));
    expect(updateCharges).not.toHaveBeenCalled();
    expect(reassignCharge.mock.calls[0]).toEqual([
      "bqi_acme",
      {
        chargeKind: "Accessorial",
        additionalChargeId: DETENTION_ID,
        allocations: [
          { billToCustomerId: "cus_peak", method: "Percent", percent: "100", amount: null },
        ],
      },
    ]);
  });
});

describe("BillingQueueChargesTab removing a line", () => {
  it("takes the line off the sheet from the save's answer, without waiting for a refetch", async () => {
    const user = userEvent.setup();
    updateCharges.mockResolvedValueOnce(acmeItemWithoutDetention());
    renderFromQuery(queueItem({ payer: "acme" }));

    await user.click(screen.getByRole("button", { name: "Edit Detention Fee" }));
    await user.click(screen.getByRole("button", { name: "Remove line" }));

    await waitFor(() =>
      expect(screen.queryByTestId(`payer-line-${DETENTION_ID}`)).not.toBeInTheDocument(),
    );
    expect(screen.queryByRole("button", { name: "Save line" })).not.toBeInTheDocument();
    expect(screen.getByTestId("payer-bill-total")).toHaveTextContent("$1,350.00");
    expect(updateCharges.mock.calls[0]).toEqual([
      "bqi_acme",
      { additionalCharges: [], expandShipmentDetails: true },
    ]);
  });

  it("refreshes the shipment's margin once the charges change", async () => {
    const user = userEvent.setup();
    getShipmentProfitabilityGraphQL
      .mockResolvedValueOnce(profitability("4004.38", "2204.38", "55.05"))
      .mockResolvedValue(profitability("2850.00", "1050.00", "36.84"));
    updateCharges.mockResolvedValueOnce(queueItem({ payer: "acme" }));
    renderTab(queueItem({ payer: "acme" }));

    expect(await screen.findByText("$2,204.38 · 55%")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Edit Detention Fee" }));
    await user.click(screen.getByRole("button", { name: "Remove line" }));

    expect(await screen.findByText("$1,050.00 · 37%")).toBeInTheDocument();
    expect(screen.queryByText("$2,204.38 · 55%")).not.toBeInTheDocument();
    expect(getShipmentProfitabilityGraphQL).toHaveBeenCalledTimes(2);
  });
});

describe("BillingQueueChargesTab adding a line in place", () => {
  const accessorials = {
    count: 1,
    results: [
      {
        id: "acc_lump",
        label: "LUMP",
        description: "Lumper",
        meta: { method: "Flat", amount: 185 },
      },
    ],
  };

  it("adds a line under the charges, priced from the accessorial chosen", async () => {
    const user = userEvent.setup();
    fetchGraphQLSelectOptions.mockResolvedValue(accessorials);
    updateCharges.mockResolvedValueOnce(queueItem({ payer: "acme" }));
    renderTab(queueItem({ payer: "acme" }));

    await user.click(screen.getByRole("button", { name: "Add charge" }));
    const row = screen.getByRole("row", { name: "New charge line" });
    expect(within(row).getByText("Choose the accessorial this line bills")).toBeInTheDocument();
    expect(within(row).getByRole("button", { name: "Add line" })).toBeEnabled();

    const [accessorial] = within(row).getAllByRole("combobox");
    await user.click(accessorial!);
    await user.click(await screen.findByRole("option", { name: /LUMP/u }));

    expect(within(row).getByText("Adds $185.00 to the bill when you add it")).toBeInTheDocument();
    await user.click(within(row).getByRole("button", { name: "Add line" }));

    await waitFor(() => expect(updateCharges).toHaveBeenCalledTimes(1));
    expect(updateCharges.mock.calls[0]![1].additionalCharges.at(-1)).toEqual({
      accessorialChargeId: "acc_lump",
      method: "Flat",
      unit: 1,
      amount: 185,
    });
    expect(reassignCharge).not.toHaveBeenCalled();
  });

  it("moves a line added from another payer's bill to that payer once it is added", async () => {
    const user = userEvent.setup();
    fetchGraphQLSelectOptions.mockResolvedValue(accessorials);
    const peak = queueItem({ payer: "peak" });
    const lumper = {
      id: "ac_lump",
      accessorialChargeId: "acc_lump",
      method: "Flat",
      amount: 185,
      unit: 1,
      isSystemGenerated: false,
      accessorialCharge: { id: "acc_lump", code: "LUMP", description: "Lumper" },
    };
    updateCharges.mockResolvedValueOnce({
      ...peak,
      shipment: {
        ...peak.shipment!,
        additionalCharges: [...(peak.shipment!.additionalCharges ?? []), lumper],
      },
    });
    reassignCharge.mockResolvedValueOnce({
      item: peak,
      items: [],
      createdItemIds: [],
      canceledItemIds: [],
    });
    renderTab(peak);

    await user.click(screen.getByRole("button", { name: "Add charge" }));
    const row = screen.getByRole("row", { name: "New charge line" });
    const [accessorial] = within(row).getAllByRole("combobox");
    await user.click(accessorial!);
    await user.click(await screen.findByRole("option", { name: /LUMP/u }));
    await user.click(within(row).getByRole("button", { name: "Add line" }));

    await waitFor(() => expect(reassignCharge).toHaveBeenCalledTimes(1));
    expect(reassignCharge.mock.calls[0]).toEqual([
      "bqi_peak",
      {
        chargeKind: "Accessorial",
        additionalChargeId: "ac_lump",
        allocations: [
          { billToCustomerId: "cus_peak", method: "Percent", percent: "100", amount: null },
        ],
      },
    ]);
  });

  it("puts the added line on the sheet from the save's answer and closes its editor", async () => {
    const user = userEvent.setup();
    fetchGraphQLSelectOptions.mockResolvedValue(accessorials);
    updateCharges.mockResolvedValueOnce(acmeItemWithLumper());
    renderFromQuery(queueItem({ payer: "acme" }));

    await user.click(screen.getByRole("button", { name: "Add charge" }));
    const row = screen.getByRole("row", { name: "New charge line" });
    const [accessorial] = within(row).getAllByRole("combobox");
    await user.click(accessorial!);
    await user.click(await screen.findByRole("option", { name: /LUMP/u }));
    await user.click(within(row).getByRole("button", { name: "Add line" }));

    expect(await screen.findByTestId(`payer-line-${LUMPER_ID}`)).toHaveTextContent("$185.00");
    await waitFor(() =>
      expect(screen.queryByRole("row", { name: "New charge line" })).not.toBeInTheDocument(),
    );
    expect(screen.getByTestId("payer-bill-total")).toHaveTextContent("$2,689.38");
    expect(reassignCharge).not.toHaveBeenCalled();
  });

  it("closes the new line once the retry that converts a stale split adds it", async () => {
    const user = userEvent.setup();
    fetchGraphQLSelectOptions.mockResolvedValue(accessorials);
    updateCharges
      .mockRejectedValueOnce(
        new ApiRequestError(422, {
          type: "https://api.trenova.app/problems/validation-error",
          title: "Validation failed",
          status: 422,
          errors: [
            {
              field: "convertAmountSplitsToPercent",
              code: "INVALID_OPERATION",
              message:
                "The freight split no longer adds up: the payer amounts total 2850.00 but the charge is now 3000.00. Convert the split to percentages to keep each payer's proportion, or change the split on the shipment.",
            },
          ],
        }),
      )
      .mockResolvedValueOnce(acmeItemWithLumper());
    renderFromQuery(queueItem({ payer: "acme" }));

    await user.click(screen.getByRole("button", { name: "Add charge" }));
    const row = screen.getByRole("row", { name: "New charge line" });
    const [accessorial] = within(row).getAllByRole("combobox");
    await user.click(accessorial!);
    await user.click(await screen.findByRole("option", { name: /LUMP/u }));
    await user.click(within(row).getByRole("button", { name: "Add line" }));
    const dialog = await screen.findByRole("alertdialog");
    await user.click(within(dialog).getByRole("button", { name: "Convert to percentages" }));

    expect(await screen.findByTestId(`payer-line-${LUMPER_ID}`)).toBeInTheDocument();
    await waitFor(() =>
      expect(screen.queryByRole("row", { name: "New charge line" })).not.toBeInTheDocument(),
    );
    expect(updateCharges.mock.calls[1]![1]).toMatchObject({
      convertAmountSplitsToPercent: true,
      expandShipmentDetails: true,
    });
  });

  it("refuses a line with no accessorial or rate", async () => {
    const user = userEvent.setup();
    renderTab(queueItem({ payer: "peak" }));

    await user.click(screen.getByRole("button", { name: "Add charge" }));
    await user.click(screen.getByRole("button", { name: "Add line" }));

    expect(await screen.findAllByText("Required")).not.toHaveLength(0);
    expect(updateCharges).not.toHaveBeenCalled();
  });

  it("discards the new line", async () => {
    const user = userEvent.setup();
    renderTab(queueItem({ payer: "peak" }));

    await user.click(screen.getByRole("button", { name: "Add charge" }));
    await user.click(screen.getByRole("button", { name: "Discard new line" }));

    expect(screen.queryByRole("row", { name: "New charge line" })).not.toBeInTheDocument();
  });
});

// A receipt the shipment carries that no charge bills, as the review raises it: Nova's
// when Nova read the receipt, a rule's when a person entered it.
function unbilled(
  id: string,
  source: BillingIssue["source"],
  label: string,
  amount: string,
  reasoning: string,
): BillingIssue {
  return {
    id,
    itemId: "bqi_acme",
    checkKey: "charges",
    code: "UnbilledAccessorial",
    subjectKey: `doc_${id}`,
    summary: `${label} $${amount} isn't billed`,
    reasoning,
    source,
    flaggedChargeId: null,
    options: [
      {
        key: "add",
        label: `Bill it · $${amount}`,
        amount: Number(amount),
        done: `${label} billed`,
        effect: {
          kind: "add",
          accessorialChargeId: `acc_${id}`,
          label,
          amount: Number(amount),
          basis: "From the receipt",
        },
      },
      {
        key: "skip",
        label: "Not billable",
        amount: null,
        done: `${label} marked not billable`,
        effect: { kind: "accept" },
      },
    ],
    resolutionKey: null,
    resolutionText: null,
    resolvedById: null,
    resolvedAt: null,
    requestedAt: null,
    undoable: false,
    createdAt: 1_800_000_000,
  } as unknown as BillingIssue;
}

describe("BillingQueueChargesTab charges the shipment does not bill", () => {
  const lumper = unbilled(
    "lump",
    "Agent",
    "Lumper",
    "185.00",
    "Receipt LMP-5521 for $185.00 is on the shipment, but no lumper charge bills it.",
  );
  const scale = unbilled(
    "scale",
    "Deterministic",
    "Scale ticket",
    "12.00",
    "A receipt for $12.00 is on the shipment, but no scale ticket charge bills it.",
  );
  const withUnbilled = (): BillingQueueItem =>
    ({
      ...queueItem({ payer: "acme" }),
      review: {
        checks: [],
        issues: [lumper, scale],
        needsCount: 0,
        ready: true,
        blocker: "",
        documents: [],
        billerName: "",
      },
    }) as unknown as BillingQueueItem;

  it("names each one with who found it, why, and the line that would bill it", () => {
    renderTab(withUnbilled());

    expect(screen.getByRole("img", { name: "Nova found this" })).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "A rule found this" })).toBeInTheDocument();
    expect(screen.getByText("Lumper").parentElement).toHaveTextContent(
      "Lumper $185.00 isn't billed",
    );
    expect(screen.getByText("$185.00")).toHaveClass("font-mono");
    expect(
      screen.getByText(
        "Receipt LMP-5521 for $185.00 is on the shipment, but no lumper charge bills it.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add $185.00" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Add $12.00" })).toBeEnabled();
    expect(screen.getAllByRole("button", { name: "Not billable" })).toHaveLength(2);
  });

  it("bills the line, or marks it not billable, through the issue it settles", async () => {
    const user = userEvent.setup();
    resolveIssue.mockResolvedValue(queueItem({ payer: "acme" }));
    renderTab(withUnbilled());

    await user.click(screen.getByRole("button", { name: "Add $185.00" }));
    await waitFor(() => expect(resolveIssue).toHaveBeenCalledWith("bqi_acme", "lump", "add"));

    await user.click(screen.getAllByRole("button", { name: "Not billable" })[1]!);
    await waitFor(() => expect(resolveIssue).toHaveBeenCalledWith("bqi_acme", "scale", "skip"));
  });
});

describe("BillingQueueChargesTab rated by", () => {
  beforeEach(() => {
    seedPermissions({
      [Resource.RateAgreement]: Operation.Read,
      [Resource.FormulaTemplate]: Operation.Read,
    });
  });
  afterEach(clearPermissions);

  it("names the rate agreement that priced the shipment, opening its terms in place", () => {
    renderTab({
      ...queueItem({ payer: "acme" }),
      ratedBy: { kind: "Agreement", id: "ra_2041", code: "RA-2041", name: "Acme 2026" },
    });

    const trigger = screen.getByRole("button", { name: "RA-2041 · Acme 2026" });
    expect(trigger.closest("span")).toHaveTextContent("Rated by RA-2041 · Acme 2026");
    expect(screen.queryByRole("link", { name: "RA-2041 · Acme 2026" })).not.toBeInTheDocument();
  });

  it("names the formula template when no contract priced it", () => {
    renderTab({
      ...queueItem({ payer: "acme" }),
      ratedBy: { kind: "Template", id: "ft_permile", code: null, name: "Per mile" },
    });

    expect(screen.getByRole("button", { name: "Per mile" })).toBeInTheDocument();
  });

  it("names a contract the biller cannot read without opening its terms", () => {
    seedPermissions({ [Resource.FormulaTemplate]: Operation.Read });
    renderTab({
      ...queueItem({ payer: "acme" }),
      ratedBy: { kind: "Agreement", id: "ra_2041", code: "RA-2041", name: "Acme 2026" },
    });

    expect(screen.queryByRole("button", { name: "RA-2041 · Acme 2026" })).not.toBeInTheDocument();
    expect(
      screen.getByText("RA-2041 · Acme 2026").closest("[data-slot=locked-reference]"),
    ).not.toBeNull();
  });

  it("names nothing when nothing priced the shipment", () => {
    renderTab(queueItem({ payer: "acme" }));

    expect(screen.queryByText(/Rated by/u)).not.toBeInTheDocument();
  });
});
