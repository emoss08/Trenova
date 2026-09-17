import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import type { AssistantProposal } from "@/types/assistant";
import { ProposalCard } from "../proposal-card";

afterEach(cleanup);

function proposal(overrides: Partial<AssistantProposal> = {}): AssistantProposal {
  return {
    id: "aprop_1",
    runId: "arun_1",
    threadId: "athr_1",
    sourceMessageId: "amsg_1",
    toolName: "flag_for_manual_review",
    arguments: {
      runId: "ar_01M2PRQ1C0QN91R9TNRXSSSFZ5",
      subjectId: "shp_01M2PRNXAMQNKK9HK9V5B817QE",
      category: "MissingBOL",
      severity: "Medium",
      blastRadius: 1,
      attemptSummary: "Signed bill of lading is missing.",
      evidence: [{ id: "shp_01M2PRNXAMQNKK9HK9V5B817QE", type: "Shipment" }],
    },
    rationale: "The item cannot be billed until a person resolves the rate.",
    confidence: 0.82,
    autonomyTier: "Propose",
    status: "Pending",
    executedAt: null,
    executionError: "",
    createdAt: 1,
    updatedAt: 1,
    ...overrides,
  } as AssistantProposal;
}

function renderCard(value = proposal()) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });

  return render(
    <QueryClientProvider client={client}>
      <ProposalCard proposal={value} threadId="athr_1" />
    </QueryClientProvider>,
  );
}

/**
 * The card asks one question. Everything it shows has to bear on answering it.
 *
 * The disclosure it used to carry printed the run id, the subject's PULID and a
 * raw evidence blob, then repeated the category and severity the sentence above
 * already said — and pushed Approve and Reject below the fold while it was
 * open.
 */
describe("ProposalCard", () => {
  it("offers no way to expand into the raw payload", () => {
    renderCard();

    expect(screen.queryByText("Details")).not.toBeInTheDocument();
  });

  it("keeps the identifiers and the evidence blob off the card", () => {
    const { container } = renderCard();
    const text = container.textContent ?? "";

    expect(text).not.toContain("ar_01M2PRQ1C0QN91R9TNRXSSSFZ5");
    expect(text).not.toContain("shp_01M2PRNXAMQNKK9HK9V5B817QE");
    expect(text).not.toContain('"type":"Shipment"');
  });

  it("leads with the sentence and the two decisions", () => {
    renderCard();

    expect(
      screen.getByText("Record a missing BOL case for a person to resolve."),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /approve/i })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /reject/i })).toBeInTheDocument();
  });

  it("collapses to one line once a decision has been made", () => {
    renderCard(proposal({ status: "Rejected" }));

    expect(screen.queryByRole("button", { name: /approve/i })).not.toBeInTheDocument();
    expect(screen.getByText("Rejected. Nothing was changed.")).toBeInTheDocument();
  });
});
