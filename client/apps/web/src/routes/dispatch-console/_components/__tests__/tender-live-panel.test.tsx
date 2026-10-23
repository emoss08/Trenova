import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
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
    counterRate: null,
    counterNote: null,
    counteredAt: null,
    counterMarginPercent: null,
    counterNeedsApproval: false,
    counterDecidedAt: null,
    agreedRate: "1500.00",
    revokedAt: null,
    revokeReason: null,
    reofferedFromId: null,
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
    bounceCount: 0,
    lastBouncedAt: null,
    escalatedFromId: null,
    escalationRound: 0,
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

function countered(overrides: Partial<LiveTenderOffer> = {}): LiveTenderOffer {
  return offer({
    status: "Countered",
    counterRate: "1850.00",
    counterNote: "Fuel is up on this lane",
    counteredAt: 1_720_000_600,
    counterMarginPercent: "7.5",
    counterNeedsApproval: true,
    agreedRate: "1850.00",
    ...overrides,
  });
}

describe("TenderLivePanel counter-offers", () => {
  afterEach(() => {
    cleanup();
    usePermissionStore.getState().clearPermissions();
  });

  it("shows the counter's rate, margin, note and that it needs approval", () => {
    seedPermissions({ ...FULL_TENDER_ACCESS, [Resource.ProtectedMargins]: Operation.Read });
    renderPanel(tender({ offers: [countered()] }));

    expect(screen.getByText("Countered at $1,850.00")).toBeInTheDocument();
    expect(screen.getByText("· margin 7.5%")).toBeInTheDocument();
    expect(screen.getByText("“Fuel is up on this lane”")).toBeInTheDocument();
    expect(screen.getByText("Needs approval")).toBeInTheDocument();
  });

  it("draws the counter's margin as withheld for a dispatcher without the margins group", () => {
    seedPermissions(FULL_TENDER_ACCESS);
    renderPanel(tender({ offers: [countered({ counterMarginPercent: null })] }));

    expect(screen.getByText("Countered at $1,850.00")).toBeInTheDocument();
    expect(screen.queryByText("· margin 7.5%")).toBeNull();
    const hidden = screen.getByText("Hidden").closest('[data-slot="locked-reference"]');
    expect(hidden?.getAttribute("aria-description")).toBe("You don't have access to margins");
    expect(screen.getByText("Needs approval")).toBeInTheDocument();
  });

  it("keeps a below-floor counter from a user who cannot approve tenders", () => {
    seedPermissions(FULL_TENDER_ACCESS);
    renderPanel(tender({ offers: [countered()] }));

    expect(screen.queryByRole("button", { name: "Take counter" })).toBeNull();
    expect(
      screen.getByText("Below the margin floor; someone who can approve tenders has to take it."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Refuse counter" })).toBeInTheDocument();
  });

  it("lets a tender approver take a below-floor counter as a counter decision", async () => {
    seedPermissions({
      ...FULL_TENDER_ACCESS,
      [Resource.Tender]: FULL_TENDER_ACCESS[Resource.Tender]! | Operation.Approve,
    });
    const onRecordResponse = vi.fn(async () => undefined);
    render(
      <TenderLivePanel
        tender={tender({ offers: [countered()] })}
        isTendering={false}
        onCancelTender={vi.fn(async () => undefined)}
        onRecordResponse={onRecordResponse}
      />,
    );
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: "Take counter" }));
    await user.click(screen.getByRole("button", { name: "Take counter" }));

    await waitFor(() =>
      expect(onRecordResponse).toHaveBeenCalledWith({
        offerId: "tof_01",
        payload: { action: "Accept", declineReason: "", counterRate: "", counterNote: "" },
      }),
    );
  });

  it("lets anyone with tender:update take a counter above the floor", () => {
    seedPermissions(FULL_TENDER_ACCESS);
    renderPanel(tender({ offers: [countered({ counterNeedsApproval: false })] }));

    expect(screen.getByRole("button", { name: "Take counter" })).toBeInTheDocument();
    expect(screen.queryByText("Needs approval")).toBeNull();
  });

  it("requires a reason to refuse a counter and sends it as a decline", async () => {
    seedPermissions(FULL_TENDER_ACCESS);
    const onRecordResponse = vi.fn(async () => undefined);
    render(
      <TenderLivePanel
        tender={tender({ offers: [countered()] })}
        isTendering={false}
        onCancelTender={vi.fn(async () => undefined)}
        onRecordResponse={onRecordResponse}
      />,
    );
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: "Refuse counter" }));
    await user.click(screen.getByRole("button", { name: "Refuse counter" }));
    expect(await screen.findByText("A decline reason is required")).toBeInTheDocument();
    expect(onRecordResponse).not.toHaveBeenCalled();

    await user.type(screen.getByLabelText(/Reason/), "Above the customer rate");
    await user.click(screen.getByRole("button", { name: "Refuse counter" }));

    await waitFor(() =>
      expect(onRecordResponse).toHaveBeenCalledWith({
        offerId: "tof_01",
        payload: {
          action: "Decline",
          declineReason: "Above the customer rate",
          counterRate: "",
          counterNote: "",
        },
      }),
    );
  });

  it("records a carrier's off-channel counter with its rate", async () => {
    seedPermissions(FULL_TENDER_ACCESS);
    const onRecordResponse = vi.fn(async () => undefined);
    render(
      <TenderLivePanel
        tender={tender()}
        isTendering={false}
        onCancelTender={vi.fn(async () => undefined)}
        onRecordResponse={onRecordResponse}
      />,
    );
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: "Record response" }));
    await user.click(screen.getByRole("button", { name: "Countered" }));
    await user.type(screen.getByLabelText(/Counter rate/), "1850");
    await user.click(screen.getByRole("button", { name: "Record response" }));

    await waitFor(() =>
      expect(onRecordResponse).toHaveBeenCalledWith({
        offerId: "tof_01",
        payload: { action: "Counter", declineReason: "", counterRate: "1850", counterNote: "" },
      }),
    );
  });

  it("shows a refused counter with the refusal reason", () => {
    seedPermissions(FULL_TENDER_ACCESS);
    renderPanel(
      tender({
        offers: [
          countered({ status: "CounterRejected", declineReason: "Above the customer rate" }),
        ],
      }),
    );

    expect(screen.getByText("Counter of $1,850.00 refused")).toBeInTheDocument();
    expect(screen.getByText("“Above the customer rate”")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Take counter" })).toBeNull();
  });
});

describe("TenderLivePanel backouts and escalation", () => {
  beforeEach(() => seedPermissions(FULL_TENDER_ACCESS));
  afterEach(() => {
    cleanup();
    usePermissionStore.getState().clearPermissions();
  });

  it("shows a carrier that backed out with its reason, and the escalation round", () => {
    renderPanel(
      tender({
        bounceCount: 1,
        escalationRound: 2,
        offers: [
          offer({ id: "tof_01", status: "Revoked", revokeReason: "Truck broke down" }),
          offer({
            id: "tof_02",
            rank: 2,
            carrierId: "car_02",
            reofferedFromId: "tof_00",
            carrier: { id: "car_02", name: "Sunset Logistics", scac: "SNST" },
          }),
        ],
      }),
    );

    expect(screen.getByText("Backed out: “Truck broke down”")).toBeInTheDocument();
    expect(screen.getByText("Escalation round 2")).toBeInTheDocument();
    expect(screen.getByText("· a carrier backed out once")).toBeInTheDocument();
    expect(screen.getByText("· asked again")).toBeInTheDocument();
  });

  it("records a backout on a tender in review with the typed reason", async () => {
    const onBounceTender = vi.fn(async () => undefined);
    render(
      <TenderLivePanel
        tender={tender({
          status: "NeedsReview",
          acceptedOfferId: "tof_01",
          offers: [offer({ status: "Accepted" })],
        })}
        isTendering={false}
        onCancelTender={vi.fn(async () => undefined)}
        onRecordResponse={vi.fn(async () => undefined)}
        onBounceTender={onBounceTender}
      />,
    );
    const user = userEvent.setup();

    await user.click(screen.getByRole("button", { name: "Carrier backed out" }));
    await user.type(screen.getByLabelText(/Reason/), "Truck broke down");
    await user.click(screen.getByRole("button", { name: "Record backout" }));

    await waitFor(() =>
      expect(onBounceTender).toHaveBeenCalledWith({
        tenderId: "tdr_01",
        payload: { reason: "Truck broke down" },
      }),
    );
  });

  it("offers no backout without the bounce handler or on an active tender", () => {
    renderPanel(
      tender({
        status: "NeedsReview",
        acceptedOfferId: "tof_01",
        offers: [offer({ status: "Accepted" })],
      }),
    );
    expect(screen.queryByRole("button", { name: "Carrier backed out" })).toBeNull();
  });
});
