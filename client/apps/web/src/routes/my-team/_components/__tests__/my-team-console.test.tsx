import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import MyTeamConsole from "../my-team-console";

const { fetchMyTeam } = vi.hoisted(() => ({ fetchMyTeam: vi.fn() }));

vi.mock("@/lib/graphql/org-structure", () => ({
  fetchMyTeam,
  MY_TEAM_KEY: "my-team",
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
}));

function member(over: Record<string, unknown> = {}) {
  return {
    workerId: "wrk_1",
    name: "Ada Byrne",
    status: "Active",
    fleetCodeId: "fc_1",
    fleetCode: "SOUTH",
    fleetColor: "#2563eb",
    positionId: "jpos_1",
    positionTitle: "Over-the-Road Driver",
    direct: true,
    complianceStatus: "Compliant",
    trainingHealth: "Current",
    safetyRating: "Excellent",
    hireDate: 1_700_000_000,
    terminationDate: null,
    ...over,
  };
}

function renderConsole(rows: unknown[]) {
  fetchMyTeam.mockResolvedValue(rows);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <MemoryRouter>
      <QueryClientProvider client={client}>
        <MyTeamConsole />
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

afterEach(() => {
  cleanup();
  vi.clearAllMocks();
});

describe("MyTeamConsole", () => {
  // A terminal manager covering forty drivers is not forty direct reports, and
  // a page that conflated the two would misstate the span of control.
  it("separates direct reports from people reached through a terminal", async () => {
    renderConsole([
      member(),
      member({ workerId: "wrk_2", name: "Ben Cole", direct: false }),
      member({ workerId: "wrk_3", name: "Cal Diaz", direct: false }),
    ]);

    const directHeading = await screen.findByRole("heading", { name: "Direct reports" });

    const directSection = directHeading.closest("section") as HTMLElement;
    expect(within(directSection).getByText("Ada Byrne")).toBeInTheDocument();
    expect(within(directSection).queryByText("Ben Cole")).not.toBeInTheDocument();

    const terminalSection = screen
      .getByRole("heading", { name: "Through a terminal you run" })
      .closest("section") as HTMLElement;
    expect(within(terminalSection).getByText("Ben Cole")).toBeInTheDocument();
    expect(within(terminalSection).getByText("Cal Diaz")).toBeInTheDocument();
  });

  it("counts the two paths separately in the header", async () => {
    renderConsole([
      member(),
      member({ workerId: "wrk_2", name: "Ben Cole", direct: false }),
      member({ workerId: "wrk_3", name: "Cal Diaz", direct: false }),
    ]);

    const directFigure = (await screen.findByText("Direct reports", { selector: "p" }))
      .parentElement as HTMLElement;
    expect(within(directFigure).getByText("1")).toBeInTheDocument();

    const terminalFigure = screen.getByText("Through a terminal", { selector: "p" })
      .parentElement as HTMLElement;
    expect(within(terminalFigure).getByText("2")).toBeInTheDocument();
  });

  // The point of the page is knowing who needs chasing, so the count has to
  // cover every way somebody can need it — not just one.
  it("counts anybody non-compliant, overdue on training or at risk", async () => {
    renderConsole([
      member(),
      member({ workerId: "wrk_2", name: "Ben Cole", complianceStatus: "NonCompliant" }),
      member({ workerId: "wrk_3", name: "Cal Diaz", trainingHealth: "Overdue" }),
      member({ workerId: "wrk_4", name: "Dee Ellis", safetyRating: "AtRisk" }),
    ]);

    const figure = (await screen.findByText("Needing attention")).parentElement as HTMLElement;
    expect(within(figure).getByText("3")).toBeInTheDocument();
  });

  it("says how somebody joins the team when nobody has", async () => {
    renderConsole([]);

    expect(
      await screen.findByText(/A worker joins your team when their record names you/),
    ).toBeInTheDocument();
  });
});
