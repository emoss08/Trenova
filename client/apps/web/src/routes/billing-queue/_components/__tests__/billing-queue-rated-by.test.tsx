import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { BillingQueueRatedBy } from "../billing-queue-rated-by";

const api = vi.hoisted(() => ({
  shipment: vi.fn(),
  agreement: vi.fn(),
  template: vi.fn(),
}));

vi.mock("@/lib/graphql/billing-queue", () => ({
  getBillingQueueShipmentGraphQL: api.shipment,
  getBillingQueueRateAgreementGraphQL: api.agreement,
  getBillingQueueFormulaTemplateGraphQL: api.template,
}));

afterEach(() => {
  cleanup();
  clearPermissions();
  vi.clearAllMocks();
});

const agreementRatedBy = {
  kind: "Agreement",
  id: "ra_2041",
  code: "RA-2041",
  name: "Acme 2026",
} as const;

// An item whose shipment bills detention and carries a lumper receipt nobody billed.
const item = {
  id: "bqi_1",
  shipmentId: "shp_1",
  shipment: { additionalCharges: [{ id: "ac_det", accessorialChargeId: "acc_det" }] },
  review: {
    issues: [
      {
        id: "bqis_lump",
        code: "UnbilledAccessorial",
        resolutionKey: null,
        options: [
          { key: "add", effect: { kind: "add", accessorialChargeId: "acc_lump" } },
          { key: "skip", effect: { kind: "accept" } },
        ],
      },
    ],
  },
  ratedBy: agreementRatedBy,
} as unknown as BillingQueueItem;

// The rating and the agreement as the BillingQueueShipment and BillingQueueRateAgreement queries select them.
const rating = {
  id: "shp_1",
  ratingDetail: {
    agreementId: "ra_2041",
    agreementName: "Acme 2026",
    ruleId: "rar_1",
    ruleLabel: "TX → TN dry van",
    formulaTemplateName: "Flat",
    ratedAt: 1_793_090_000,
    source: "Rated",
    inputs: {},
  },
};
const agreement = {
  id: "ra_2041",
  code: "RA-2041",
  name: "Acme 2026",
  status: "Active",
  effectiveFrom: 1_767_225_600,
  effectiveTo: 1_798_675_200,
  accessorials: [
    {
      id: "raa_layover",
      accessorialChargeId: "acc_layover",
      chargeCode: "LAY",
      chargeDescription: "Layover",
      method: "PerUnit",
      rateUnit: "Day",
      amount: "250",
      freeUnits: null,
      waived: false,
    },
    {
      id: "raa_lump",
      accessorialChargeId: "acc_lump",
      chargeCode: "LUMP",
      chargeDescription: "Lumper",
      method: "Flat",
      rateUnit: "",
      amount: "185",
      freeUnits: null,
      waived: false,
    },
    {
      id: "raa_det",
      accessorialChargeId: "acc_det",
      chargeCode: "DET",
      chargeDescription: "Detention",
      method: "PerUnit",
      rateUnit: "Hour",
      amount: "75",
      freeUnits: 2,
      waived: false,
    },
  ],
  fuelBinding: { id: "rafb_1", waived: false, programName: "DOE weekly" },
};

function renderRatedBy(ratedBy = item.ratedBy!) {
  render(
    <MemoryRouter>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <BillingQueueRatedBy item={item} ratedBy={ratedBy} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("BillingQueueRatedBy", () => {
  it("reads nothing until the terms are opened, and draws their outline while they load", async () => {
    seedPermissions({ [Resource.RateAgreement]: Operation.Read });
    api.shipment.mockReturnValue(new Promise(() => {}));
    api.agreement.mockReturnValue(new Promise(() => {}));
    const user = userEvent.setup();
    renderRatedBy();

    expect(api.agreement).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "RA-2041 · Acme 2026" }));

    expect(await screen.findByLabelText("Loading the terms")).toHaveAttribute("aria-busy", "true");
    expect(api.agreement).toHaveBeenCalledWith("ra_2041", { signal: expect.any(AbortSignal) });
  });

  it("shows the rule that matched, the contract's accessorials as this load bills them, and its fuel program", async () => {
    seedPermissions({ [Resource.RateAgreement]: Operation.Read });
    api.shipment.mockResolvedValue(rating);
    api.agreement.mockResolvedValue(agreement);
    const user = userEvent.setup();
    renderRatedBy();

    await user.click(screen.getByRole("button", { name: "RA-2041 · Acme 2026" }));
    const card = await screen.findByTestId("billing-queue-rated-by");

    expect(card).toHaveTextContent("Active");
    expect(card).toHaveTextContent("TX → TN dry van");
    const accessorials = within(card)
      .getAllByRole("listitem")
      .map((row) => row.textContent);
    expect(accessorials).toEqual([
      "Detention · $75.00 an hour after 2 free hoursBilled",
      "Lumper · $185.00Suggested",
      "Layover · $250.00 a dayNot billed",
    ]);
    expect(card).toHaveTextContent("DOE weekly");
    expect(within(card).getByRole("link", { name: "Open agreement" })).toHaveAttribute(
      "href",
      expect.stringContaining("ra_2041"),
    );
  });

  it("names a contract the biller cannot read with a lock, and reads none of its terms", () => {
    seedPermissions({});
    renderRatedBy();

    expect(screen.queryByRole("button", { name: "RA-2041 · Acme 2026" })).not.toBeInTheDocument();
    expect(
      screen.getByText("RA-2041 · Acme 2026").closest("[data-slot=locked-reference]"),
    ).not.toBeNull();
    expect(api.agreement).not.toHaveBeenCalled();
  });
});
