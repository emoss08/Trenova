import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { PendingProposalFieldsFragment } from "@trenova/graphql/generated/graphql";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { preview, reason, warning } from "@/components/assistant/__tests__/preview-fixtures";
import type { ProposalPreview } from "@/lib/graphql/agent-preview";
import { DecisionDetail, type DecisionActions } from "../decision-detail";
import type { PendingProposalNode } from "../use-pending-decisions";

const fetchProposalPreview = vi.fn<(...args: unknown[]) => Promise<ProposalPreview>>();

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (...args: unknown[]) => fetchProposalPreview(...args),
}));

afterEach(() => {
  fetchProposalPreview.mockReset();
});

type ParameterField = PendingProposalFieldsFragment["parameterFields"][number];

const SHIPMENT_FIELD: ParameterField = {
  name: "shipment",
  label: "Shipment",
  description: "The shipment, in the shape the shipment form sends.",
  kind: "JSON",
  required: true,
  options: [],
  minimum: null,
  maximum: null,
  maxLength: null,
  readOnly: false,
  resource: null,
  choices: null,
};

/** The Shipment Desk's copy of SEED-DET-009, proposed with its BOL. */
function copiedShipment(): PendingProposalNode {
  return {
    __typename: "AgentProposal",
    id: "aprop_1",
    organizationId: "org_1",
    businessUnitId: "bu_1",
    runId: "arun_1",
    toolName: "create_shipment",
    toolParams: { shipment: { customerId: "cus_1", bol: "SEED-BOL-009", moves: [] } },
    confidence: 0.9,
    rationale: "A copy of SEED-DET-009 for next week.",
    autonomyTier: "Propose",
    status: "Pending",
    planId: null,
    planStep: 0,
    simulatedAt: null,
    simulation: null,
    modifications: null,
    version: 1,
    createdAt: 1_790_000_000,
    updatedAt: 1_790_000_000,
    parameterFields: [SHIPMENT_FIELD],
    evidence: [],
    run: null,
  };
}

function renderDetail(overrides: Partial<DecisionActions> = {}) {
  const actions: DecisionActions = {
    onAccept: vi.fn(),
    onReject: vi.fn(),
    onModify: vi.fn(),
    busy: false,
    ...overrides,
  };
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DecisionDetail node={copiedShipment()} actions={actions} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return actions;
}

/*
The Desk's decision pane used to show a create_shipment that reused a taken
BOL as "This would not go through as it stands." with nothing to act on. It
now names the BOL and offers the two ways forward a queue has: change the
value, or turn the proposal down with the reasons so the agent fixes it.
*/
describe("DecisionDetail for a write that would be refused", () => {
  it("names each reason and keeps approve offered", async () => {
    fetchProposalPreview.mockResolvedValue(
      preview({ tool: "create_shipment", warnings: [warning({ reasons: [reason()] })] }),
    );
    renderDetail();

    expect(
      await screen.findByText("BOL is already in use by shipment SEED-DET-009"),
    ).toBeInTheDocument();
    expect(screen.queryByText("This would not go through as it stands.")).not.toBeInTheDocument();
    expect(screen.getByRole("button", { name: /^Approve/ })).toBeEnabled();
  });

  it("opens the editor on the value a reason names", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockResolvedValue(
      preview({ tool: "create_shipment", warnings: [warning({ reasons: [reason()] })] }),
    );
    const actions = renderDetail();

    await user.click(await screen.findByRole("button", { name: "Change BOL" }));

    expect(actions.onModify).toHaveBeenCalledWith({ param: "shipment.bol", label: "BOL" });
  });

  it("rejects with the reasons written when the agent is asked to fix it", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockResolvedValue(
      preview({ tool: "create_shipment", warnings: [warning({ reasons: [reason()] })] }),
    );
    const actions = renderDetail();

    await user.click(await screen.findByRole("button", { name: "Ask the agent to fix it" }));

    expect(actions.onReject).toHaveBeenCalledWith(
      "The proposal to create shipment would not go through as it stands: BOL: BOL is already in use by shipment SEED-DET-009. Fix it and propose it again; ask me for anything you need.",
    );
  });

  it("offers neither while a decision is being recorded", async () => {
    fetchProposalPreview.mockResolvedValue(
      preview({ tool: "create_shipment", warnings: [warning({ reasons: [reason()] })] }),
    );
    renderDetail({ busy: true });

    await screen.findByText("BOL is already in use by shipment SEED-DET-009");
    expect(screen.queryByRole("button", { name: "Change BOL" })).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "Ask the agent to fix it" }),
    ).not.toBeInTheDocument();
  });
});
