import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { preview } from "@/components/assistant/__tests__/preview-fixtures";
import type { ProposalPreview, ProposalPreviewRequest } from "@/lib/graphql/agent-preview";
import { DecisionFlow } from "../decision-flow";
import type { PendingDecisionNode } from "../use-pending-decisions";
import { proposalNode } from "./pending-fixtures";

const fetchProposalPreview =
  vi.fn<(request: ProposalPreviewRequest, options?: unknown) => Promise<ProposalPreview>>();
const decideAgentProposals = vi.fn<(...args: unknown[]) => Promise<unknown>>();
let nodes: PendingDecisionNode[] = [];

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (request: ProposalPreviewRequest, options?: unknown) =>
    fetchProposalPreview(request, options),
}));

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideAgentPlan: vi.fn(),
  decideAgentProposal: vi.fn(),
}));

vi.mock("@/hooks/use-permission", () => ({
  usePermission: () => ({ allowed: true, isLoading: false }),
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
  decideAgentProposals.mockReset();
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
const approveButton = () => screen.getByRole("button", { name: /^Approve/ });

describe("DecisionFlow approving changes together", () => {
  it("holds a batch until every change that must be reviewed has been opened", async () => {
    nodes = [
      proposalNode({ id: "aprop_1", reviewRequired: true }),
      proposalNode({ id: "aprop_2", reviewRequired: true, createdAt: 1_790_000_100 }),
    ];
    fetchProposalPreview.mockImplementation(async (request) =>
      preview({
        proposalId: request.id,
        tool: "receivable_post_customer_payment",
        digest: `sha256:${request.id}`,
      }),
    );
    decideAgentProposals.mockResolvedValue([
      { proposalId: "aprop_1", error: null },
      { proposalId: "aprop_2", error: null },
    ]);
    renderFlow();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    press("x");

    expect(
      await screen.findByText(/1 change must be opened before these can be approved/),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Approve 2/ })).toBeDisabled();
    press("a");
    expect(decideAgentProposals).not.toHaveBeenCalled();

    press("j");
    await waitFor(() =>
      expect(screen.queryByText(/must be opened before these can be approved/)).toBeNull(),
    );
    await waitFor(() => expect(screen.getByRole("button", { name: /^Approve 2/ })).toBeEnabled());
    press("a");

    await waitFor(() => expect(decideAgentProposals).toHaveBeenCalledTimes(1));
    const [ids, input] = decideAgentProposals.mock.calls[0] as [string[], Record<string, unknown>];
    expect([...ids].sort()).toEqual(["aprop_1", "aprop_2"]);
    expect(input).toEqual(
      expect.objectContaining({
        decision: "Accepted",
        previewDigests: expect.arrayContaining([
          { proposalId: "aprop_1", digest: "sha256:aprop_1" },
          { proposalId: "aprop_2", digest: "sha256:aprop_2" },
        ]),
      }),
    );
  });

  it("never counts an opened change whose record moved since as reviewed", async () => {
    nodes = [
      proposalNode({ id: "aprop_1", reviewRequired: true }),
      proposalNode({ id: "aprop_2", reviewRequired: true, createdAt: 1_790_000_100 }),
    ];
    fetchProposalPreview.mockImplementation(async (request) =>
      preview({
        proposalId: request.id,
        tool: "receivable_post_customer_payment",
        digest: `sha256:${request.id}`,
        stale: request.id === "aprop_2",
      }),
    );
    renderFlow();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    press("x");
    press("j");

    await waitFor(() =>
      expect(
        screen.getByText(/1 change must be opened before these can be approved/),
      ).toBeInTheDocument(),
    );
    expect(screen.getByRole("button", { name: /^Approve 2/ })).toBeDisabled();
    press("a");
    expect(decideAgentProposals).not.toHaveBeenCalled();
  });

  it("approves a batch of internal changes without opening each one", async () => {
    nodes = [
      proposalNode({ id: "aprop_1", reviewRequired: false }),
      proposalNode({ id: "aprop_2", reviewRequired: false, createdAt: 1_790_000_100 }),
    ];
    fetchProposalPreview.mockImplementation(async (request) =>
      preview({
        proposalId: request.id,
        tool: "receivable_post_customer_payment",
        digest: `sha256:${request.id}`,
      }),
    );
    decideAgentProposals.mockResolvedValue([
      { proposalId: "aprop_1", error: null },
      { proposalId: "aprop_2", error: null },
    ]);
    renderFlow();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    press("x");
    expect(screen.queryByText(/must be opened before these can be approved/)).toBeNull();
    await waitFor(() => expect(screen.getByRole("button", { name: /^Approve 2/ })).toBeEnabled());
    press("a");

    await waitFor(() => expect(decideAgentProposals).toHaveBeenCalledTimes(1));
  });
});
