import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AssistantLauncher } from "../assistant-launcher";

afterEach(cleanup);

function renderLauncher(pendingCount: number) {
  render(<AssistantLauncher pendingCount={pendingCount} onClick={vi.fn()} />);

  return screen.getByRole("button");
}

/**
 * The launcher is on every page in the product, so what it says and when it
 * says it is the whole design. Quiet by default; a count and a word when
 * something is genuinely waiting on a person.
 */
describe("AssistantLauncher", () => {
  it("says nothing when nothing is waiting", () => {
    const button = renderLauncher(0);

    expect(button).toHaveAccessibleName("Open the assistant");
    expect(button).not.toHaveTextContent(/waiting/);
    expect(button).not.toHaveTextContent(/\d/);
  });

  // A bare number in the corner could be anything. The word is what makes it
  // readable without opening the panel.
  it("names the count and what it is", () => {
    const button = renderLauncher(3);

    expect(button).toHaveTextContent("3");
    expect(button).toHaveTextContent("waiting");
  });

  it("caps a count that would stretch the corner", () => {
    expect(renderLauncher(150)).toHaveTextContent("99+");
  });

  // The visible text is capped; what a screen reader is told is not, because
  // "99+ changes" is worse than the number.
  it("tells a screen reader the real count", () => {
    expect(renderLauncher(150)).toHaveAccessibleName(
      "Open the assistant, 150 changes await your decision",
    );
  });
});
