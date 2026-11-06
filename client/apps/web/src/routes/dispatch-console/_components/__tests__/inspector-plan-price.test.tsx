import type {
  DispatchBoardMove,
  DispatchCandidate,
  DispatchMovePlanPrice,
} from "@/lib/graphql/dispatch-console";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
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
function candidate(workerId: string, workerName: string, costCents: number): DispatchCandidate {
  return {
    workerId,
    workerName,
    tractorId: `trc_${workerId}`,
    trailerId: null,
    moveId: "smv_1",
    costCents,
    costLines: [{ component: "emptyMiles", cents: costCents, soft: false }],
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
    endMarket: null,
  } as DispatchCandidate;
}

function signIn() {
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
            name: "Fleet Co",
            brokerageEnabled: false,
            assetOperationsEnabled: true,
          },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });
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
      permissions: { [Resource.ShipmentMove]: Operation.Read | Operation.Assign },
      routeAccess: {},
      availableOrgs: [],
      checksum: "abc123",
      expiresAt: 1_782_403_304,
    } as PermissionManifest,
    lastFetched: Date.now(),
    isLoading: false,
  });
}

// Written from DispatchMovePlanPrice in dispatch_console.graphqls.
const plannedPrice: DispatchMovePlanPrice = {
  workerId: "wrk_1",
  workerName: "Ana Ruiz",
  costCents: 14_200,
  preferenceFloorCents: 0,
  coveragePriceCents: 90_000,
  driverValueCents: 3_100,
  uncoveredReason: null,
  alternatives: [{ workerId: "wrk_2", workerName: "Ben Cho", extraCents: 1_200 }],
  generatedAt: 1_800_000_000,
};

function boardMove(overrides: Partial<DispatchBoardMove> = {}): DispatchBoardMove {
  return {
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
    pendingProposal: null,
    failedProposal: null,
    planPrice: null,
    ...overrides,
  } as unknown as DispatchBoardMove;
}

const actions = { isTendering: false, isAssigning: false } as unknown as DispatchActions;

function renderInspector(move: DispatchBoardMove) {
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

function candidateRow(name: string): HTMLElement {
  const drivers = screen.getByRole("region", { name: "Drivers for this move" });
  return within(drivers).getByText(name).closest("[data-glide-target]") as HTMLElement;
}

describe("Inspector plan price", () => {
  beforeEach(() => {
    candidatesQueryFn.mockReset();
    candidatesQueryFn.mockResolvedValue([]);
    useDispatchConsoleStore.getState().reset();
    signIn();
  });

  afterEach(() => {
    cleanup();
    usePermissionStore.getState().clearPermissions();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("shows what the last plan priced an uncovered move at", async () => {
    renderInspector(boardMove({ planPrice: plannedPrice }));

    const panel = await screen.findByRole("region", { name: "Plan price" });
    expect(within(panel).getByText("Worth $900 to cover")).toBeInTheDocument();
    expect(within(panel).getByText("Ana Ruiz is worth $31 to the plan")).toBeInTheDocument();
  });

  it("draws no plan price for a move no plan has priced", async () => {
    renderInspector(boardMove({ planPrice: null }));

    await screen.findByText("PRO-1");
    expect(screen.queryByRole("region", { name: "Plan price" })).toBeNull();
  });

  it("draws no plan price once the move is covered", async () => {
    renderInspector(
      boardMove({
        isCovered: true,
        assignedWorkerName: "Cy Diaz",
        assignedTractorCode: "",
        planPrice: plannedPrice,
      }),
    );

    await screen.findByText("Covered by Cy Diaz");
    expect(screen.queryByRole("region", { name: "Plan price" })).toBeNull();
  });

  it("marks the plan's pick and what choosing each other weighed driver adds", async () => {
    candidatesQueryFn.mockResolvedValue([
      candidate("wrk_1", "Ana Ruiz", 14_200),
      candidate("wrk_2", "Ben Cho", 15_400),
      candidate("wrk_3", "Cy Diaz", 16_000),
    ]);

    renderInspector(boardMove({ planPrice: plannedPrice }));
    await within(screen.getByRole("region", { name: "Drivers for this move" })).findByText(
      "Cy Diaz",
    );

    expect(candidateRow("Ana Ruiz")).toHaveTextContent("Plan's pick");
    expect(candidateRow("Ana Ruiz")).not.toHaveTextContent("to the plan");
    expect(candidateRow("Ben Cho")).toHaveTextContent("+$12 to the plan");
    expect(candidateRow("Ben Cho")).not.toHaveTextContent("Plan's pick");
    expect(candidateRow("Cy Diaz")).not.toHaveTextContent("Plan's pick");
    expect(candidateRow("Cy Diaz")).not.toHaveTextContent("to the plan");
  });

  it("marks no driver when the move has no plan price", async () => {
    candidatesQueryFn.mockResolvedValue([candidate("wrk_1", "Ana Ruiz", 14_200)]);

    renderInspector(boardMove({ planPrice: null }));
    await within(screen.getByRole("region", { name: "Drivers for this move" })).findByText(
      "Ana Ruiz",
    );

    expect(candidateRow("Ana Ruiz")).not.toHaveTextContent("Plan's pick");
    expect(candidateRow("Ana Ruiz")).not.toHaveTextContent("to the plan");
  });

  it("shows each driver's cost to the plan, less what the best driver pays in preferences", async () => {
    candidatesQueryFn.mockResolvedValue([candidate("wrk_1", "Ana Ruiz", 14_200)]);

    renderInspector(boardMove({ planPrice: { ...plannedPrice, preferenceFloorCents: 1_200 } }));
    await within(screen.getByRole("region", { name: "Drivers for this move" })).findByText(
      "Ana Ruiz",
    );

    expect(candidateRow("Ana Ruiz")).toHaveTextContent("$130");
  });

  it("says the hours are unknown and leaves out a margin it cannot measure", async () => {
    candidatesQueryFn.mockResolvedValue([
      { ...candidate("wrk_1", "Ana Ruiz", 14_200), hosKnown: false, slackKnown: false },
    ]);

    renderInspector(boardMove({ planPrice: null }));
    await within(screen.getByRole("region", { name: "Drivers for this move" })).findByText(
      "Ana Ruiz",
    );

    expect(candidateRow("Ana Ruiz")).toHaveTextContent("Hours unknown");
    expect(candidateRow("Ana Ruiz")).not.toHaveTextContent("drive left");
    expect(candidateRow("Ana Ruiz")).not.toHaveTextContent("margin");
  });
});
