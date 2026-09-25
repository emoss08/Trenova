import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { GraphQLRequestError } from "@trenova/shared/lib/graphql";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { ProposalPreview } from "@/lib/graphql/agent-preview";
import type { AssistantProposal } from "@/types/assistant";
import { ProposalCard } from "../proposal-card";
import { preview } from "./preview-fixtures";

const fetchProposalPreview = vi.fn<(...args: unknown[]) => Promise<ProposalPreview>>();
const decideMyProposal = vi.fn<(...args: unknown[]) => Promise<unknown>>();

vi.mock("@/lib/graphql/agent-preview", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/graphql/agent-preview")>()),
  fetchProposalPreview: (...args: unknown[]) => fetchProposalPreview(...args),
}));

vi.mock("@/lib/graphql/agent-decisions", () => ({
  decideMyProposal: (...args: unknown[]) => decideMyProposal(...args),
}));

afterEach(() => {
  fetchProposalPreview.mockReset();
  decideMyProposal.mockReset();
});

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
    ...overrides,
  };
}

function renderCard(value = proposal()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ProposalCard proposal={value} threadId="athr_1" />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function conflict() {
  const type = "https://trenova.app/problems/resource-conflict";
  return new GraphQLRequestError({
    kind: "graphql",
    message: "The change looks different now",
    status: 200,
    graphQLErrors: [{ message: "The change looks different now", extensions: { type }, type }],
  });
}

const approve = () => screen.getByRole("button", { name: /^approve$/i });
const reject = () => screen.getByRole("button", { name: /^reject$/i });

describe("ProposalCard with a preview", () => {
  it("reads the preview as the person whose conversation raised it", async () => {
    fetchProposalPreview.mockResolvedValue(preview());
    renderCard();

    await screen.findByRole("link", { name: "S-1001" });
    expect(fetchProposalPreview).toHaveBeenCalledWith(
      { scope: "mine", id: "aprop_1", modifications: null },
      expect.anything(),
    );
  });

  // Nothing is on screen yet, so there is nothing to approve. Reject needs
  // no preview: turning a change down is safe whatever it would have done.
  it("keeps Approve off until the preview is in, and Reject on", () => {
    fetchProposalPreview.mockReturnValue(new Promise(() => {}));
    renderCard();

    expect(approve()).toBeDisabled();
    expect(reject()).toBeEnabled();
    expect(screen.getByLabelText("Loading what would change")).toBeInTheDocument();
  });

  it("approves with the digest of the preview on screen", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockResolvedValue(preview({ digest: "sha256:shown" }));
    decideMyProposal.mockResolvedValue({});
    renderCard();

    await screen.findByRole("link", { name: "S-1001" });
    await user.click(approve());

    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenCalledWith("aprop_1", {
        decision: "Accepted",
        reasonCode: "",
        previewDigest: "sha256:shown",
      }),
    );
  });

  // The server refuses a stale approval; the card does not offer one.
  it("turns Approve off for a record that changed since, leaving Reject", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockResolvedValue(preview({ stale: true, digest: "sha256:stale" }));
    decideMyProposal.mockResolvedValue({});
    renderCard();

    await screen.findByText("Changed since it was proposed");
    expect(approve()).toBeDisabled();
    expect(reject()).toBeEnabled();

    await user.click(reject());
    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenCalledWith("aprop_1", {
        decision: "Rejected",
        reasonCode: "",
        previewDigest: "sha256:stale",
      }),
    );
  });

  // Owner decision: a digest mismatch records nothing. The card reads the
  // preview again, says the change looks different now, and the person
  // approves again against what is there now.
  it("reads the preview again after a digest conflict, says so, and approves the new one", async () => {
    const user = userEvent.setup();
    fetchProposalPreview
      .mockResolvedValueOnce(preview({ digest: "sha256:before" }))
      .mockResolvedValue(preview({ digest: "sha256:after" }));
    decideMyProposal.mockRejectedValueOnce(conflict()).mockResolvedValue({});
    renderCard();

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

  it("still allows approving without a digest when the preview cannot be read, and says so", async () => {
    const user = userEvent.setup();
    fetchProposalPreview.mockRejectedValue(new Error("network down"));
    decideMyProposal.mockResolvedValue({});
    renderCard();

    expect(
      await screen.findByText(
        "What this would change could not be loaded. Approving now is recorded as approved without a preview.",
      ),
    ).toBeInTheDocument();
    await user.click(approve());

    await waitFor(() =>
      expect(decideMyProposal).toHaveBeenCalledWith("aprop_1", {
        decision: "Accepted",
        reasonCode: "",
        previewDigest: undefined,
      }),
    );
  });

  it("reads no preview for a proposal already decided", () => {
    renderCard(proposal({ status: "Rejected" }));

    expect(fetchProposalPreview).not.toHaveBeenCalled();
  });
});
