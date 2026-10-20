import type { AgentProposalRow } from "@/lib/graphql/agent-activity-tables";
import { stubLayout } from "@/test/layout";
import { translate } from "@trenova/shared/i18n/runtime";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { NuqsTestingAdapter } from "nuqs/adapters/testing";
import type { ReactNode } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { decidable } from "../activity-model";
import { PlanStatusBadge, ProposalStatusBadge } from "../agent-badges";
import { ProposalActionsContext, ProposalSheet, type ProposalActions } from "../proposal-sheet";

const permissions = vi.hoisted(() => ({ manage: true }));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: permissions.manage, isLoading: false }),
}));

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout();
});

afterEach(() => {
  restoreLayout();
  cleanup();
  permissions.manage = true;
});

const t = translate;
const paused = { reason: "OrganizationPaused" as const, agentName: "" };

function wrap(children: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return (
    <QueryClientProvider client={client}>
      <NuqsTestingAdapter>
        <MemoryRouter>{children}</MemoryRouter>
      </NuqsTestingAdapter>
    </QueryClientProvider>
  );
}

/* AgentProposalTableRowFields as the agentProposals connection serves it. */
function proposal(overrides: Partial<AgentProposalRow> = {}): AgentProposalRow {
  return {
    id: "aprop_1",
    runId: "arun_1",
    toolName: "home_add_widget",
    toolParams: {},
    confidence: "0",
    rationale: "Add the fleet status widget",
    autonomyTier: "ActWithApproval",
    status: "Pending",
    parameterFields: [],
    tainted: false,
    hold: null,
    createdAt: 1_800_000_000,
    updatedAt: 1_800_000_000,
    ...overrides,
  } as unknown as AgentProposalRow;
}

function actions(): ProposalActions {
  return { canDecide: true, approve: vi.fn(), modify: vi.fn(), reject: vi.fn() };
}

/*
Every organization starts with its agents paused. AI Control's activity
lists used to read empty while it was; they now list what waits, marked
held, with the switch named and linked where the decision buttons were.
*/
describe("AI Control activity with held changes", () => {
  it("names the organization's pause in a held proposal's sheet, with nothing to decide", () => {
    render(
      wrap(
        <ProposalActionsContext.Provider value={actions()}>
          <ProposalSheet proposal={proposal({ hold: paused })} onClose={() => {}} />
        </ProposalActionsContext.Provider>,
      ),
    );

    expect(screen.getByText("Held")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Agents are paused for the organization");
    expect(screen.getByRole("link", { name: "Resume agents" })).toHaveAttribute(
      "href",
      "/admin/agent-control",
    );
    expect(screen.queryByRole("button", { name: "Approve" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Reject" })).toBeNull();
  });

  it("tells someone who may not turn the switch off that an administrator can", () => {
    permissions.manage = false;
    render(
      wrap(
        <ProposalActionsContext.Provider value={actions()}>
          <ProposalSheet
            proposal={proposal({
              hold: { reason: "AgentShadow", agentName: "Billing exceptions" },
            })}
            onClose={() => {}}
          />
        </ProposalActionsContext.Provider>,
      ),
    );

    expect(screen.getByRole("status")).toHaveTextContent(
      "Billing exceptions is in shadow · An administrator can turn this off.",
    );
    expect(screen.queryByRole("link")).toBeNull();
  });

  it("keeps the decision buttons on a waiting proposal nothing holds", () => {
    render(
      wrap(
        <ProposalActionsContext.Provider value={actions()}>
          <ProposalSheet proposal={proposal()} onClose={() => {}} />
        </ProposalActionsContext.Provider>,
      ),
    );

    expect(screen.getByRole("button", { name: "Approve" })).toBeInTheDocument();
    expect(screen.queryByText("Held")).toBeNull();
  });

  it("offers a decision from the lists only on a waiting row nothing holds", () => {
    expect(decidable({ status: "Pending", hold: null }, true)).toBe(true);
    expect(decidable({ status: "Pending", hold: paused }, true)).toBe(false);
    expect(decidable({ status: "Executed", hold: null }, true)).toBe(false);
    expect(decidable({ status: "Pending", hold: null }, false)).toBe(false);
  });

  it("reads a held waiting proposal or plan as held, and a decided one as decided", () => {
    const { rerender } = render(<ProposalStatusBadge value="Pending" held t={t} />);
    expect(screen.getByText("Held")).toBeInTheDocument();

    rerender(<ProposalStatusBadge value="Executed" held t={t} />);
    expect(screen.getByText("Done")).toBeInTheDocument();

    rerender(<PlanStatusBadge value="Pending" held t={t} />);
    expect(screen.getByText("Held")).toBeInTheDocument();

    rerender(<PlanStatusBadge value="Pending" t={t} />);
    expect(screen.getByText("Awaiting decision")).toBeInTheDocument();
  });
});
