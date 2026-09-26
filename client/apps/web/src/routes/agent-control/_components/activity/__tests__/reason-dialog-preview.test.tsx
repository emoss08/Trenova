import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { useState } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  planPreview,
  preview,
  reason,
  warning,
} from "@/components/assistant/__tests__/preview-fixtures";
import type { PlanPreview, ProposalPreview } from "@/lib/graphql/agent-preview";
import { ReasonDialog, type ReasonDialogRequest } from "../reason-dialog";

const fetchProposalPreview = vi.fn<(...args: unknown[]) => Promise<ProposalPreview>>();
const fetchPlanPreview = vi.fn<(...args: unknown[]) => Promise<PlanPreview>>();

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (...args: unknown[]) => fetchProposalPreview(...args),
  fetchPlanPreview: (...args: unknown[]) => fetchPlanPreview(...args),
}));

afterEach(() => {
  fetchProposalPreview.mockReset();
  fetchPlanPreview.mockReset();
});

function openDialog(overrides: Partial<ReasonDialogRequest> = {}) {
  const onConfirm = vi.fn<ReasonDialogRequest["onConfirm"]>(async () => {});
  const onClose = vi.fn();
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const request: ReasonDialogRequest = {
    title: "Approve this change?",
    description: "update_shipment will run.",
    confirmLabel: "Approve and run",
    reasonLabel: "Reason",
    requireReason: false,
    preview: { kind: "proposal", scope: "approver", id: "aprop_1", approving: true },
    onConfirm,
    ...overrides,
  };

  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ReasonDialog request={request} onClose={onClose} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return { onConfirm, onClose };
}

const confirm = (name = "Approve and run") => screen.getByRole("button", { name });

/*
AI Control's context menu used to open a dialog that named only the tool. It
now shows what the write would do, and an approval waits for that and carries
its digest, so an approver in AI Control approves what they saw like anyone
else.
*/
describe("ReasonDialog with a preview", () => {
  it("shows what changes and approves with the digest it showed", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockResolvedValue(preview({ digest: "sha256:shown" }));
    const { onConfirm } = openDialog();

    expect(confirm()).toBeDisabled();
    await screen.findByRole("link", { name: "S-1001" });
    expect(screen.getByText("What changes")).toBeInTheDocument();
    await user.click(confirm());

    await waitFor(() => expect(onConfirm).toHaveBeenCalledWith("", "sha256:shown"));
  });

  it("will not approve a stale change but lets it be rejected", async () => {
    fetchProposalPreview.mockResolvedValue(preview({ stale: true }));
    openDialog();
    await screen.findByText("Changed since it was proposed");

    expect(confirm()).toBeDisabled();
  });

  it("does not hold a rejection on the preview", () => {
    fetchProposalPreview.mockReturnValue(new Promise(() => {}));
    openDialog({
      confirmLabel: "Reject",
      destructive: true,
      preview: { kind: "proposal", scope: "approver", id: "aprop_1", approving: false },
    });

    expect(confirm("Reject")).toBeEnabled();
  });

  it("stays open on a digest conflict, reads the preview again and says the change looks different", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockResolvedValue(preview());
    const type = "https://trenova.app/problems/resource-conflict";
    const onConfirm = vi.fn<ReasonDialogRequest["onConfirm"]>(async () => {
      throw new GraphQLRequestError({
        kind: "graphql",
        message: "conflict",
        status: 200,
        graphQLErrors: [{ message: "conflict", extensions: { type }, type }],
      });
    });
    const { onClose } = openDialog({ onConfirm });

    await screen.findByRole("link", { name: "S-1001" });
    await user.click(confirm());

    expect(
      await screen.findByText("This change looks different now — review it again."),
    ).toBeInTheDocument();
    await waitFor(() => expect(fetchProposalPreview).toHaveBeenCalledTimes(2));
    expect(onClose).not.toHaveBeenCalled();
  });

  it("previews a plan's steps and approves with the plan's digest", async () => {
    const user = userEvent.setup();
    fetchPlanPreview.mockResolvedValue(planPreview());
    const { onConfirm } = openDialog({
      preview: { kind: "plan", scope: "approver", id: "apl_1", approving: true },
    });

    expect(await screen.findByText("Uses the record step 1 changes")).toBeInTheDocument();
    await user.click(confirm());

    await waitFor(() => expect(onConfirm).toHaveBeenCalledWith("", "sha256:plan"));
    expect(fetchProposalPreview).not.toHaveBeenCalled();
  });
});

function refusedOverBOL(): ProposalPreview {
  return preview({ tool: "create_shipment", warnings: [warning({ reasons: [reason()] })] });
}

const ASKED =
  "The proposal to create shipment would not go through as it stands: BOL: BOL is already in use by shipment SEED-DET-009. Fix it and propose it again; ask me for anything you need.";

function renderSwitching(
  approve: ReasonDialogRequest,
  reject: (reason: string) => ReasonDialogRequest,
) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  function Surface() {
    const [rejecting, setRejecting] = useState<ReasonDialogRequest | null>(null);
    const [approving] = useState<ReasonDialogRequest>(() => ({
      ...approve,
      onAskAgent: (text) => setRejecting(reject(text)),
    }));

    return <ReasonDialog request={rejecting ?? approving} onClose={() => {}} />;
  }

  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <Surface />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

/*
A write the preview says would be refused used to leave the approver with a
bare "would not go through" and nothing to do but guess a rejection. The
dialog now names each reason, and asking the agent to fix it turns the
approval into a rejection that carries those reasons to the agent.
*/
describe("ReasonDialog for a write that would be refused", () => {
  it("opens a rejection with the reason it was given", () => {
    fetchProposalPreview.mockReturnValue(new Promise(() => {}));
    openDialog({
      confirmLabel: "Reject",
      destructive: true,
      initialReason: "BOL is taken",
      preview: { kind: "proposal", scope: "approver", id: "aprop_1", approving: false },
    });

    expect(screen.getByRole("textbox", { name: /^Reason/ })).toHaveValue("BOL is taken");
  });

  it("keeps approve offered, and turns it into a rejection with the reasons written", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockResolvedValue(refusedOverBOL());
    const onReject = vi.fn<ReasonDialogRequest["onConfirm"]>(async () => {});
    const base: ReasonDialogRequest = {
      title: "Approve this change?",
      description: "create_shipment will run.",
      confirmLabel: "Approve and run",
      reasonLabel: "Reason",
      requireReason: false,
      preview: { kind: "proposal", scope: "approver", id: "aprop_1", approving: true },
      onConfirm: vi.fn(async () => {}),
    };
    renderSwitching(base, (text) => ({
      ...base,
      title: "Reject this change?",
      confirmLabel: "Reject",
      destructive: true,
      requireReason: true,
      initialReason: text,
      preview: { kind: "proposal", scope: "approver", id: "aprop_1", approving: false },
      onConfirm: onReject,
    }));

    await screen.findByText("BOL is already in use by shipment SEED-DET-009");
    expect(confirm()).toBeEnabled();
    await user.click(screen.getByRole("button", { name: "Ask the agent to fix it" }));

    expect(await screen.findByRole("heading", { name: "Reject this change?" })).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: /^Reason/ })).toHaveValue(ASKED);
    await user.click(confirm("Reject"));
    await waitFor(() => expect(onReject).toHaveBeenCalledWith(ASKED, expect.any(String)));
  });

  it("writes the reasons into its own reason when the surface sends them nowhere else", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockResolvedValue(refusedOverBOL());
    openDialog({
      confirmLabel: "Reject",
      destructive: true,
      preview: { kind: "proposal", scope: "approver", id: "aprop_1", approving: false },
    });

    await user.click(await screen.findByRole("button", { name: "Ask the agent to fix it" }));

    expect(screen.getByRole("textbox", { name: /^Reason/ })).toHaveValue(ASKED);
  });

  it("names the step's own write when a plan's step would be refused", async () => {
    const user = userEvent.setup();
    const plan = planPreview();
    const steps = plan.steps.map((step, index) =>
      index === 0 ? { ...step, preview: refusedOverBOL() } : step,
    );
    fetchPlanPreview.mockResolvedValue({ ...plan, steps });
    openDialog({
      confirmLabel: "Reject",
      destructive: true,
      preview: { kind: "plan", scope: "approver", id: "apl_1", approving: false },
    });

    await user.click(await screen.findByRole("button", { name: "Ask the agent to fix it" }));

    expect(screen.getByRole("textbox", { name: /^Reason/ })).toHaveValue(ASKED);
  });
});
