import type { DispatchBoardSummary } from "@/lib/graphql/dispatch-console";
import { BASE_ONLY_UNAVAILABLE, seedPlan, unavailableFeatures } from "@/test/plan";
import { cleanup, render, screen } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { usePermissionStore } from "@trenova/shared/stores/permission-store";
import { Operation, Resource, type UnavailableFeature } from "@trenova/shared/types/permission";
import type { User } from "@trenova/shared/types/user";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ConsolePageActions } from "../console-page-actions";
import { SummaryStrip } from "../summary-strip";
import type { DispatchActions } from "../use-dispatch-actions";

vi.mock("../dispatch-timeline", () => ({
  DispatchTimeline: () => <div data-testid="dispatch-timeline" />,
}));

const summary = {
  uncoveredMoves: 4,
  coveredMoves: 9,
  lateMoves: 1,
  atRiskMoves: 2,
  unseatedDrivers: 3,
  availableDrivers: 7,
} as DispatchBoardSummary;

const actions = {
  canUndo: false,
  undoLabel: null,
  isAssigning: false,
  isDeciding: false,
  isPlanning: false,
  undo: vi.fn(),
} as unknown as DispatchActions;

function signIn(unavailable: readonly UnavailableFeature[]) {
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
            brokerageEnabled: true,
            assetOperationsEnabled: true,
          },
        },
      ],
    } as unknown as User,
    isAuthenticated: true,
  });
  seedPlan({ [Resource.ShipmentMove]: Operation.Read | Operation.Assign }, unavailable);
}

function renderConsoleChrome() {
  return render(
    <NuqsTestingAdapter searchParams="">
      <ConsolePageActions actions={actions} onPropose={vi.fn()} />
      <SummaryStrip
        summary={summary}
        proposalsWaiting={5}
        exceptions={{ count: 2, more: false }}
        isPending={false}
        isError={false}
        urgencyFocus={null}
        onUrgencyFocus={vi.fn()}
        onShowExceptions={vi.fn()}
        onShowOpenDrivers={vi.fn()}
        showsDrivers
      />
    </NuqsTestingAdapter>,
  );
}

afterEach(() => {
  cleanup();
  useAuthStore.setState({ user: null, isAuthenticated: false });
  usePermissionStore.getState().clearPermissions();
});

describe("dispatch console on a base-only plan", () => {
  it("shows no Nova mark, no Propose coverage, no Proposals waiting and no Exceptions figure", () => {
    signIn(BASE_ONLY_UNAVAILABLE);

    const { container } = renderConsoleChrome();

    expect(screen.queryByRole("button", { name: "Propose coverage" })).toBeNull();
    expect(screen.queryByText("Proposals waiting")).toBeNull();
    expect(screen.queryByText("Nova picked a driver")).toBeNull();
    expect(screen.queryByText("Exceptions")).toBeNull();
    expect(container.querySelector('[data-slot="nova-badge"]')).toBeNull();
    expect(screen.getByText("Needs a driver")).toBeInTheDocument();
    expect(screen.getByText("Open drivers")).toBeInTheDocument();
  });
});

describe("dispatch console with every add-on", () => {
  it("offers Propose coverage and draws the proposals and exceptions figures", () => {
    signIn([]);

    renderConsoleChrome();

    expect(screen.getByRole("button", { name: "Propose coverage" })).toBeInTheDocument();
    expect(screen.getByText("Proposals waiting")).toBeInTheDocument();
    expect(screen.getByText("Exceptions")).toBeInTheDocument();
  });
});

describe("Propose coverage needs both add-ons it uses", () => {
  it("is withheld with Dispatch Intelligence but without Trenova AI", () => {
    signIn(unavailableFeatures("ai.desk"));

    renderConsoleChrome();

    expect(screen.queryByRole("button", { name: "Propose coverage" })).toBeNull();
    expect(screen.queryByText("Proposals waiting")).toBeNull();
  });

  it("is withheld with Trenova AI but without Dispatch Intelligence", () => {
    signIn(unavailableFeatures("tms.dispatch_intelligence"));

    renderConsoleChrome();

    expect(screen.queryByRole("button", { name: "Propose coverage" })).toBeNull();
    expect(screen.getByText("Proposals waiting")).toBeInTheDocument();
  });
});

describe("the exceptions figure follows the Watchtower", () => {
  it("is not drawn without the Watchtower, while proposals still are", () => {
    signIn(unavailableFeatures("ai.watchtower"));

    renderConsoleChrome();

    expect(screen.queryByText("Exceptions")).toBeNull();
    expect(screen.getByText("Proposals waiting")).toBeInTheDocument();
  });
});
