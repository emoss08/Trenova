import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter } from "react-router";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { message, preview, record } from "@/components/assistant/__tests__/preview-fixtures";
import type {
  BatchProposalPreview,
  ProposalPreview,
  ProposalPreviewRequest,
} from "@/lib/graphql/agent-preview";
import { DecisionFlow } from "../decision-flow";
import type { PendingDecisionNode } from "../use-pending-decisions";
import { proposalNode } from "./pending-fixtures";
import { stubListReadToEnd } from "./read-to-end";

const fetchProposalPreview =
  vi.fn<(request: ProposalPreviewRequest, options?: unknown) => Promise<ProposalPreview>>();
const fetchProposalPreviews =
  vi.fn<(ids: readonly string[], options?: unknown) => Promise<BatchProposalPreview[]>>();
const decideAgentProposals = vi.fn<(...args: unknown[]) => Promise<unknown>>();
let nodes: PendingDecisionNode[] = [];

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (request: ProposalPreviewRequest, options?: unknown) =>
    fetchProposalPreview(request, options),
  fetchProposalPreviews: (ids: readonly string[], options?: unknown) =>
    fetchProposalPreviews(ids, options),
}));

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideAgentPlan: vi.fn(),
  decideAgentProposal: vi.fn(),
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

beforeEach(() => {
  stubListReadToEnd();
});

afterEach(() => {
  fetchProposalPreview.mockReset();
  fetchProposalPreviews.mockReset();
  decideAgentProposals.mockReset();
  vi.unstubAllGlobals();
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

const DRIVERS = ["Laura Wright", "Theresa Aguilar", "Sam Ortiz", "Ana Ruiz"];

/** A Dash message asking a driver for a signed POD, as the routine files it. */
function podMessage(id: string, index: number, overrides: Partial<ProposalPreview> = {}) {
  const driver = DRIVERS[(index - 1) % DRIVERS.length];
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
      createdAt: 1_790_000_000 + i,
    }),
  );
  fetchProposalPreview.mockImplementation(async (request) =>
    podMessage(request.id, Number(request.id.split("_")[1])),
  );
}

function previewsOf(overrides: Record<string, Partial<BatchProposalPreview>> = {}) {
  fetchProposalPreviews.mockImplementation(async (ids) =>
    ids.map((id) => ({
      proposalId: id,
      preview: podMessage(id, Number(id.split("_")[1])),
      error: null,
      ...overrides[id],
    })),
  );
}

describe("DecisionFlow deciding changes together", () => {
  it("shows every change on one sheet and approves them with the digest of each", async () => {
    queueOf(3);
    previewsOf();
    decideAgentProposals.mockResolvedValue(
      ["aprop_1", "aprop_2", "aprop_3"].map((proposalId) => ({ proposalId, error: null })),
    );
    renderFlow();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    press("x");

    const sheet = await screen.findByRole("region", { name: "The same for every one" });
    expect(within(sheet).getByText("Driver portal")).toBeInTheDocument();
    expect(within(sheet).getByText("Billing")).toBeInTheDocument();
    const rows = screen.getByRole("list", { name: "Changes in this decision" });
    expect(within(rows).getByText("Laura Wright")).toBeInTheDocument();
    expect(within(rows).getByText("Theresa Aguilar")).toBeInTheDocument();
    expect(within(rows).getByText("Sam Ortiz")).toBeInTheDocument();
    expect(fetchProposalPreview).toHaveBeenCalledTimes(1);

    await waitFor(() => expect(screen.getByRole("button", { name: "Approve 3" })).toBeEnabled());
    press("a");

    await waitFor(() => expect(decideAgentProposals).toHaveBeenCalledTimes(1));
    const [ids, input] = decideAgentProposals.mock.calls[0] as [string[], Record<string, unknown>];
    expect(ids).toEqual(["aprop_1", "aprop_2", "aprop_3"]);
    expect(input).toEqual({
      decision: "Accepted",
      reasonCode: "approved_from_desk_batch",
      previewDigests: [
        { proposalId: "aprop_1", digest: "sha256:aprop_1" },
        { proposalId: "aprop_2", digest: "sha256:aprop_2" },
        { proposalId: "aprop_3", digest: "sha256:aprop_3" },
      ],
    });
  });

  it("holds the approval until the list has been read to its end", async () => {
    let reachEnd: (() => void) | null = null;
    vi.stubGlobal(
      "IntersectionObserver",
      class {
        callback: IntersectionObserverCallback;
        constructor(callback: IntersectionObserverCallback) {
          this.callback = callback;
        }
        observe() {
          reachEnd = () =>
            this.callback(
              [{ isIntersecting: true } as IntersectionObserverEntry],
              this as unknown as IntersectionObserver,
            );
        }
        disconnect() {}
      },
    );
    queueOf(2);
    previewsOf();
    renderFlow();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    press("x");

    expect(await screen.findByText(/Read to the end of the list to approve/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Approve 2" })).toBeDisabled();
    press("a");
    expect(decideAgentProposals).not.toHaveBeenCalled();

    act(() => reachEnd?.());

    await waitFor(() => expect(screen.getByRole("button", { name: "Approve 2" })).toBeEnabled());
    expect(screen.queryByText(/Read to the end of the list to approve/)).toBeNull();
  });

  it("leaves a change the person unticks in the queue", async () => {
    queueOf(3);
    previewsOf();
    decideAgentProposals.mockResolvedValue([
      { proposalId: "aprop_1", error: null },
      { proposalId: "aprop_3", error: null },
    ]);
    renderFlow();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    press("x");
    fireEvent.click(await screen.findByRole("checkbox", { name: "Include Theresa Aguilar" }));

    await waitFor(() => expect(screen.getByRole("button", { name: "Approve 2" })).toBeEnabled());
    press("a");

    await waitFor(() => expect(decideAgentProposals).toHaveBeenCalledTimes(1));
    const [ids, input] = decideAgentProposals.mock.calls[0] as [string[], Record<string, unknown>];
    expect(ids).toEqual(["aprop_1", "aprop_3"]);
    expect(input.previewDigests).toEqual([
      { proposalId: "aprop_1", digest: "sha256:aprop_1" },
      { proposalId: "aprop_3", digest: "sha256:aprop_3" },
    ]);
  });

  it("decides a change whose record moved, or whose preview failed, on its own card", async () => {
    queueOf(3);
    previewsOf({
      aprop_2: { preview: podMessage("aprop_2", 2, { stale: true }) },
      aprop_3: { preview: null, error: "Proposal not found" },
    });
    renderFlow();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    press("x");

    expect(
      await screen.findByText(
        "Its record changed since it was proposed. Decide it on its own card.",
      ),
    ).toBeInTheDocument();
    expect(screen.getByText("Proposal not found")).toBeInTheDocument();
    expect(screen.queryByRole("checkbox", { name: "Include Sam Ortiz" })).toBeNull();
    await waitFor(() => expect(screen.getByRole("button", { name: "Approve 1" })).toBeEnabled());
    expect(screen.getByRole("button", { name: "Decline 1" })).toBeEnabled();
  });

  it("approves internal changes without previews when the sheet could not read them", async () => {
    queueOf(2, false);
    fetchProposalPreviews.mockRejectedValue(new Error("network"));
    decideAgentProposals.mockResolvedValue([
      { proposalId: "aprop_1", error: null },
      { proposalId: "aprop_2", error: null },
    ]);
    renderFlow();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    press("x");

    await waitFor(() => expect(screen.getByRole("button", { name: "Approve 2" })).toBeEnabled());
    press("a");

    await waitFor(() => expect(decideAgentProposals).toHaveBeenCalledTimes(1));
    const [ids, input] = decideAgentProposals.mock.calls[0] as [string[], Record<string, unknown>];
    expect(ids).toEqual(["aprop_1", "aprop_2"]);
    expect(input.previewDigests).toEqual([]);
  });

  it("never approves changes that must be reviewed when the sheet could not read them", async () => {
    queueOf(2, true);
    fetchProposalPreviews.mockRejectedValue(new Error("network"));
    renderFlow();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    press("x");

    expect(await screen.findByText(/The changes could not be read/)).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Approve 0" })).toBeDisabled();
    expect(screen.getByRole("button", { name: "Decline 2" })).toBeEnabled();
  });

  it("decides a long run fifty at a time", async () => {
    queueOf(51);
    previewsOf();
    renderFlow();

    await waitFor(() => expect(approveButton()).toBeEnabled());
    expect(screen.getByRole("button", { name: "Decide 50 of 51 together" })).toBeInTheDocument();
    press("x");

    await waitFor(() => expect(fetchProposalPreviews).toHaveBeenCalledTimes(1));
    expect(fetchProposalPreviews.mock.calls[0][0]).toHaveLength(50);
    expect(await screen.findByText(/The first 50 of 51/)).toBeInTheDocument();
  });
});
