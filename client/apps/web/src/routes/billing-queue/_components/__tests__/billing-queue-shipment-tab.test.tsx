import { clearPermissions, seedPermissions } from "@/test/permissions";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import BillingQueueShipmentTab from "../billing-queue-shipment-tab";

const { getShipment } = vi.hoisted(() => ({ getShipment: vi.fn() }));

vi.mock("@/lib/graphql/billing-queue", () => ({
  getBillingQueueShipmentGraphQL: getShipment,
  getBillingQueueRateAgreementGraphQL: vi.fn(),
  getBillingQueueFormulaTemplateGraphQL: vi.fn(),
}));

afterEach(() => {
  cleanup();
  clearPermissions();
  vi.clearAllMocks();
});

const item = {
  id: "bqi_1",
  shipmentId: "shp_1",
  poNumber: "PO-55821",
  ratedBy: { kind: "Agreement", id: "ra_2041", code: "RA-2041", name: "Acme 2026" },
} as unknown as BillingQueueItem;

// The shipment as the BillingQueueShipment query selects it.
const shipment = {
  id: "shp_1",
  proNumber: "PRO-24118",
  bol: "BOL-778120",
  externalReference: "ACM-77310",
  pieces: 26,
  weight: 38200,
  temperatureMin: null,
  temperatureMax: null,
  customer: { id: "cus_acme", name: "Acme Foods" },
  commodities: [{ id: "shc_1", commodity: { id: "com_1", name: "Packaged foods" } }],
  moves: [
    {
      id: "smv_1",
      sequence: 0,
      loaded: true,
      distance: 452,
      assignment: {
        id: "asg_1",
        tractor: { id: "trc_1", code: "T-2241" },
        trailer: { id: "trl_1", code: "TR-5310" },
        primaryWorker: { id: "wrk_1", wholeName: "Marcus Lee" },
      },
      stops: [
        {
          id: "stp_p",
          type: "Pickup",
          sequence: 0,
          scheduledWindowStart: 1_793_100_000,
          scheduledWindowEnd: 1_793_107_200,
          actualArrival: 1_793_102_400,
          actualDeparture: 1_793_107_500,
          addressLine: "",
          location: {
            id: "loc_1",
            name: "Acme Dallas DC",
            city: "Dallas",
            state: { id: "st_tx", abbreviation: "TX" },
          },
        },
        {
          id: "stp_d",
          type: "Delivery",
          sequence: 1,
          scheduledWindowStart: 1_793_190_000,
          scheduledWindowEnd: null,
          actualArrival: 1_793_190_000,
          actualDeparture: 1_793_205_000,
          addressLine: "",
          location: {
            id: "loc_2",
            name: "Acme Memphis DC",
            city: "Memphis",
            state: { id: "st_tn", abbreviation: "TN" },
          },
        },
      ],
    },
  ],
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

function renderTab() {
  render(
    <MemoryRouter>
      <QueryClientProvider
        client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}
      >
        <BillingQueueShipmentTab item={item} />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

describe("BillingQueueShipmentTab", () => {
  it("draws its own outline while the shipment loads", () => {
    seedPermissions({ [Resource.Shipment]: Operation.Read });
    getShipment.mockReturnValue(new Promise(() => {}));
    renderTab();

    expect(screen.getByLabelText("Loading the shipment")).toHaveAttribute("aria-busy", "true");
    expect(getShipment).toHaveBeenCalledWith("shp_1", { signal: expect.any(AbortSignal) });
  });

  it("lists each stop with its time on site and the load's facts", async () => {
    seedPermissions({ [Resource.Shipment]: Operation.Read, [Resource.Customer]: Operation.Read });
    getShipment.mockResolvedValue(shipment);
    renderTab();

    const stops = await screen.findAllByTestId("billing-queue-stop");
    expect(stops).toHaveLength(2);
    expect(stops[0]).toHaveTextContent("Acme Dallas DC");
    expect(stops[0]).toHaveTextContent("Pickup · Dallas, TX");
    expect(stops[0]).toHaveTextContent("1h 25m");
    expect(stops[1]).toHaveTextContent("4h 10m");

    const load = screen.getByTestId("billing-queue-load");
    expect(load).toHaveTextContent("BOL-778120");
    expect(load).toHaveTextContent("PO-55821");
    expect(load).toHaveTextContent("Marcus Lee");
    expect(load).toHaveTextContent("T-2241 / TR-5310");
    expect(load).toHaveTextContent("Packaged foods");
    expect(load).toHaveTextContent("452 loaded · 0 empty");
    expect(within(load).getByRole("link", { name: "Acme Foods" })).toBeInTheDocument();

    expect(screen.getByText("Rated by RA-2041 · Acme 2026")).toBeInTheDocument();
    expect(screen.getByText(/Rule TX → TN dry van/u)).toBeInTheDocument();
  });

  it("reads nothing for someone who cannot read shipments, and says so", () => {
    seedPermissions({});
    renderTab();

    expect(screen.getByTestId("billing-queue-shipment-locked")).toHaveTextContent(
      "You don't have access to shipments",
    );
    expect(getShipment).not.toHaveBeenCalled();
  });
});
