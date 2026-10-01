import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type {
  PlanPreview,
  ProposalPreview,
  ProposalPreviewRequest,
} from "@/lib/graphql/agent-preview";
import { stubLayout } from "@/test/layout";
import type { AssistantPlan, AssistantProposal, ProposalField } from "@/types/assistant";
import { ApprovalDock } from "../approval-dock";
import { approvalQueue, type ApprovalEntry } from "../approval-queue";
import { DecisionFollowUpProvider } from "../decision-follow-up";
import { planPreview, preview, reason, warning } from "./preview-fixtures";

const fetchProposalPreview =
  vi.fn<(request: ProposalPreviewRequest, options?: unknown) => Promise<ProposalPreview>>();
const fetchPlanPreview = vi.fn<(...args: unknown[]) => Promise<PlanPreview>>();
const decideMyProposal = vi.fn<(...args: unknown[]) => Promise<unknown>>();
const decideMyProposals = vi.fn<(...args: unknown[]) => Promise<unknown>>();
const decideMyPlan = vi.fn<(...args: unknown[]) => Promise<unknown>>();
const decideAgentProposal = vi.fn<(...args: unknown[]) => Promise<unknown>>();

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (request: ProposalPreviewRequest, options?: unknown) =>
    fetchProposalPreview(request, options),
  fetchPlanPreview: (...args: unknown[]) => fetchPlanPreview(...args),
}));

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideMyProposal: (...args: unknown[]) => decideMyProposal(...args),
  decideMyProposals: (...args: unknown[]) => decideMyProposals(...args),
  decideMyPlan: (...args: unknown[]) => decideMyPlan(...args),
  decideAgentProposal: (...args: unknown[]) => decideAgentProposal(...args),
}));

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout(320);
  decideMyProposal.mockResolvedValue({});
  decideMyPlan.mockResolvedValue({});
});

afterEach(() => {
  cleanup();
  restoreLayout();
  fetchProposalPreview.mockReset();
  fetchPlanPreview.mockReset();
  decideMyProposal.mockReset();
  decideMyProposals.mockReset();
  decideMyPlan.mockReset();
  decideAgentProposal.mockReset();
});

/*
Fixtures follow services.AssistantProposal and services.AssistantPlan as the
thread's routes serve them, and the previews follow agentpreview.graphqls
(preview-fixtures). A proposal outside any plan has an empty planId.
*/
function proposal(overrides: Partial<AssistantProposal> = {}): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    toolName: "update_shipment",
    arguments: { shipmentId: "shp_1", status: "Delayed" },
    rationale: "",
    autonomyTier: "Propose",
    status: "Pending",
    sourceMessageId: "amsg_1",
    executedAt: null,
    executionError: "",
    expiresAt: 0,
    planId: "",
    planStep: 0,
    fields: [],
    createdAt: 100,
    decidedAt: null,
    decidedByUserId: "",
    decisionNote: "",
    ...overrides,
  };
}

function plan(overrides: Partial<AssistantPlan> = {}): AssistantPlan {
  return {
    id: "apl_1",
    runId: "arun_1",
    title: "Dispatch coverage: 2 changes",
    summary: "Both open moves get a driver.",
    status: "Pending",
    stepCount: 2,
    completedSteps: 0,
    failedStep: null,
    failureError: "",
    decidedAt: null,
    decidedByUserId: "",
    expiresAt: 0,
    hold: null,
    createdAt: 100,
    ...overrides,
  };
}

function entryOf(proposals: AssistantProposal[], plans: AssistantPlan[] = []): ApprovalEntry {
  const [first] = approvalQueue(proposals, plans, null, 1_000);
  if (!first) {
    throw new Error("nothing waiting");
  }
  return first;
}

function renderDock({
  entry = entryOf([proposal()]),
  canTell = true,
  compact = false,
  position = 1,
  total = 1,
}: {
  entry?: ApprovalEntry;
  canTell?: boolean;
  compact?: boolean;
  position?: number;
  total?: number;
} = {}) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const followUp = vi.fn();
  const onDefer = vi.fn();
  const onDecided = vi.fn();

  const view = render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DecisionFollowUpProvider value={followUp}>
          <div data-slot="assistant-thread" className="relative">
            <ApprovalDock
              threadId="athr_1"
              entry={entry}
              position={position}
              total={total}
              compact={compact}
              canTell={canTell}
              onDefer={onDefer}
              onDecided={onDecided}
            />
          </div>
        </DecisionFollowUpProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { ...view, followUp, onDefer, onDecided };
}

const dock = () => screen.getByRole("region");
const approve = () => screen.getByRole("button", { name: /^approve/i });
const reject = () => screen.getByRole("button", { name: /^reject/i });

/** Lets a mutation the keys may have started reach the server mock. */
const settle = () => act(() => new Promise<void>((resolve) => setTimeout(resolve, 20)));

function conflict() {
  const type = "https://trenova.app/problems/resource-conflict";
  return new GraphQLRequestError({
    kind: "graphql",
    message: "The change looks different now",
    status: 200,
    graphQLErrors: [{ message: "The change looks different now", extensions: { type }, type }],
  });
}

describe("ApprovalDock for one proposal", () => {
  it("reads the preview as the conversation's owner and keeps Approve off until it is in", () => {
    fetchProposalPreview.mockReturnValue(new Promise(() => {}));
    renderDock();

    expect(fetchProposalPreview).toHaveBeenCalledWith(
      { scope: "mine", id: "aprop_1", modifications: null },
      expect.anything(),
    );
    expect(approve()).toBeDisabled();
    expect(reject()).toBeEnabled();
  });

  it("approves with the digest of the preview on screen", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockResolvedValue(preview({ digest: "sha256:shown" }));
    const { followUp, onDecided } = renderDock();

    await screen.findByRole("link", { name: "S-1001" });
    await user.click(approve());

    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenCalledWith("aprop_1", {
        decision: "Accepted",
        reasonCode: "",
        previewDigest: "sha256:shown",
      }),
    );
    await waitFor(() => expect(followUp).toHaveBeenCalledWith("aprop_1"));
    expect(onDecided).toHaveBeenCalledTimes(1);
    expect(decideAgentProposal).not.toHaveBeenCalled();
  });

  it("approves on ⌘/Ctrl+Enter and never on Enter alone", async () => {
    fetchProposalPreview.mockResolvedValue(preview({ digest: "sha256:shown" }));
    renderDock();
    await screen.findByRole("link", { name: "S-1001" });

    fireEvent.keyDown(dock(), { key: "Enter" });
    fireEvent.keyDown(dock(), { key: "Enter", shiftKey: true });
    await settle();
    expect(decideMyProposal).not.toHaveBeenCalled();

    fireEvent.keyDown(dock(), { key: "Enter", ctrlKey: true });
    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenCalledWith(
        "aprop_1",
        expect.objectContaining({ decision: "Accepted", previewDigest: "sha256:shown" }),
      ),
    );
  });

  it("does not approve on the keys before there is a preview to approve", async () => {
    fetchProposalPreview.mockReturnValue(new Promise(() => {}));
    renderDock();

    fireEvent.keyDown(dock(), { key: "Enter", metaKey: true });
    await settle();

    expect(decideMyProposal).not.toHaveBeenCalled();
  });

  it("turns Approve off for a record that changed since, and still rejects", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockResolvedValue(preview({ stale: true, digest: "sha256:stale" }));
    renderDock();

    await screen.findByText("Changed since it was proposed");
    expect(approve()).toBeDisabled();

    await user.click(reject());
    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenCalledWith("aprop_1", {
        decision: "Rejected",
        reasonCode: "",
        previewDigest: "sha256:stale",
      }),
    );
  });

  it("reads the preview again after a digest conflict and approves the new one", async () => {
    const user = userEvent.setup();
    fetchProposalPreview
      .mockResolvedValueOnce(preview({ digest: "sha256:before" }))
      .mockResolvedValue(preview({ digest: "sha256:after" }));
    decideMyProposal.mockRejectedValueOnce(conflict()).mockResolvedValue({});
    renderDock();

    await screen.findByRole("link", { name: "S-1001" });
    await user.click(approve());

    expect(
      await screen.findByText("This change looks different now — review it again."),
    ).toBeInTheDocument();
    await waitFor(() => expect(fetchProposalPreview.mock.calls.length).toBeGreaterThanOrEqual(2));
    await waitFor(() => expect(approve()).toBeEnabled());

    await user.click(approve());
    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenLastCalledWith("aprop_1", {
        decision: "Accepted",
        reasonCode: "",
        previewDigest: "sha256:after",
      }),
    );
  });

  it("says how many wait and which this is", () => {
    fetchProposalPreview.mockReturnValue(new Promise(() => {}));
    renderDock({ position: 1, total: 3 });

    expect(within(dock()).getByText("1 of 3")).toBeInTheDocument();
  });
});

describe("telling the agent instead", () => {
  it("rejects with the words typed, and the decision's own follow-up answers them", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockResolvedValue(preview({ digest: "sha256:shown" }));
    const { followUp } = renderDock();
    await screen.findByRole("link", { name: "S-1001" });

    await user.click(screen.getByRole("button", { name: "Tell the agent instead" }));
    const box = screen.getByLabelText(/What should the agent do instead/);
    expect(box).toHaveFocus();
    await user.type(box, "Mark it delayed tomorrow, not today.");
    await user.click(screen.getByRole("button", { name: "Send to the agent" }));

    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenCalledWith("aprop_1", {
        decision: "Rejected",
        reasonCode: "",
        previewDigest: "sha256:shown",
        note: "Mark it delayed tomorrow, not today.",
      }),
    );
    await waitFor(() => expect(followUp).toHaveBeenCalledTimes(1));
    expect(decideMyProposal).toHaveBeenCalledTimes(1);
  });

  it("opens and closes on Esc, keeping what was typed, and sends on Enter", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockReturnValue(new Promise(() => {}));
    renderDock();

    fireEvent.keyDown(dock(), { key: "Escape" });
    const box = screen.getByLabelText(/What should the agent do instead/);
    await user.type(box, "Wrong customer");
    fireEvent.keyDown(box, { key: "Escape" });
    expect(screen.queryByLabelText(/What should the agent do instead/)).not.toBeInTheDocument();
    expect(decideMyProposal).not.toHaveBeenCalled();

    fireEvent.keyDown(dock(), { key: "Escape" });
    const reopened = screen.getByLabelText(/What should the agent do instead/);
    expect(reopened).toHaveValue("Wrong customer");
    fireEvent.keyDown(reopened, { key: "Enter" });

    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenCalledWith(
        "aprop_1",
        expect.objectContaining({ decision: "Rejected", note: "Wrong customer" }),
      ),
    );
  });

  it("sends nothing until there is something to say", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockReturnValue(new Promise(() => {}));
    renderDock();

    await user.click(screen.getByRole("button", { name: "Tell the agent instead" }));
    const box = screen.getByLabelText(/What should the agent do instead/);
    await user.type(box, "   ");
    fireEvent.keyDown(box, { key: "Enter" });

    expect(screen.getByRole("button", { name: "Send to the agent" })).toBeDisabled();
    expect(decideMyProposal).not.toHaveBeenCalled();
  });

  it("is not offered where the agent can no longer answer", () => {
    fetchProposalPreview.mockReturnValue(new Promise(() => {}));
    renderDock({ canTell: false });

    expect(screen.queryByRole("button", { name: "Tell the agent instead" })).toBeNull();
    fireEvent.keyDown(dock(), { key: "Escape" });
    expect(screen.queryByLabelText(/What should the agent do instead/)).toBeNull();
  });

  // A write the preview says would be refused is turned down with its
  // reasons written for the agent, which answers with a corrected proposal.
  it("writes a refusal's reasons for the agent when asked to fix it", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockResolvedValue(
      preview({
        tool: "create_shipment",
        changes: [],
        warnings: [warning({ reasons: [reason()] })],
      }),
    );
    renderDock({
      entry: entryOf([
        proposal({
          toolName: "create_shipment",
          arguments: { shipment: { bol: "SEED-BOL-009" } },
        }),
      ]),
    });

    await screen.findByText("BOL is already in use by shipment SEED-DET-009");
    await user.click(screen.getByRole("button", { name: "Ask the agent to fix it" }));

    expect(screen.getByLabelText(/What should the agent do instead/)).toHaveValue(
      "The proposal to create shipment would not go through as it stands: BOL: BOL is already in use by shipment SEED-DET-009. Fix it and propose it again; ask me for anything you need.",
    );
    expect(decideMyProposal).not.toHaveBeenCalled();
  });
});

describe("deciding later", () => {
  it("puts the decision off from the button and from Alt+L", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockReturnValue(new Promise(() => {}));
    const { onDefer } = renderDock();

    await user.click(screen.getByRole("button", { name: "Decide later" }));
    fireEvent.keyDown(dock(), { key: "¬", code: "KeyL", altKey: true });

    expect(onDefer).toHaveBeenCalledTimes(2);
    expect(decideMyProposal).not.toHaveBeenCalled();
  });
});

describe("ApprovalDock for a plan", () => {
  function planEntry() {
    return entryOf(
      [
        proposal({ id: "aprop_1", planId: "apl_1", planStep: 1 }),
        proposal({ id: "aprop_2", planId: "apl_1", planStep: 2 }),
      ],
      [plan()],
    );
  }

  it("shows each step and approves the whole plan against its digest", async () => {
    const user = userEvent.setup();
    fetchPlanPreview.mockResolvedValue(planPreview({ digest: "sha256:plan" }));
    const { followUp } = renderDock({ entry: planEntry() });

    await waitFor(() => expect(approve()).toBeEnabled());
    expect(screen.getByText("2 changes, in order")).toBeInTheDocument();
    expect(screen.getByText("Uses the record step 1 changes")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Approve all 2" }));

    await waitFor(() =>
      expect(decideMyPlan).toHaveBeenCalledWith("apl_1", {
        decision: "Accepted",
        reasonCode: "",
        previewDigest: "sha256:plan",
      }),
    );
    await waitFor(() => expect(followUp).toHaveBeenCalledTimes(1));
    expect(decideMyProposal).not.toHaveBeenCalled();
  });

  it("tells the agent why the whole plan was turned down", async () => {
    const user = userEvent.setup();
    fetchPlanPreview.mockResolvedValue(planPreview({ digest: "sha256:plan" }));
    renderDock({ entry: planEntry() });

    await user.click(screen.getByRole("button", { name: "Tell the agent instead" }));
    await user.type(screen.getByLabelText(/What should the agent do instead/), "Swap the drivers");
    fireEvent.keyDown(dock(), { key: "Enter", ctrlKey: true });

    await waitFor(() =>
      expect(decideMyPlan).toHaveBeenCalledWith("apl_1", {
        decision: "Rejected",
        reasonCode: "",
        previewDigest: "sha256:plan",
        note: "Swap the drivers",
      }),
    );
  });
});

describe("ApprovalDock for several changes of one kind", () => {
  function batch(count: number) {
    return entryOf(
      Array.from({ length: count }, (_, index) =>
        proposal({
          id: `aprop_${index + 1}`,
          toolName: "post_invoice",
          arguments: { invoiceId: `inv_${index + 1}` },
          createdAt: 100 + index,
        }),
      ),
    );
  }

  it("approves them in one answer with the digest of each preview shown", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockImplementation(async (request) =>
      preview({ proposalId: request.id, digest: `sha256:${request.id}` }),
    );
    decideMyProposals.mockResolvedValue([
      { proposalId: "aprop_1", executed: true, error: null, decision: null },
      { proposalId: "aprop_2", executed: true, error: null, decision: null },
    ]);
    const { followUp } = renderDock({ entry: batch(2) });

    await waitFor(() => expect(approve()).toBeEnabled());
    await waitFor(() => expect(screen.getAllByRole("link", { name: "S-1001" })).toHaveLength(2));
    await user.click(screen.getByRole("button", { name: "Approve all 2" }));

    await waitFor(() =>
      expect(decideMyProposals).toHaveBeenCalledWith(["aprop_1", "aprop_2"], {
        decision: "Accepted",
        reasonCode: undefined,
        previewDigests: [
          { proposalId: "aprop_1", digest: "sha256:aprop_1" },
          { proposalId: "aprop_2", digest: "sha256:aprop_2" },
        ],
        note: undefined,
      }),
    );
    await waitFor(() => expect(followUp).toHaveBeenCalledTimes(1));
  });

  // Past a few, each opens on demand; only what was open on screen sends a
  // digest, and the rest are recorded as approved without review.
  it("sends a digest only for the previews that were opened", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockImplementation(async (request) =>
      preview({ proposalId: request.id, digest: `sha256:${request.id}` }),
    );
    decideMyProposals.mockResolvedValue([]);
    renderDock({ entry: batch(6) });

    expect(fetchProposalPreview).not.toHaveBeenCalled();
    expect(
      screen.getByText(
        "6 of them are not open; approving records them as approved without reviewing what they change.",
      ),
    ).toBeInTheDocument();

    const rows = screen.getAllByRole("button", { expanded: false });
    await user.click(rows[2]);
    await screen.findByRole("link", { name: "S-1001" });
    await user.click(screen.getByRole("button", { name: "Approve all 6" }));

    await waitFor(() => expect(decideMyProposals).toHaveBeenCalledTimes(1));
    expect(decideMyProposals.mock.calls[0][1]).toMatchObject({
      decision: "Accepted",
      previewDigests: [{ proposalId: "aprop_3", digest: "sha256:aprop_3" }],
    });
  });

  it("rejects them together with the note, under a reason the server asks for", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockReturnValue(new Promise(() => {}));
    decideMyProposals.mockResolvedValue([]);
    renderDock({ entry: batch(2) });

    await user.click(screen.getByRole("button", { name: "Tell the agent instead" }));
    await user.type(
      screen.getByLabelText(/What should the agent do instead/),
      "These customers are on credit hold",
    );
    await user.click(screen.getByRole("button", { name: "Send to the agent" }));

    await waitFor(() =>
      expect(decideMyProposals).toHaveBeenCalledWith(["aprop_1", "aprop_2"], {
        decision: "Rejected",
        reasonCode: "rejected_in_conversation",
        previewDigests: undefined,
        note: "These customers are on credit hold",
      }),
    );
  });
});

describe("ApprovalDock for a write over a set of records", () => {
  const SHP_1 = "shp_01K0000000000000000000001";
  const SHP_2 = "shp_01K0000000000000000000002";
  const subset: ProposalField = {
    name: "shipmentIds",
    label: "Shipment ids",
    description: "",
    kind: "RecordSubset",
    required: true,
    options: [],
    minimum: null,
    maximum: null,
    maxLength: null,
    readOnly: false,
    resource: "shipment",
    choices: [
      { id: SHP_1, label: "PRO-1001" },
      { id: SHP_2, label: "PRO-1002" },
    ],
  };

  it("approves only the records kept, against the preview of what was kept", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockImplementation(async ({ modifications }) =>
      preview({
        tool: "transfer_to_billing",
        digest:
          modifications === null
            ? "sha256:both"
            : `sha256:${(modifications.shipmentIds as string[]).join("+")}`,
      }),
    );
    renderDock({
      entry: entryOf([
        proposal({
          toolName: "transfer_to_billing",
          arguments: { shipmentIds: [SHP_1, SHP_2] },
          fields: [subset],
        }),
      ]),
    });

    await screen.findAllByRole("link", { name: "S-1001" });
    await user.click(screen.getByRole("checkbox", { name: "PRO-1002" }));
    const kept = screen.getByRole("button", { name: /Approve the kept records/ });
    await waitFor(() => expect(kept).toBeEnabled());
    await user.click(kept);

    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenCalledWith("aprop_1", {
        decision: "Modified",
        modifications: { shipmentIds: [SHP_1] },
        reasonCode: "",
        previewDigest: `sha256:${SHP_1}`,
      }),
    );
  });
});

describe("ApprovalDock in the floating panel", () => {
  it("offers the same answers and keys as on the Desk, without the key hints", async () => {
    fetchProposalPreview.mockResolvedValue(preview({ digest: "sha256:shown" }));
    renderDock({ compact: true });
    await screen.findByRole("link", { name: "S-1001" });

    for (const name of [/^approve$/i, /^reject$/i, /tell the agent instead/i, /decide later/i]) {
      expect(screen.getByRole("button", { name })).toBeInTheDocument();
    }
    expect(within(dock()).queryByText("Esc")).toBeNull();

    fireEvent.keyDown(dock(), { key: "Enter", metaKey: true });
    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenCalledWith(
        "aprop_1",
        expect.objectContaining({ decision: "Accepted", previewDigest: "sha256:shown" }),
      ),
    );
  });
});
