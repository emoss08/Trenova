import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import type { User } from "@trenova/shared/types/user";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Inspector } from "../inspector";
import type { DispatchActions } from "../use-dispatch-actions";

const tender = {
  id: "tnd_1",
  status: "Accepted",
  mode: "Spot",
  createdAt: 1_720_000_000,
  acceptedAt: null,
  acceptedOfferId: null,
  cancellationReason: null,
  offers: [],
};

vi.mock("@/lib/queries/dispatch-console", () => ({
  dispatchConsoleQueries: {
    moveCandidates: (input: { moveId: string; includeBlocked: boolean }) => ({
      queryKey: ["test-move-candidates", input] as const,
      queryFn: async () => [],
    }),
    shipmentTenders: (shipmentId: string) => ({
      queryKey: ["test-shipment-tenders", shipmentId] as const,
      queryFn: async () => [tender],
    }),
    driverMoves: (input: unknown) => ({
      queryKey: ["test-driver-moves", input] as const,
      queryFn: async () => [],
    }),
  },
}));

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

function signIn(brokerageEnabled: boolean) {
  useAuthStore.setState({
    user: {
      currentOrganizationId: "org_01",
      memberships: [
        {
          userId: "usr_01",
          organizationId: "org_01",
          isDefault: true,
          organization: {
            id: "org_01",
            name: "Asset Carrier Co",
            brokerageEnabled,
            assetOperationsEnabled: true,
          },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });

  // The dispatcher holds every carrier permission: only the capability differs
  // between these cases, which is what makes this a visibility gate and not an
  // authorization one.
  usePermissionStore.setState({
    manifest: manifest({
      [Resource.ShipmentMove]: Operation.Read | Operation.Assign,
      [Resource.Tender]: Operation.Read | Operation.Create,
    }),
    lastFetched: Date.now(),
    isLoading: false,
  });
}

const move = {
  moveId: "smv_1",
  shipmentId: "shp_1",
  proNumber: "PRO-1",
  originCity: "Dallas",
  originState: "TX",
  destinationCity: "Tulsa",
  destinationState: "OK",
  originWindowStart: 1_720_000_000,
  originLocationId: null,
  destinationLocationId: null,
  coverageType: "none",
  isCovered: false,
  carrierAssignmentId: null,
  liveTender: null,
} as unknown as DispatchBoardMove;

const actions = {
  sendSpotTender: vi.fn(),
  startWaterfallTender: vi.fn(),
  cancelTender: vi.fn(),
  recordOfferResponse: vi.fn(),
  isTendering: false,
} as unknown as DispatchActions;

function renderInspector() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <Inspector
          selectedMove={move}
          selectedDriver={null}
          onAssign={vi.fn()}
          onSelectMove={vi.fn()}
          isAssigning={false}
          hotkeysEnabled={false}
          actions={actions}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("Inspector carrier procurement is hidden when the organization does not broker", () => {
  beforeEach(() => {
    usePermissionStore.getState().clearPermissions();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  afterEach(() => {
    cleanup();
    usePermissionStore.getState().clearPermissions();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("offers both carrier coverage paths to a brokerage-enabled organization", async () => {
    signIn(true);

    renderInspector();

    expect(screen.getByRole("button", { name: "Assign to carrier" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tender to carriers" })).toBeInTheDocument();
    expect(await screen.findByText("Tenders")).toBeInTheDocument();
  });

  it("hides the tender and carrier-assign paths from an asset-only organization", () => {
    signIn(false);

    renderInspector();

    expect(screen.queryByRole("button", { name: "Assign to carrier" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Tender to carriers" })).toBeNull();
  });

  it("hides the tender history from an asset-only organization", async () => {
    signIn(false);

    renderInspector();

    await screen.findByRole("button", { name: /ineligible/ });
    expect(screen.queryByText("Tenders")).toBeNull();
  });

  it("keeps driver assignment intact for an asset-only organization", () => {
    signIn(false);

    renderInspector();

    expect(screen.getByText("0 candidates")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Show ineligible" })).toBeInTheDocument();
  });
});
