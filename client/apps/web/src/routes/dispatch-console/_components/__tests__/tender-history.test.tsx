import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ShipmentTender } from "@/lib/graphql/tender";
import { TenderHistory } from "../tender-history";

const mocks = vi.hoisted(() => ({ shipmentTenders: vi.fn() }));

vi.mock("@/lib/queries/dispatch-console", () => ({
  dispatchConsoleQueries: {
    shipmentTenders: (shipmentId: string) => ({
      queryKey: ["test-shipment-tenders", shipmentId] as const,
      queryFn: () => mocks.shipmentTenders(shipmentId),
    }),
  },
}));

function tender(overrides: Partial<ShipmentTender> = {}): ShipmentTender {
  return {
    id: "tdr_01",
    shipmentId: "shp_01",
    shipmentMoveId: "smv_01",
    routingGuideId: null,
    mode: "Waterfall",
    status: "Accepted",
    currentRank: 1,
    cancellationReason: "",
    acceptedOfferId: "tof_01",
    acceptedAt: 1_720_000_600,
    exhaustedAt: null,
    canceledAt: null,
    createdAt: 1_720_000_000,
    updatedAt: 1_720_000_600,
    routingGuide: null,
    offers: [
      {
        id: "tof_01",
        tenderId: "tdr_01",
        carrierId: "car_01",
        rank: 1,
        rateMethod: "FlatRate",
        rate: "1500.00",
        offerTtlSeconds: 3600,
        channel: "Email",
        status: "Accepted",
        recipientEmail: "dispatch@knight.test",
        sentAt: 1_720_000_000,
        expiresAt: 1_720_003_600,
        respondedAt: 1_720_000_600,
        responseSource: "Manual",
        declineReason: "",
        deliveryError: "",
        createdAt: 1_720_000_000,
        updatedAt: 1_720_000_600,
        carrier: { id: "car_01", name: "Knight Logistics", scac: "KNGT" },
      },
    ],
    ...overrides,
  } as ShipmentTender;
}

function renderHistory() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <TenderHistory shipmentId="shp_01" />
    </QueryClientProvider>,
  );
}

describe("TenderHistory response provenance", () => {
  beforeEach(() => {
    mocks.shipmentTenders.mockReset();
  });

  afterEach(() => cleanup());

  it("names the source of each answered offer once the tender is expanded", async () => {
    mocks.shipmentTenders.mockResolvedValue([tender()]);
    const user = userEvent.setup();

    renderHistory();
    await user.click(await screen.findByRole("button", { expanded: false }));

    expect(await screen.findByText("· recorded by dispatcher")).toBeInTheDocument();
  });

  it("names a carrier's own accept from the public offer page", async () => {
    mocks.shipmentTenders.mockResolvedValue([
      tender({
        offers: [{ ...tender().offers![0], responseSource: "Email" }],
      } as Partial<ShipmentTender>),
    ]);
    const user = userEvent.setup();

    renderHistory();
    await user.click(await screen.findByRole("button", { expanded: false }));

    expect(await screen.findByText("· via email link")).toBeInTheDocument();
  });

  it("shows no source chip on an offer that was never answered", async () => {
    mocks.shipmentTenders.mockResolvedValue([
      tender({
        status: "Canceled",
        acceptedOfferId: null,
        acceptedAt: null,
        offers: [
          {
            ...tender().offers![0],
            status: "Withdrawn",
            respondedAt: null,
            responseSource: null,
          },
        ],
      } as Partial<ShipmentTender>),
    ]);
    const user = userEvent.setup();

    renderHistory();
    await user.click(await screen.findByRole("button", { expanded: false }));

    expect(await screen.findByText("Knight Logistics")).toBeInTheDocument();
    expect(screen.queryByText(/via email link|via EDI 990|recorded by dispatcher/)).toBeNull();
  });
});
