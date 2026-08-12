import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import type { OrganizationCapabilities } from "@trenova/shared/types/organization-capability";
import type { User } from "@trenova/shared/types/user";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DispatchConsoleContent } from "../page-content";

vi.mock("../capacity-rail", () => ({
  CapacityRail: () => <div data-testid="capacity-rail" />,
}));
vi.mock("../console-center-pane", () => ({
  ConsoleCenterPane: () => <div data-testid="console-center-pane" />,
}));
vi.mock("../inspector", () => ({
  Inspector: () => <div data-testid="console-inspector" />,
}));
vi.mock("../console-overlays", () => ({
  ConsoleOverlays: () => null,
}));
vi.mock("../summary-strip", () => ({
  SummaryStrip: () => null,
}));
vi.mock("../console-toolbar", () => ({
  ConsoleToolbar: () => null,
}));
vi.mock("../use-dispatch-actions", () => ({
  useDispatchActions: () => ({
    assign: vi.fn(),
    undo: vi.fn(),
    planAutoAssign: vi.fn(),
    canUndo: false,
    isAssigning: false,
    isPlanning: false,
    plan: null,
  }),
}));
vi.mock("../use-dispatch-hotkeys", () => ({
  useDispatchHotkeys: () => undefined,
}));
vi.mock("@/lib/queries/dispatch-console", () => ({
  dispatchConsoleQueries: {
    board: (input: unknown) => ({
      queryKey: ["test-dispatch-board", input] as const,
      queryFn: async () => ({ moves: [], drivers: [], summary: null }),
    }),
  },
}));

const hybrid: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: true,
};
const brokerageOnly: OrganizationCapabilities = {
  brokerageEnabled: true,
  assetOperationsEnabled: false,
};

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
}

function renderConsole() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <NuqsTestingAdapter searchParams="">
      <QueryClientProvider client={queryClient}>
        <DispatchConsoleContent />
      </QueryClientProvider>
    </NuqsTestingAdapter>,
  );
}

function boardGrid(): HTMLElement {
  const pane = screen.getByTestId("console-center-pane");
  const grid = pane.parentElement;
  if (!grid) {
    throw new Error("center pane is not inside a grid");
  }
  return grid;
}

describe("dispatch console capacity rail", () => {
  afterEach(() => {
    cleanup();
    useAuthStore.setState({ user: null, isAuthenticated: false });
  });

  it("rails driver capacity for an organization that employs drivers", () => {
    signIn(hybrid);

    renderConsole();

    expect(screen.getByTestId("capacity-rail")).toBeInTheDocument();
    expect(boardGrid().className).toContain("lg:grid-cols-[300px_minmax(0,1fr)_320px]");
  });

  it("leaves the rail unmounted for a brokerage", () => {
    signIn(brokerageOnly);

    renderConsole();

    expect(screen.queryByTestId("capacity-rail")).toBeNull();
  });

  it("closes the gap the rail leaves behind instead of holding its column open", () => {
    signIn(brokerageOnly);

    renderConsole();

    expect(boardGrid().className).toContain("lg:grid-cols-[minmax(0,1fr)_320px]");
    expect(boardGrid().className).not.toContain("300px");
  });

  it("keeps the board and inspector a brokerage still works from", () => {
    signIn(brokerageOnly);

    renderConsole();

    expect(screen.getByTestId("console-center-pane")).toBeInTheDocument();
    expect(screen.getByTestId("console-inspector")).toBeInTheDocument();
  });
});
