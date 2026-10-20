import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { preview } from "@/components/assistant/__tests__/preview-fixtures";
import type { ProposalPreview, ProposalPreviewRequest } from "@/lib/graphql/agent-preview";
import { DecisionFlow } from "../decision-flow";
import type { PendingDecisionNode } from "../use-pending-decisions";
import { planNode, proposalNode } from "./pending-fixtures";

const fetchProposalPreview =
  vi.fn<(request: ProposalPreviewRequest, options?: unknown) => Promise<ProposalPreview>>();
const decideAgentProposal = vi.fn<(...args: unknown[]) => Promise<unknown>>();
const decideAgentProposals = vi.fn<(...args: unknown[]) => Promise<unknown>>();
const permissions = vi.hoisted(() => ({ manage: true }));
let nodes: PendingDecisionNode[] = [];

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (request: ProposalPreviewRequest, options?: unknown) =>
    fetchProposalPreview(request, options),
}));

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideAgentPlan: vi.fn(),
  decideAgentProposal: (...args: unknown[]) => decideAgentProposal(...args),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: (resource: string) => ({
    allowed: resource === "agent_proposal" || permissions.manage,
    isLoading: false,
  }),
}));

vi.mock("../use-pending-decisions", async (importOriginal) => ({
  ...(await importOriginal<typeof import("../use-pending-decisions")>()),
  usePendingDecisions: () => ({
    data: { nodes, totalCount: nodes.length },
    hasNextPage: false,
    isLoading: false,
    isFetchingNextPage: false,
    fetchNextPage: vi.fn(),
  }),
  useRecentDecisions: () => ({ data: [] }),
  usePlanSteps: () => ({ data: [] }),
  decideAgentProposals: (...args: unknown[]) => decideAgentProposals(...args),
}));

afterEach(() => {
  fetchProposalPreview.mockReset();
  decideAgentProposal.mockReset();
  decideAgentProposals.mockReset();
  permissions.manage = true;
  nodes = [];
});

function renderFlow() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DecisionFlow />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const press = (key: string) => act(() => void fireEvent.keyDown(document.body, { key }));
const paused = { reason: "OrganizationPaused" as const, agentName: "" };

/*
Every organization starts with its agents paused. The queue used to read
"Queue clear" while changes waited behind the pause; it now lists them,
marked held, and in place of Approve names the switch and links to it.
*/
describe("DecisionFlow with held changes", () => {
  it("lists a change held by the organization's pause, with nothing to approve or decline", async () => {
    nodes = [proposalNode({ id: "aprop_1", hold: paused })];
    fetchProposalPreview.mockImplementation(async (request) =>
      preview({ proposalId: request.id, tool: "receivable_post_customer_payment" }),
    );
    renderFlow();

    expect(await screen.findByRole("status")).toHaveTextContent(
      "Agents are paused for the organization",
    );
    expect(screen.queryByText("Queue clear")).toBeNull();
    expect(screen.getByRole("link", { name: "Resume agents" })).toHaveAttribute(
      "href",
      "/admin/agent-control",
    );
    expect(screen.queryByRole("button", { name: /^Approve/ })).toBeNull();
    expect(screen.queryByRole("button", { name: /^Decline/ })).toBeNull();

    press("a");
    press("d");
    expect(decideAgentProposal).not.toHaveBeenCalled();
    expect(decideAgentProposals).not.toHaveBeenCalled();
    expect(screen.queryByText(/could not be loaded/)).toBeNull();
  });

  it("names an agent in shadow and links to its sheet", async () => {
    nodes = [
      planNode({
        hold: { reason: "AgentShadow", agentName: "Billing exceptions" },
        run: {
          id: "arun_1",
          agentDefinitionId: "agdef_billing",
          trigger: "Chat",
          subjectType: "AssistantThread",
          subjectId: "athr_1",
          summary: "",
          definition: {
            id: "agdef_billing",
            name: "Billing exceptions",
            template: "BillingException",
            icon: "receipt",
            accent: "teal",
          },
        } as PendingDecisionNode["run"],
      }),
    ];
    renderFlow();

    expect(await screen.findByRole("status")).toHaveTextContent("Billing exceptions is in shadow");
    expect(screen.getByRole("link", { name: "Turn off shadow" })).toHaveAttribute(
      "href",
      "/admin/agent-control?tab=agents&openAgent=agdef_billing",
    );
  });

  it("tells someone who may not turn the switch off that an administrator can", async () => {
    permissions.manage = false;
    nodes = [proposalNode({ id: "aprop_1", hold: paused })];
    fetchProposalPreview.mockImplementation(async (request) =>
      preview({ proposalId: request.id, tool: "receivable_post_customer_payment" }),
    );
    renderFlow();

    expect(await screen.findByRole("status")).toHaveTextContent(
      "Agents are paused for the organization · An administrator can turn this off.",
    );
    expect(screen.queryByRole("link", { name: "Resume agents" })).toBeNull();
  });

  it("never offers to decide a held change together with live ones", async () => {
    nodes = [
      proposalNode({ id: "aprop_1" }),
      proposalNode({ id: "aprop_2", createdAt: 1_790_000_100, hold: paused }),
    ];
    fetchProposalPreview.mockImplementation(async (request) =>
      preview({ proposalId: request.id, tool: "receivable_post_customer_payment" }),
    );
    renderFlow();

    await waitFor(() =>
      expect(screen.getByRole("button", { name: /^Approve/ })).toBeInTheDocument(),
    );
    expect(screen.queryByText(/Decide all/)).toBeNull();
    expect(screen.queryByText(/more receivable_post_customer_payment waiting/)).toBeNull();
  });
});
