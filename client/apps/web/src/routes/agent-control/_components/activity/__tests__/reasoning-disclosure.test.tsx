import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";
import { ReasoningDisclosure } from "../reasoning-disclosure";

afterEach(cleanup);

/**
 * A run read back shows what the model thought apart from what it said,
 * folded into how long it took.
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

  it("draws nothing for an empty thought that has finished", () => {
    const { container } = render(<ReasoningDisclosure text="" />);

    expect(container).toBeEmptyDOMElement();
  });
});
