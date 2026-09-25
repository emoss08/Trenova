import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { planPreview, preview } from "@/components/assistant/__tests__/preview-fixtures";
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
