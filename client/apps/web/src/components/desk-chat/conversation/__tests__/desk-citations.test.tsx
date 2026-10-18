import type { ToolStep } from "@/components/assistant/activity";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it } from "vitest";
import { CITATION_HREF, type Citation } from "../citations";
import { useCitationOverrides } from "../desk-citations";

afterEach(cleanup);

function step(reasoning: string): ToolStep {
  return {
    id: "call_1",
    name: "get_shipment",
    arguments: {},
    status: "done",
    content: "",
    effect: "lookup",
    summary: "In transit",
    durationSeconds: 1.2,
    reasoning,
  };
}

function Mark({ citation }: { citation: Citation }): ReactNode {
  const overrides = useCitationOverrides([citation], [], () => {});
  const Link = overrides?.a;
  if (!Link || typeof Link === "string") return null;
  return <Link href={`${CITATION_HREF}1`}>1</Link>;
}

function openMark(reasoning: string) {
  render(<Mark citation={{ n: 1, offset: 0, step: step(reasoning) }} />);
  fireEvent.mouseEnter(screen.getByRole("button", { name: "Step 1" }).parentElement!);
}

/**
 * A citation's card shows the step's own reasoning: what the model thought in
 * the reply that asked for the call, folded until the reader opens it. A step
 * whose reply returned no readable thinking offers nothing to open.
 */
describe("a citation's step card", () => {
  it("folds the model's reasoning for the step under the step", () => {
    openMark("S1 is the load the person named; look it up first.");

    const fold = screen.getByRole("button", { name: "Thought it through" });
    expect(screen.queryByText("S1 is the load the person named; look it up first.")).toBeNull();

    fireEvent.click(fold);

    expect(
      screen.getByText("S1 is the load the person named; look it up first."),
    ).toBeInTheDocument();
  });

  it("offers no reasoning when the model returned none", () => {
    openMark("");

    expect(screen.getByRole("tooltip")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Thought it through" })).toBeNull();
  });
});
