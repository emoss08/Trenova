import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import type { DispatchBoardMove } from "@/lib/graphql/dispatch-console";
import { Inspector } from "../inspector";
import type { DispatchActions } from "../use-dispatch-actions";

vi.mock("@/lib/queries/dispatch-console", () => ({
  dispatchConsoleQueries: {
    moveCandidates: (input: { moveId: string; includeBlocked: boolean }) => ({
      queryKey: ["test-move-candidates", input] as const,
      queryFn: async () => [],
    }),
    shipmentTenders: (shipmentId: string) => ({
      queryKey: ["test-shipment-tenders", shipmentId] as const,
      queryFn: async () => [],
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

function seedPermissions(permissions: Record<string, number>) {
  usePermissionStore.setState({
    manifest: manifest(permissions),
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

describe("Inspector coverage actions are gated on the backend's own permissions", () => {
  beforeEach(() => {
    usePermissionStore.getState().clearPermissions();
  });

  afterEach(() => {
    cleanup();
    usePermissionStore.getState().clearPermissions();
  });

  it("offers both coverage paths to a dispatcher who can assign moves and create tenders", () => {
    seedPermissions({
      [Resource.ShipmentMove]: Operation.Read | Operation.Assign,
      [Resource.Tender]: Operation.Read | Operation.Create,
    });

    renderInspector();

    expect(screen.getByRole("button", { name: "Assign to carrier" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tender to carriers" })).toBeInTheDocument();
  });

  it("hides the carrier-assign path from a user who can only read moves", () => {
    seedPermissions({
      [Resource.ShipmentMove]: Operation.Read,
      [Resource.Tender]: Operation.Read | Operation.Create,
    });

    renderInspector();

    expect(screen.queryByRole("button", { name: "Assign to carrier" })).toBeNull();
    expect(screen.getByRole("button", { name: "Tender to carriers" })).toBeInTheDocument();
  });

  it("hides the tender path from a user without tender:create", () => {
    seedPermissions({
      [Resource.ShipmentMove]: Operation.Read | Operation.Assign,
      [Resource.Tender]: Operation.Read,
    });

    renderInspector();

    expect(screen.queryByRole("button", { name: "Tender to carriers" })).toBeNull();
    expect(screen.getByRole("button", { name: "Assign to carrier" })).toBeInTheDocument();
  });

  it("hides both from a user holding only shipment_move:read", () => {
    seedPermissions({ [Resource.ShipmentMove]: Operation.Read });

    renderInspector();

    expect(screen.queryByRole("button", { name: "Assign to carrier" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Tender to carriers" })).toBeNull();
  });
});
