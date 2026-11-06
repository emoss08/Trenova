import type { DispatchBoardMove, DispatchCandidate } from "@/lib/graphql/dispatch-console";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import { Operation, Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import type { User } from "@trenova/shared/types/user";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { useDispatchConsoleStore } from "@/stores/dispatch-console-store";
import { Inspector } from "../inspector";
import type { DispatchActions } from "../use-dispatch-actions";

const candidatesQueryFn = vi.hoisted(() => vi.fn(async (): Promise<DispatchCandidate[]> => []));

vi.mock("@/lib/queries/dispatch-console", () => ({
  dispatchConsoleQueries: {
    moveCandidates: (input: { moveId: string; includeBlocked: boolean }) => ({
      queryKey: ["test-move-candidates", input] as const,
      queryFn: candidatesQueryFn,
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

// Fields as the candidates query selects them (DispatchCandidate in dispatch_console.graphqls).
function candidate(index: number, workerName: string): DispatchCandidate {
  return {
    workerId: `wrk_${index}`,
    workerName,
    tractorId: `trc_${index}`,
    trailerId: null,
    moveId: "smv_1",
    costCents: 2_800,
    costLines: [{ component: "emptyMiles", cents: 2_800, soft: false }],
    verdict: "feasible",
    blocked: false,
    deadheadMiles: 14,
    originSource: "position",
    loadedMiles: 120,
    estimatedDriveMs: 9_000_000,
    restMs: 0,
    projectedArrival: 1_720_003_600,
    projectedDelivery: 1_720_014_400,
    projectedFreeAt: 1_720_018_000,
    minutesOfSlack: 45,
    slackKnown: true,
    hosKnown: true,
    deliveryLateMinutes: 0,
    driveRemainingMs: 28_800_000,
    shiftRemainingMs: 36_000_000,
    cycleRemainingMs: 180_000_000,
    projectedTimeAvailable: 0,
    hosStrategy: "currentClocks",
    hosRestStartDeadline: 0,
    hosProjectedDriveMs: 19_800_000,
    hosProjectedShiftMs: 27_000_000,
    hosProjectedCycleMs: 171_000_000,
    findings: [],
  } as DispatchCandidate;
}

const hybrid: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: true,
};
const brokerageOnly: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: false,
};

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

function signIn(capabilities: OrganizationCapabilities) {
  useAuthStore.setState({
    user: {
      currentOrganizationId: "org_01",
      memberships: [
        {
          userId: "usr_01",
          organizationId: "org_01",
          isDefault: true,
          organization: { id: "org_01", name: "Brokerage Co", ...capabilities },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });

  // The dispatcher holds every assignment permission there is: only the
  // capability separates these cases.
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
  coverageType: "unassigned",
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
          move={move}
          moveRead={null}
          outsideWindow={false}
          driver={null}
          range={{ start: 0, end: 86_400 }}
          hotkeysEnabled={false}
          queueScope={null}
          onClose={vi.fn()}
          onSelectMove={vi.fn()}
          onNextMove={vi.fn()}
          onCovered={vi.fn()}
          onJumpTo={vi.fn()}
          actions={actions}
        />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("Inspector driver ranking is hidden when the organization owns no trucks", () => {
  beforeEach(() => {
    candidatesQueryFn.mockClear();
    useDispatchConsoleStore.getState().reset();
    usePermissionStore.getState().clearPermissions();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  afterEach(() => {
    cleanup();
    usePermissionStore.getState().clearPermissions();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("ranks drivers for an organization that employs them", async () => {
    signIn(hybrid);

    renderInspector();

    expect(await screen.findByText("No eligible driver")).toBeInTheDocument();
    expect(screen.getAllByRole("button", { name: "Show ineligible" }).length).toBeGreaterThan(0);
  });

  it("counts a single ranked driver in the singular", async () => {
    signIn(hybrid);
    candidatesQueryFn.mockResolvedValueOnce([candidate(1, "Ana Ruiz")]);

    renderInspector();

    expect(await screen.findByText("1 eligible")).toBeInTheDocument();
    expect(screen.getByText("Ana Ruiz")).toBeInTheDocument();
  });

  it("folds the ranked drivers away and back from the heading's button", async () => {
    signIn(hybrid);
    candidatesQueryFn.mockResolvedValueOnce([candidate(1, "Ana Ruiz")]);
    const user = userEvent.setup();

    renderInspector();
    await screen.findByText("Ana Ruiz");

    await user.click(screen.getByRole("button", { name: "Hide the drivers" }));

    const show = screen.getByRole("button", { name: "Show the drivers" });
    expect(show).toHaveAttribute("aria-expanded", "false");
    // Folded, the list is inert: its drivers take no focus and no clicks.
    expect(document.getElementById(show.getAttribute("aria-controls") ?? "")).toHaveAttribute(
      "inert",
    );
    expect(useDispatchConsoleStore.getState().candidatesOpen).toBe(false);

    await user.click(show);

    expect(screen.getByRole("button", { name: "Hide the drivers" })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
  });

  it("opens the rest of the drivers on request and folds them back", async () => {
    signIn(hybrid);
    candidatesQueryFn.mockResolvedValueOnce(
      Array.from({ length: 9 }, (_, index) => candidate(index, `Driver ${index + 1}`)),
    );
    const user = userEvent.setup();

    renderInspector();
    await screen.findByText("Driver 6");

    const more = screen.getByRole("button", { name: "3 more drivers" });
    const rest = () => document.getElementById(more.getAttribute("aria-controls") ?? "");
    expect(more).toHaveAttribute("aria-expanded", "false");
    expect(rest()).toHaveAttribute("inert");
    expect(rest()).toHaveTextContent("Driver 7");
    expect(rest()).not.toHaveTextContent("Driver 6");

    await user.click(more);

    expect(more).toHaveAttribute("aria-expanded", "true");
    expect(rest()).not.toHaveAttribute("inert");

    await user.click(screen.getByRole("button", { name: "Show fewer" }));

    expect(screen.getByRole("button", { name: "3 more drivers" })).toHaveAttribute(
      "aria-expanded",
      "false",
    );
    expect(rest()).toHaveAttribute("inert");
  });

  it("hides the candidate list and its controls from a brokerage", () => {
    signIn(brokerageOnly);

    renderInspector();

    expect(screen.queryByText("No eligible driver")).toBeNull();
    expect(screen.queryByRole("button", { name: "Show ineligible" })).toBeNull();
  });

  it("does not ask the server to rank drivers it will never show", async () => {
    signIn(brokerageOnly);

    renderInspector();

    await screen.findByText("PRO-1");
    expect(candidatesQueryFn).not.toHaveBeenCalled();
  });

  it("keeps the carrier coverage paths a brokerage actually uses", () => {
    signIn(brokerageOnly);

    renderInspector();

    expect(screen.getByRole("button", { name: "Assign to carrier" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Tender to carriers" })).toBeInTheDocument();
  });
});
