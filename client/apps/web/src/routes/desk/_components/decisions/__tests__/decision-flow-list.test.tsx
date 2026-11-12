import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { stubLayout } from "@/test/layout";
import {
  message,
  preview,
  reason,
  record,
  warning,
} from "@/components/assistant/__tests__/preview-fixtures";
import type {
  BatchProposalPreview,
  ProposalPreview,
  ProposalPreviewRequest,
} from "@/lib/graphql/agent-preview";
import { DecisionFlow, type DecisionFlowProgress } from "../decision-flow";
import type { PendingDecisionNode } from "../use-pending-decisions";
import { planNode, proposalNode } from "./pending-fixtures";

const fetchProposalPreview =
  vi.fn<(request: ProposalPreviewRequest, options?: unknown) => Promise<ProposalPreview>>();
const fetchProposalPreviews =
  vi.fn<(ids: readonly string[], options?: unknown) => Promise<BatchProposalPreview[]>>();
const decideAgentProposal = vi.fn<(...args: unknown[]) => Promise<unknown>>();
const decideAgentProposals = vi.fn<(...args: unknown[]) => Promise<unknown>>();
let nodes: PendingDecisionNode[] = [];

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (request: ProposalPreviewRequest, options?: unknown) =>
    fetchProposalPreview(request, options),
  fetchProposalPreviews: (ids: readonly string[], options?: unknown) =>
    fetchProposalPreviews(ids, options),
  fetchPlanPreview: () => new Promise(() => {}),
}));

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideAgentPlan: vi.fn(),
  decideAgentProposal: (...args: unknown[]) => decideAgentProposal(...args),
  decideAgentProposals: (...args: unknown[]) => decideAgentProposals(...args),
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
}));

let restoreLayout = () => {};

beforeEach(() => {
  restoreLayout = stubLayout(600);
});

afterEach(() => {
  restoreLayout();
  fetchProposalPreview.mockReset();
  fetchProposalPreviews.mockReset();
  decideAgentProposal.mockReset();
  decideAgentProposals.mockReset();
  nodes = [];
});

function renderFlow(head: (progress: DecisionFlowProgress) => null = () => null) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DecisionFlow head={head} />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

const press = (key: string) => act(() => void fireEvent.keyDown(document.body, { key }));
const button = (name: string | RegExp) => screen.getByRole("button", { name });
const box = (driver: string) => screen.getByRole("checkbox", { name: `Select ${driver}` });
const dock = () => screen.getByRole("region", { name: "Selected changes" });

const DRIVERS = ["Laura Wright", "Theresa Aguilar", "Sam Ortiz", "Ana Ruiz"];
const driverOf = (id: string) => DRIVERS[(Number(id.split("_")[1]) - 1) % DRIVERS.length];

/** A Dash message asking a driver for a signed POD, as the routine files it. */
function podMessage(id: string, overrides: Partial<ProposalPreview> = {}) {
  const index = Number(id.split("_")[1]);
  const driver = driverOf(id);
  const pro = `S10${String(index).padStart(2, "0")}`;
  return preview({
    proposalId: id,
    tool: "worker_send_message",
    digest: `sha256:${id}`,
    reviewRequired: true,
    changes: [
      record({
        resource: "worker",
        record: { entityType: "worker", id: `wrk_${index}` },
        entityId: `wrk_${index}`,
        label: driver,
        operation: "Send",
        fields: [],
        message: message({
          channel: "Dash",
          from: "Billing",
          to: [driver],
          subject: `Signed POD needed: PRO ${pro}`,
          body: `Hi ${driver.split(" ")[0]}, please send the signed proof of delivery for PRO ${pro}. A clear photo of the signed paperwork in Dash is fine.`,
        }),
      }),
    ],
    ...overrides,
  });
}

function queueOf(count: number, reviewRequired = true) {
  nodes = Array.from({ length: count }, (_, i) =>
    proposalNode({
      id: `aprop_${i + 1}`,
      toolName: "worker_send_message",
      reviewRequired,
      rationale: `No signed proof of delivery is on file for load ${i + 1}.`,
      createdAt: 1_790_000_000 + i,
    }),
  );
}

function previewsOf(overrides: Record<string, Partial<BatchProposalPreview>> = {}) {
  fetchProposalPreviews.mockImplementation(async (ids) =>
    ids.map((id) => ({ proposalId: id, preview: podMessage(id), error: null, ...overrides[id] })),
  );
}

const approved = (proposalId: string) => ({
  proposalId,
  executed: true,
  refused: false,
  error: null,
  decision: null,
});

const sensitive = warning({
  code: "sensitive_content",
  message: "The message contains details your organization marks as sensitive.",
});

describe("DecisionFlow: the list picks and ticks, the page reads one change", () => {
  it("names each row by who its message goes to and shows only the open change on the page", async () => {
    queueOf(3);
    previewsOf();
    renderFlow();

    for (const driver of ["Laura Wright", "Theresa Aguilar", "Sam Ortiz"]) {
      expect(await screen.findByRole("checkbox", { name: `Select ${driver}` })).not.toBeChecked();
    }
    expect(
      screen.getByText("No signed proof of delivery is on file for load 1."),
    ).toBeInTheDocument();
    expect(screen.queryByText("No signed proof of delivery is on file for load 2.")).toBeNull();
    expect(screen.queryByRole("region", { name: "The same for every one" })).toBeNull();
    expect(fetchProposalPreviews).toHaveBeenCalledTimes(1);
    expect(fetchProposalPreview).not.toHaveBeenCalled();

    press("j");
    expect(
      await screen.findByText("No signed proof of delivery is on file for load 2."),
    ).toBeInTheDocument();
    expect(screen.queryByText("No signed proof of delivery is on file for load 1.")).toBeNull();
  });

  it("tells the page's head how much is decided and how much waits", async () => {
    queueOf(3);
    previewsOf();
    const head = vi.fn<(progress: DecisionFlowProgress) => null>(() => null);
    renderFlow(head);

    await screen.findByRole("checkbox", { name: "Select Laura Wright" });
    expect(head).toHaveBeenLastCalledWith({
      progressLabel: "0 of 3 decided",
      done: 0,
      total: 3,
      waiting: 3,
      keysEnabled: true,
    });
  });

  it("approves the open change alone with its digest and opens the next", async () => {
    queueOf(3);
    previewsOf();
    decideAgentProposal.mockResolvedValue(approved("aprop_1"));
    renderFlow();

    await waitFor(() => expect(button("Approve")).toBeEnabled());
    press("a");

    await waitFor(() => expect(decideAgentProposal).toHaveBeenCalledTimes(1));
    expect(decideAgentProposal).toHaveBeenCalledWith("aprop_1", {
      decision: "Accepted",
      reasonCode: "approved_from_desk",
      previewDigest: "sha256:aprop_1",
    });
    expect(decideAgentProposals).not.toHaveBeenCalled();
    expect(
      await screen.findByText("No signed proof of delivery is on file for load 2."),
    ).toBeInTheDocument();
  });

  it("decides changes ticked in the list together, after one review, with the digest of each", async () => {
    queueOf(3);
    previewsOf();
    decideAgentProposals.mockResolvedValue([approved("aprop_1"), approved("aprop_3")]);
    renderFlow();

    fireEvent.click(await screen.findByRole("checkbox", { name: "Select Laura Wright" }));
    fireEvent.click(box("Sam Ortiz"));

    expect(within(dock()).getByText("2 selected")).toBeInTheDocument();
    expect(within(dock()).getByRole("button", { name: "Decline 2" })).toBeEnabled();
    // The page still decides the open change alone.
    expect(button("Approve")).toBeEnabled();
    expect(decideAgentProposals).not.toHaveBeenCalled();

    fireEvent.click(await screen.findByRole("button", { name: "Review and approve 2" }));
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByRole("heading", { name: "Approve 2 changes?" })).toBeInTheDocument();
    const shared = within(dialog).getByRole("region", { name: "The same for every one" });
    expect(within(shared).getByText("Driver portal")).toBeInTheDocument();
    expect(within(dialog).getByText("Laura Wright")).toBeInTheDocument();
    expect(within(dialog).getByText("Sam Ortiz")).toBeInTheDocument();
    expect(within(dialog).queryByText("Theresa Aguilar")).toBeNull();

    fireEvent.click(within(dialog).getByRole("button", { name: "Approve 2" }));

    await waitFor(() => expect(decideAgentProposals).toHaveBeenCalledTimes(1));
    expect(decideAgentProposals).toHaveBeenCalledWith(["aprop_1", "aprop_3"], {
      decision: "Accepted",
      reasonCode: "approved_from_desk_batch",
      previewDigests: [
        { proposalId: "aprop_1", digest: "sha256:aprop_1" },
        { proposalId: "aprop_3", digest: "sha256:aprop_3" },
      ],
    });
    expect(decideAgentProposal).not.toHaveBeenCalled();
  });

  it("ticks the open change with X, a whole kind from its heading, and lets them go with Escape", async () => {
    queueOf(3);
    previewsOf();
    renderFlow();

    await screen.findByRole("checkbox", { name: "Select Laura Wright" });
    press("x");
    expect(box("Laura Wright")).toBeChecked();
    expect(box("Sam Ortiz")).not.toBeChecked();

    fireEvent.click(button("Select all"));
    for (const driver of ["Laura Wright", "Theresa Aguilar", "Sam Ortiz"]) {
      expect(box(driver)).toBeChecked();
    }
    expect(within(dock()).getByText("3 selected")).toBeInTheDocument();

    press("Escape");
    for (const driver of ["Laura Wright", "Theresa Aguilar", "Sam Ortiz"]) {
      expect(box(driver)).not.toBeChecked();
    }
    expect(dock()).toHaveAttribute("inert");
    expect(within(dock()).getByText("3 selected")).toBeInTheDocument();
  });

  it("ticks only the changes with nothing flagged when asked", async () => {
    queueOf(3);
    previewsOf({ aprop_2: { preview: podMessage("aprop_2", { warnings: [sensitive] }) } });
    renderFlow();

    fireEvent.click(await screen.findByRole("button", { name: "Select 2 with nothing flagged" }));

    expect(box("Laura Wright")).toBeChecked();
    expect(box("Theresa Aguilar")).not.toBeChecked();
    expect(box("Sam Ortiz")).toBeChecked();
    expect(within(dock()).queryByText(/something to look at first/)).toBeNull();

    fireEvent.click(box("Theresa Aguilar"));
    expect(
      within(dock()).getByText("1 of these has something to look at first."),
    ).toBeInTheDocument();
    expect(within(dock()).getByRole("button", { name: "Review and approve 3" })).toBeEnabled();
  });

  it("leaves a change whose record moved out of an approval, though it can still be declined", async () => {
    queueOf(3);
    previewsOf({ aprop_2: { preview: podMessage("aprop_2", { stale: true }) } });
    renderFlow();

    await screen.findByRole("checkbox", { name: "Select Laura Wright" });
    fireEvent.click(button("Select all"));

    expect(within(dock()).getByRole("button", { name: "Review and approve 2" })).toBeEnabled();
    expect(within(dock()).getByRole("button", { name: "Decline 3" })).toBeEnabled();
    expect(
      within(dock()).getByText("1 can't be approved as it stands and is left out of an approval."),
    ).toBeInTheDocument();
  });

  it("approves internal changes unreviewed when they could not be read", async () => {
    queueOf(2, false);
    fetchProposalPreviews.mockRejectedValue(new Error("network"));
    decideAgentProposals.mockResolvedValue([approved("aprop_1"), approved("aprop_2")]);
    renderFlow();

    await waitFor(() => expect(button("Approve")).toBeEnabled());
    fireEvent.click(button("Select all"));
    fireEvent.click(await screen.findByRole("button", { name: "Review and approve 2" }));
    fireEvent.click(await screen.findByRole("button", { name: "Approve 2" }));

    await waitFor(() => expect(decideAgentProposals).toHaveBeenCalledTimes(1));
    expect(decideAgentProposals).toHaveBeenCalledWith(["aprop_1", "aprop_2"], {
      decision: "Accepted",
      reasonCode: "approved_from_desk_batch",
      previewDigests: [],
    });
  });

  it("never approves changes that must be reviewed when they could not be read", async () => {
    queueOf(2, true);
    fetchProposalPreviews.mockRejectedValue(new Error("network"));
    renderFlow();

    expect(
      await screen.findByText(
        "What this would change could not be loaded. It has to be reviewed before it can be approved, so try again.",
      ),
    ).toBeInTheDocument();
    expect(button("Approve")).toBeDisabled();

    fireEvent.click(button("Select all"));
    expect(within(dock()).getByRole("button", { name: "Review and approve 0" })).toBeDisabled();
    expect(within(dock()).getByRole("button", { name: "Decline 2" })).toBeEnabled();
  });

  it("sends a change that can't run back to its agent with the reasons written", async () => {
    queueOf(1);
    previewsOf({
      aprop_1: {
        preview: podMessage("aprop_1", {
          warnings: [
            warning({
              reasons: [
                reason({
                  label: "Worker",
                  param: "workerId",
                  message: "Laura Wright has no driver portal access",
                }),
              ],
            }),
          ],
        }),
      },
    });
    renderFlow();

    fireEvent.click(await screen.findByRole("button", { name: "Send back to the agent" }));

    expect(screen.queryByRole("button", { name: /^Approve/ })).toBeNull();
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByRole("heading", { name: "Decline this change?" })).toBeVisible();
    expect((within(dialog).getByRole("textbox") as HTMLTextAreaElement).value).toContain(
      "Worker: Laura Wright has no driver portal access",
    );
  });

  it("reads a long queue fifty at a time and ticks no more than one decision carries", async () => {
    queueOf(51);
    previewsOf();
    renderFlow();

    await waitFor(() => expect(fetchProposalPreviews).toHaveBeenCalledTimes(2));
    expect(fetchProposalPreviews.mock.calls.map(([ids]) => ids.length)).toEqual([50, 1]);

    fireEvent.click(button("Select all"));
    expect(within(dock()).getByText("50 selected")).toBeInTheDocument();
    expect(within(dock()).getByText("Up to 50 can be decided together.")).toBeInTheDocument();
  });

  it("gives a plan no checkbox: it is decided as one, on its own", async () => {
    nodes = [planNode({ id: "apl_1", title: "Post two payments" })];
    renderFlow();

    expect(await screen.findByRole("heading", { level: 1 })).toHaveTextContent("Post two payments");
    expect(screen.queryByRole("checkbox")).toBeNull();
    expect(screen.queryByRole("button", { name: "Select all" })).toBeNull();
  });
});
