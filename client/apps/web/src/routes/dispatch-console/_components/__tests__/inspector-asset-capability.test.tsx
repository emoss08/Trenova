import type { DispatchBoardMove, DispatchCandidate } from "@/lib/graphql/dispatch-console";
import { stubLayout } from "@/test/layout";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import { Operation, Resource, type PermissionManifest } from "@trenova/shared/types/permission";
import type { User } from "@trenova/shared/types/user";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
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
    score: 82,
    verdict: "feasible",
    blocked: false,
    deadheadMiles: 14,
    estimatedDriveMs: 9_000_000,
    projectedArrival: 1_720_003_600,
    minutesOfSlack: 45,
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
    factors: [],
  } as DispatchCandidate;
}

// The virtual list's scroll element: row wrapper, then the sized body, then the list
// wrapper, then the viewport that scrolls.
function scrollerOf(rowText: string): HTMLElement {
  const row = screen.getByText(rowText).closest<HTMLElement>("[data-index]");
  const scroller = row?.parentElement?.parentElement?.parentElement;
  if (!scroller) throw new Error(`no scroll element above ${rowText}`);
  return scroller;
}

function scrollTo(scroller: HTMLElement, top: number) {
  act(() => {
    scroller.scrollTop = top;
    scroller.dispatchEvent(new Event("scroll"));
  });
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

// The candidate list is virtualized, and jsdom lays nothing out: without a size no row
// would mount.
let restoreLayout = () => {};

describe("Inspector driver ranking is hidden when the organization owns no trucks", () => {
  beforeEach(() => {
    restoreLayout = stubLayout(320);
    candidatesQueryFn.mockClear();
    usePermissionStore.getState().clearPermissions();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  afterEach(() => {
    restoreLayout();
    cleanup();
    usePermissionStore.getState().clearPermissions();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("ranks drivers for an organization that employs them", async () => {
    signIn(hybrid);

    renderInspector();

    expect(await screen.findByText("0 candidates")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Show ineligible" })).toBeInTheDocument();
    expect(await screen.findByText(/No eligible driver for this move/)).toBeInTheDocument();
  });

  it("counts a single ranked driver in the singular", async () => {
    signIn(hybrid);
    candidatesQueryFn.mockResolvedValueOnce([candidate(1, "Ana Ruiz")]);

    renderInspector();

    expect(await screen.findByText("1 candidate")).toBeInTheDocument();
    expect(screen.getByText("Ana Ruiz")).toBeInTheDocument();
  });

  it("keeps a candidate's score detail open after it scrolls out of view and back", async () => {
    signIn(hybrid);
    candidatesQueryFn.mockResolvedValueOnce(
      Array.from({ length: 30 }, (_, index) =>
        candidate(index, `Driver ${String(index).padStart(2, "0")}`),
      ),
    );
    const user = userEvent.setup();

    renderInspector();

    await screen.findByText("Driver 00");
    await user.click(screen.getAllByRole("button", { name: "Why this score" })[0]);
    expect(screen.getAllByRole("button", { name: "Hide detail" })).toHaveLength(1);

    const scroller = scrollerOf("Driver 00");
    scrollTo(scroller, 25 * 320);
    await waitFor(() => expect(screen.queryByText("Driver 00")).toBeNull());

    scrollTo(scroller, 0);
    await screen.findByText("Driver 00");
    expect(screen.getAllByRole("button", { name: "Hide detail" })).toHaveLength(1);
  });

  it("hides the candidate list and its controls from a brokerage", () => {
    signIn(brokerageOnly);

    renderInspector();

    expect(screen.queryByText("0 candidates")).toBeNull();
    expect(screen.queryByRole("button", { name: "Show ineligible" })).toBeNull();
    expect(screen.queryByText(/No eligible driver for this move/)).toBeNull();
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
