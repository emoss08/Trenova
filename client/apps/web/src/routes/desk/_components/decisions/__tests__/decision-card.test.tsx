import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { PendingProposalFieldsFragment } from "@trenova/graphql/generated/graphql";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  field,
  preview,
  reason,
  record,
  warning,
} from "@/components/assistant/__tests__/preview-fixtures";
import type { ProposalPreview } from "@/lib/graphql/agent-preview";
import { DecisionCard, type CardGate } from "../decision-card";
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

function node(overrides: Partial<PendingProposalNode> = {}): PendingProposalNode {
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
    ...overrides,
  };
}

function renderCard(proposal = node()) {
  const props = {
    onGate: vi.fn<(gate: CardGate) => void>(),
    onPreviewShown: vi.fn(),
    onDeclineWith: vi.fn(),
    onModify: vi.fn(),
  };
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DecisionCard node={proposal} leaving={null} {...props} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return props;
}

const refused = () =>
  preview({ tool: "create_shipment", warnings: [warning({ reasons: [reason()] })] });

describe("DecisionCard", () => {
  it("lists each record with the value it sets and says the dry run is clear", async () => {
    fetchProposalPreview.mockResolvedValue(
      preview({
        changes: [
          record({
            label: "S-1001",
            fields: [field({ label: "Driver", before: null, after: "Marcus Hill" })],
          }),
          record({
            label: "S-1002",
            entityId: "shp_2",
            fields: [field({ label: "Driver", before: null, after: "Dana Ortiz" })],
          }),
        ],
      }),
    );
    const props = renderCard(node({ toolName: "assign_driver" }));

    expect(await screen.findByText("S-1001")).toBeInTheDocument();
    expect(screen.getByText("Driver")).toBeInTheDocument();
    expect(screen.getByText("Marcus Hill")).toBeInTheDocument();
    expect(screen.getByText("All 2 shipments would go through as shown.")).toBeInTheDocument();
    expect(props.onPreviewShown).toHaveBeenCalledWith("aprop_1", "sha256:aaaa");
    expect(props.onGate).toHaveBeenLastCalledWith(
      expect.objectContaining({ approvable: true, digest: "sha256:aaaa", count: 2, refused: 0 }),
    );
  });

  it("marks the records the write would refuse and says how many go through", async () => {
    fetchProposalPreview.mockResolvedValue(
      preview({
        changes: [
          record({
            label: "INV-1",
            fields: [field({ label: "Status", after: "Posted" })],
          }),
          record({
            label: "INV-2",
            entityId: "inv_2",
            fields: [
              field({ label: "Status", after: "Posted" }),
              field({
                path: "outcome",
                label: "Outcome",
                after: "Refused: Missing bill-to address",
              }),
            ],
          }),
        ],
      }),
    );
    renderCard(node({ toolName: "post_invoices" }));

    expect(await screen.findByText("Refused")).toBeInTheDocument();
    expect(
      screen.getByText("1 would go through. 1 would be refused: missing bill-to address"),
    ).toBeInTheDocument();
  });

  it("draws a refused write whole, naming each reason", async () => {
    fetchProposalPreview.mockResolvedValue(refused());
    renderCard();

    expect(
      await screen.findByText("BOL is already in use by shipment SEED-DET-009"),
    ).toBeInTheDocument();
  });

  it("opens the editor on the value a reason names", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockResolvedValue(refused());
    const props = renderCard();

    await user.click(await screen.findByRole("button", { name: "Change BOL" }));

    expect(props.onModify).toHaveBeenCalledWith({ param: "shipment.bol", label: "BOL" });
  });

  it("declines with the reasons written when the agent is asked to fix it", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockResolvedValue(refused());
    const props = renderCard();

    await user.click(await screen.findByRole("button", { name: "Ask the agent to fix it" }));

    expect(props.onDeclineWith).toHaveBeenCalledWith(
      "The proposal to create shipment would not go through as it stands: BOL: BOL is already in use by shipment SEED-DET-009. Fix it and propose it again; ask me for anything you need.",
    );
  });
});
