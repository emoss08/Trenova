import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AssistantLauncher } from "../assistant-launcher";

afterEach(cleanup);

function renderLauncher(pendingCount: number, writingCount?: number) {
  render(
    <AssistantLauncher pendingCount={pendingCount} writingCount={writingCount} onClick={vi.fn()} />,
  );

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

/**
 * A reply keeps being written after the panel closes. The launcher is the one
 * place that can say so from every page, and it says it the way it says
 * "waiting": in words, standing still.
 */
describe("AssistantLauncher while replies are being written", () => {
  it("says it is writing one reply", () => {
    const button = renderLauncher(0, 1);

    expect(button).toHaveTextContent("Writing");
    expect(button).not.toHaveTextContent(/\d/);
    expect(button).toHaveAccessibleName("Open the assistant, 1 reply is being written");
  });

  it("counts several replies", () => {
    const button = renderLauncher(0, 3);

    expect(button).toHaveTextContent("3");
    expect(button).toHaveTextContent("writing");
    expect(button).toHaveAccessibleName("Open the assistant, 3 replies are being written");
  });

  // Decisions lead: they are the only part that needs the person.
  it("puts waiting decisions ahead of replies being written", () => {
    const button = renderLauncher(2, 1);

    expect(button.textContent).toMatch(/2\s*waiting.*writing/);
    expect(button).toHaveAccessibleName(
      "Open the assistant, 2 changes await your decision, 1 reply is being written",
    );
  });

  it("caps the visible count but tells a screen reader the real one", () => {
    const button = renderLauncher(0, 120);

    expect(button).toHaveTextContent("99+");
    expect(button).toHaveAccessibleName("Open the assistant, 120 replies are being written");
  });

  it("marks the writing state for styling without animating it", () => {
    const button = renderLauncher(0, 1);

    expect(button).toHaveAttribute("data-writing", "true");
    expect(button.querySelector("[class*='animate-']")).toBeNull();
  });

  it("goes quiet again once nothing is being written", () => {
    const button = renderLauncher(0, 0);

    expect(button).toHaveAccessibleName("Open the assistant");
    expect(button).not.toHaveAttribute("data-writing");
  });
});
