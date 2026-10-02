import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { ReasoningDisclosure, UserTurn } from "../message-items";

afterEach(cleanup);

/**
 * A person's message sits on the right, as a bubble, and the side says who
 * is speaking; the reply runs open on the left under the agent's mark. The
 * word "You" went with the alignment, and the time waits behind a hover.
 */
describe("UserTurn", () => {
  it("sits on the right without a label, and keeps the time and chips under it", () => {
    render(
      <UserTurn
        content="Where is PRO-1001?"
        sentAt={1_790_000_000}
        attachments={[
          {
            documentId: "doc_1",
            fileName: "bol.pdf",
            contentType: "application/pdf",
            fileSize: 10,
          },
        ]}
        mentions={[{ type: "shipment", id: "shp_1", label: "PRO-1001" }]}
      />,
    );

    const turn = screen.getByRole("article");
    expect(turn).toHaveAttribute("data-side", "right");
    expect(screen.queryByText("You")).toBeNull();
    expect(screen.getByText("Where is PRO-1001?")).toBeInTheDocument();
    expect(screen.getByText("bol.pdf")).toBeInTheDocument();
    expect(screen.getByText("PRO-1001")).toBeInTheDocument();
    expect(turn.querySelector("time")).not.toBeNull();
  });
});

/**
 * The thought folds into how long it took, so a reader knows the wait was
 * the model's and not the network's; a saved thought nobody timed keeps
 * the old words.
 */
describe("ReasoningDisclosure", () => {
  it("says how long the thinking took once it is over", () => {
    render(<ReasoningDisclosure text="Checking the dates." seconds={4} />);

    expect(screen.getByRole("button", { name: "Thought for 4s" })).toBeInTheDocument();
  });

  it("keeps the plain words for a thought nobody timed", () => {
    render(<ReasoningDisclosure text="Checking the dates." />);

    expect(screen.getByRole("button", { name: "Thought it through" })).toBeInTheDocument();
  });

  it("says it is thinking while the thought still arrives", () => {
    render(<ReasoningDisclosure text="Checking" streaming />);

    expect(screen.getByRole("button", { name: "Thinking…" })).toBeInTheDocument();
  });
});
