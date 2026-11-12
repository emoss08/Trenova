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
import { stubRowsInView } from "./sheet-test-kit";

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

beforeEach(() => {
  stubRowsInView();
});

afterEach(() => {
  fetchProposalPreview.mockReset();
  fetchProposalPreviews.mockReset();
  decideAgentProposal.mockReset();
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
const button = (name: string | RegExp) => screen.getByRole("button", { name });

const DRIVERS = ["Laura Wright", "Theresa Aguilar", "Sam Ortiz", "Ana Ruiz"];

/** A Dash message asking a driver for a signed POD, as the routine files it. */
function podMessage(id: string, overrides: Partial<ProposalPreview> = {}) {
  const index = Number(id.split("_")[1]);
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

describe("DecisionFlow's sheet of changes of one kind", () => {
  it("shows every change of a kind on one sheet, what they share once and the open one whole", async () => {
    queueOf(3);
    previewsOf();
    renderFlow();

    const shared = await screen.findByRole("region", { name: "The same for every one" });
    expect(within(shared).getByText("Driver portal")).toBeInTheDocument();
    expect(within(shared).getByText("Billing")).toBeInTheDocument();
    const rows = screen.getByRole("list", { name: "Changes waiting" });
    for (const driver of ["Laura Wright", "Theresa Aguilar", "Sam Ortiz"]) {
      expect(within(rows).getByRole("button", { name: new RegExp(driver) })).toBeInTheDocument();
    }
    expect(within(rows).getAllByRole("listitem")[0]).toHaveAttribute("aria-current", "true");
    expect(
      screen.getByText("No signed proof of delivery is on file for load 1."),
    ).toBeInTheDocument();
    expect(screen.queryByText("No signed proof of delivery is on file for load 2.")).toBeNull();
    expect(fetchProposalPreviews).toHaveBeenCalledTimes(1);
    expect(fetchProposalPreview).not.toHaveBeenCalled();

    press("j");
    expect(
      await screen.findByText("No signed proof of delivery is on file for load 2."),
    ).toBeInTheDocument();
    expect(within(rows).getAllByRole("listitem")[1]).toHaveAttribute("aria-current", "true");
    expect(fetchProposalPreviews).toHaveBeenCalledTimes(1);
  });

  it("approves the open change alone, with its digest, while nothing is ticked", async () => {
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
  });

  it("decides the changes ticked with X together, with the digest of each", async () => {
    queueOf(3);
    previewsOf();
    decideAgentProposals.mockResolvedValue([approved("aprop_1"), approved("aprop_3")]);
    renderFlow();

    await waitFor(() => expect(button("Approve")).toBeEnabled());
    press("x");
    press("j");
    press("j");
    press("x");

    await waitFor(() => expect(button("Approve 2")).toBeEnabled());
    expect(button("Decline 2")).toBeEnabled();
    expect(screen.getByText("2 selected")).toBeInTheDocument();
    press("a");

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

  it("ticks every change at once, and lets them go with Escape", async () => {
    queueOf(3);
    previewsOf();
    renderFlow();

    fireEvent.click(await screen.findByRole("checkbox", { name: "Select all 3" }));
    await waitFor(() => expect(button("Approve 3")).toBeEnabled());

    press("Escape");
    expect(await screen.findByText("Tick changes to decide them together")).toBeInTheDocument();
    expect(button("Approve")).toBeEnabled();
  });

  it("holds approving ticked changes until each has been on screen", async () => {
    const shown: {
      observer: IntersectionObserver;
      targets: Element[];
      callback: IntersectionObserverCallback;
    }[] = [];
    vi.stubGlobal(
      "IntersectionObserver",
      class {
        targets: Element[] = [];
        callback: IntersectionObserverCallback;
        constructor(callback: IntersectionObserverCallback) {
          this.callback = callback;
          shown.push({
            observer: this as unknown as IntersectionObserver,
            targets: this.targets,
            callback,
          });
        }
        observe(target: Element) {
          this.targets.push(target);
        }
        unobserve() {}
        disconnect() {}
      },
    );
    queueOf(3);
    previewsOf();
    renderFlow();

    fireEvent.click(await screen.findByRole("checkbox", { name: "Select all 3" }));

    expect(
      await screen.findByText("Scroll past each selected change to approve them"),
    ).toBeInTheDocument();
    expect(button("Approve 1")).toBeDisabled();
    press("a");
    expect(decideAgentProposals).not.toHaveBeenCalled();

    act(() => {
      for (const { observer, targets, callback } of shown) {
        callback(
          targets.map(
            (target) => ({ isIntersecting: true, target }) as unknown as IntersectionObserverEntry,
          ),
          observer,
        );
      }
    });

    await waitFor(() => expect(button("Approve 3")).toBeEnabled());
  });

  it("leaves a change whose record moved out of an approval, though it can still be declined", async () => {
    queueOf(3);
    previewsOf({ aprop_2: { preview: podMessage("aprop_2", { stale: true }) } });
    renderFlow();

    fireEvent.click(await screen.findByRole("checkbox", { name: "Select all 3" }));

    await waitFor(() => expect(button("Approve 2")).toBeEnabled());
    expect(button("Decline 3")).toBeEnabled();
    expect(
      screen.getByText("Its record changed since it was proposed. It can only be declined."),
    ).toBeInTheDocument();
  });

  it("approves internal changes unreviewed when the sheet could not read them", async () => {
    queueOf(2, false);
    fetchProposalPreviews.mockRejectedValue(new Error("network"));
    decideAgentProposals.mockResolvedValue([approved("aprop_1"), approved("aprop_2")]);
    renderFlow();

    await waitFor(() => expect(button("Approve")).toBeEnabled());
    fireEvent.click(screen.getByRole("checkbox", { name: "Select all 2" }));
    await waitFor(() => expect(button("Approve 2")).toBeEnabled());
    press("a");

    await waitFor(() => expect(decideAgentProposals).toHaveBeenCalledTimes(1));
    expect(decideAgentProposals).toHaveBeenCalledWith(["aprop_1", "aprop_2"], {
      decision: "Accepted",
      reasonCode: "approved_from_desk_batch",
      previewDigests: [],
    });
  });

  it("never approves changes that must be reviewed when the sheet could not read them", async () => {
    queueOf(2, true);
    fetchProposalPreviews.mockRejectedValue(new Error("network"));
    renderFlow();

    expect(
      await screen.findByText(
        "What this would change could not be loaded. It has to be reviewed before it can be approved, so try again.",
      ),
    ).toBeInTheDocument();
    expect(button("Approve")).toBeDisabled();
    fireEvent.click(screen.getByRole("checkbox", { name: "Select all 2" }));
    expect(await screen.findByText("The changes could not be read")).toBeInTheDocument();
    expect(button("Approve 0")).toBeDisabled();
    expect(button("Decline 2")).toBeEnabled();
  });

  it("reads one change on its own as a card, with nothing to tick", async () => {
    queueOf(1);
    previewsOf();
    renderFlow();

    expect(
      await screen.findByText("No signed proof of delivery is on file for load 1."),
    ).toBeInTheDocument();
    expect(screen.queryByRole("checkbox")).toBeNull();
    expect(screen.queryByRole("region", { name: "The same for every one" })).toBeNull();
    await waitFor(() => expect(button("Approve")).toBeEnabled());
  });

  it("opens a long run of one kind fifty at a time", async () => {
    queueOf(51);
    previewsOf();
    renderFlow();

    await waitFor(() => expect(fetchProposalPreviews).toHaveBeenCalledTimes(1));
    expect(fetchProposalPreviews.mock.calls[0][0]).toHaveLength(50);
    expect(await screen.findByText(/the first 50 of 51/)).toBeInTheDocument();
  });
});
