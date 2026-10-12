import { DeskWriteResultCard } from "@/components/desk-chat/conversation/desk-tool-failures";
import type { AssistantProposal } from "@/types/assistant";
import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

type Proposal = Pick<AssistantProposal, "status" | "executionError" | "executionResult">;

// What the server records for a bulk write: its action and record kind are English words
// the agent reads back, not text for a person in another language.
function proposal(failedCount: number, total: number): Proposal {
  return {
    status: "Executed",
    executionError: "",
    executionResult: {
      action: "updated",
      kind: "shipments",
      name: "",
      ids: {},
      record: null,
      total,
      failed: Array.from({ length: failedCount }, (_, index) => ({
        id: `shp_${index}`,
        label: `S-10${index}`,
        reason: "Shipment is locked",
      })),
    },
  } as Proposal;
}

describe("DeskWriteResultCard", () => {
  it("counts what went through without the server's English words", () => {
    const { container } = render(<DeskWriteResultCard proposal={proposal(2, 5)} />);

    expect(screen.getByText("3 of 5 went through · 2 didn't")).toBeInTheDocument();
    expect(
      screen.getByText(
        "The 2 that didn't go through stayed as they were. The 3 that went through are final.",
      ),
    ).toBeInTheDocument();
    expect(container.textContent).not.toMatch(/shipments|updated/i);
  });

  it("speaks of one record in the singular", () => {
    render(<DeskWriteResultCard proposal={proposal(1, 2)} />);

    expect(
      screen.getByText(
        "The one that didn't go through stayed as it was. The one that went through is final.",
      ),
    ).toBeInTheDocument();
  });

  it("shows nothing when every record went through", () => {
    const { container } = render(<DeskWriteResultCard proposal={proposal(0, 4)} />);
    expect(container).toBeEmptyDOMElement();
  });
});
