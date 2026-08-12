import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import type { LiveTender, LiveTenderOffer } from "@/lib/graphql/tender";
import { TenderLivePanel } from "../tender-live-panel";

function manifest(permissions: Record<string, number>): PermissionManifest {
  return {
    version: "1.0",
    userId: "usr_01",
    organizationId: "org_01",
    activeRoleIds: [],
    authorizedRoleIds: [],
    activeRoles: [],
    authorizedRoles: [],
    requiresRoleActivation: false,
    maxSensitivity: "internal",
    permissions,
    routeAccess: {},
    availableOrgs: [],
    checksum: "abc123",
    expiresAt: 1_782_403_304,
  } as PermissionManifest;
}

function seedPermissions(permissions: Record<string, number>) {
  usePermissionStore.setState({
    manifest: manifest(permissions),
    lastFetched: Date.now(),
    isLoading: false,
  });
}

function offer(overrides: Partial<LiveTenderOffer> = {}): LiveTenderOffer {
  return {
    id: "tof_01",
    tenderId: "tdr_01",
    carrierId: "car_01",
    rank: 1,
    rateMethod: "FlatRate",
    rate: "1500.00",
    offerTtlSeconds: 3600,
    channel: "Email",
    status: "Sent",
    recipientEmail: "dispatch@knight.test",
    sentAt: 1_720_000_000,
    expiresAt: 4_102_444_800,
    respondedAt: null,
    responseSource: null,
    declineReason: "",
    deliveryError: "",
    createdAt: 1_720_000_000,
    updatedAt: 1_720_000_000,
    carrier: { id: "car_01", name: "Knight Logistics", scac: "KNGT" },
    ...overrides,
  } as LiveTenderOffer;
}

function tender(overrides: Partial<LiveTender> = {}): LiveTender {
  return {
    id: "tdr_01",
    shipmentId: "shp_01",
    shipmentMoveId: "smv_01",
    routingGuideId: null,
    mode: "Waterfall",
    status: "Active",
    currentRank: 1,
    cancellationReason: "",
    acceptedOfferId: null,
    acceptedAt: null,
    exhaustedAt: null,
    canceledAt: null,
    createdAt: 1_720_000_000,
    updatedAt: 1_720_000_000,
    routingGuide: null,
    offers: [offer()],
    ...overrides,
  } as LiveTender;
}

function renderPanel(liveTender: LiveTender, onAssignManually?: () => void) {
  return render(
    <TenderLivePanel
      tender={liveTender}
      isTendering={false}
      onCancelTender={vi.fn(async () => undefined)}
      onRecordResponse={vi.fn(async () => undefined)}
      onAssignManually={onAssignManually}
    />,
  );
}

const FULL_TENDER_ACCESS: Record<string, number> = {
  [Resource.Tender]: Operation.Read | Operation.Create | Operation.Update | Operation.Cancel,
  [Resource.ShipmentMove]: Operation.Read | Operation.Assign,
};

describe("TenderLivePanel response provenance", () => {
  beforeEach(() => seedPermissions(FULL_TENDER_ACCESS));
  afterEach(() => {
    cleanup();
    usePermissionStore.getState().clearPermissions();
  });

  it("names the source of a carrier's own accept from the public offer page", () => {
    renderPanel(
      tender({
        offers: [
          offer({ status: "Accepted", respondedAt: 1_720_000_600, responseSource: "Email" }),
        ],
      }),
    );

    expect(screen.getByText("· via email link")).toBeInTheDocument();
  });

  it("distinguishes a dispatcher-recorded response from a carrier-sourced one", () => {
    renderPanel(
      tender({
        offers: [
          offer({ status: "Declined", respondedAt: 1_720_000_600, responseSource: "Manual" }),
        ],
      }),
    );

    expect(screen.getByText("· recorded by dispatcher")).toBeInTheDocument();
  });

  it("names an EDI 990 response", () => {
    renderPanel(
      tender({
        offers: [offer({ status: "Accepted", respondedAt: 1_720_000_600, responseSource: "EDI" })],
      }),
    );

    expect(screen.getByText("· via EDI 990")).toBeInTheDocument();
  });

  it("shows no source chip on an offer that has not been answered", () => {
    renderPanel(tender());

    expect(screen.queryByText(/via email link|via EDI 990|recorded by dispatcher/)).toBeNull();
  });
});

describe("TenderLivePanel permission gating", () => {
  afterEach(() => {
    cleanup();
    usePermissionStore.getState().clearPermissions();
  });

  it("offers the cancel and record-response controls to a user who holds tender rights", () => {
    seedPermissions(FULL_TENDER_ACCESS);
    renderPanel(tender());

    expect(screen.getByRole("button", { name: "Cancel tender" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Record response" })).toBeInTheDocument();
  });

  it("hides the cancel control from a user without tender:cancel", () => {
    seedPermissions({
      [Resource.Tender]: Operation.Read | Operation.Update,
      [Resource.ShipmentMove]: Operation.Read | Operation.Assign,
    });
    renderPanel(tender());

    expect(screen.queryByRole("button", { name: "Cancel tender" })).toBeNull();
    expect(screen.getByRole("button", { name: "Record response" })).toBeInTheDocument();
  });

  it("hides the record-response control from a user without tender:update", () => {
    seedPermissions({
      [Resource.Tender]: Operation.Read | Operation.Cancel,
      [Resource.ShipmentMove]: Operation.Read | Operation.Assign,
    });
    renderPanel(tender());

    expect(screen.queryByRole("button", { name: "Record response" })).toBeNull();
    expect(screen.getByRole("button", { name: "Cancel tender" })).toBeInTheDocument();
  });

  it("shows the manual-assign escape hatch only to a user who can assign a move", () => {
    seedPermissions(FULL_TENDER_ACCESS);
    const { unmount } = renderPanel(tender({ status: "NeedsReview" }), vi.fn());
    expect(screen.getByRole("button", { name: "Assign manually" })).toBeInTheDocument();
    unmount();

    seedPermissions({
      [Resource.Tender]: Operation.Read | Operation.Update | Operation.Cancel,
      [Resource.ShipmentMove]: Operation.Read,
    });
    renderPanel(tender({ status: "NeedsReview" }), vi.fn());
    expect(screen.queryByRole("button", { name: "Assign manually" })).toBeNull();
  });
});
