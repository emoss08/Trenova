import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ShipmentTender } from "@/lib/graphql/tender";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource, type PermissionManifest } from "@trenova/shared/types/permission";
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
    bounceCount: 0,
    escalationRound: 0,
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
        counterRate: null,
        counterNote: null,
        agreedRate: "1500.00",
        revokeReason: null,
        reofferedFromId: null,
        createdAt: 1_720_000_000,
        updatedAt: 1_720_000_600,
        carrier: { id: "car_01", name: "Knight Logistics", scac: "KNGT" },
      },
    ],
    ...overrides,
  } as ShipmentTender;
}

function renderHistory(onBounceTender?: Parameters<typeof TenderHistory>[0]["onBounceTender"]) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <TenderHistory shipmentId="shp_01" onBounceTender={onBounceTender} />
    </QueryClientProvider>,
  );
}

function seedTenderPermissions(operations: number) {
  usePermissionStore.setState({
    manifest: {
      version: "1.0",
      userId: "usr_01",
      organizationId: "org_01",
      activeRoleIds: [],
      authorizedRoleIds: [],
      activeRoles: [],
      authorizedRoles: [],
      requiresRoleActivation: false,
      maxSensitivity: "internal",
      permissions: { [Resource.Tender]: operations },
      routeAccess: {},
      availableOrgs: [],
      checksum: "abc123",
      expiresAt: 1_782_403_304,
    } as PermissionManifest,
    lastFetched: Date.now(),
    isLoading: false,
  });
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

describe("TenderHistory backouts and counters", () => {
  beforeEach(() => {
    mocks.shipmentTenders.mockReset();
  });

  afterEach(() => {
    cleanup();
    usePermissionStore.getState().clearPermissions();
  });

  it("records a backout on an accepted tender with the typed reason", async () => {
    seedTenderPermissions(Operation.Read | Operation.Update);
    mocks.shipmentTenders.mockResolvedValue([tender()]);
    const onBounceTender = vi.fn(async () => undefined);
    const user = userEvent.setup();

    renderHistory(onBounceTender);
    await user.click(await screen.findByRole("button", { name: "Carrier backed out" }));
    await user.type(screen.getByLabelText(/Reason/), "Truck broke down");
    await user.click(screen.getByRole("button", { name: "Record backout" }));

    await waitFor(() =>
      expect(onBounceTender).toHaveBeenCalledWith({
        tenderId: "tdr_01",
        payload: { reason: "Truck broke down" },
      }),
    );
  });

  it("offers no backout to a user without tender:update", async () => {
    seedTenderPermissions(Operation.Read);
    mocks.shipmentTenders.mockResolvedValue([tender()]);

    renderHistory(vi.fn(async () => undefined));

    expect(await screen.findByText("Knight Logistics")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Carrier backed out" })).toBeNull();
  });

  it("offers no backout once the acceptance has been walked back", async () => {
    seedTenderPermissions(Operation.Read | Operation.Update);
    mocks.shipmentTenders.mockResolvedValue([
      tender({
        status: "Active",
        offers: [{ ...tender().offers![0], status: "Revoked", revokeReason: "Truck broke down" }],
      } as Partial<ShipmentTender>),
    ]);

    renderHistory(vi.fn(async () => undefined));

    await screen.findByRole("button", { expanded: false });
    expect(screen.queryByRole("button", { name: "Carrier backed out" })).toBeNull();
  });

  it("shows the agreed rate of a tender taken on a counter-offer", async () => {
    seedTenderPermissions(Operation.Read);
    mocks.shipmentTenders.mockResolvedValue([
      tender({
        bounceCount: 2,
        offers: [{ ...tender().offers![0], counterRate: "1850.00", agreedRate: "1850.00" }],
      } as Partial<ShipmentTender>),
    ]);

    renderHistory();

    expect(await screen.findByText(/at \$1,850\.00 after a counter-offer/)).toBeInTheDocument();
    expect(screen.getByText("carriers backed out 2 times")).toBeInTheDocument();
  });

  it("says nothing about a counter on an acceptance at the offered rate", async () => {
    seedTenderPermissions(Operation.Read);
    mocks.shipmentTenders.mockResolvedValue([tender()]);

    renderHistory();

    expect(await screen.findByText("Knight Logistics")).toBeInTheDocument();
    expect(screen.queryByText(/after a counter-offer/)).toBeNull();
  });
});
