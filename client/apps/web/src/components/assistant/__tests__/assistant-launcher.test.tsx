import type { AssistantLiveTurn } from "@/types/assistant";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { AssistantLauncher } from "../assistant-launcher";
import { beaconState, type BeaconInput } from "../beacon-state";

afterEach(cleanup);

function liveTurn(threadId: string, threadTitle = "Which loads are stuck?"): AssistantLiveTurn {
  return {
    turnId: `turn_${threadId}`,
    threadId,
    threadTitle,
    origin: "Person",
    startedAt: Math.floor(Date.now() / 1000) - 12,
  };
}

function renderBeacon(input: Partial<BeaconInput> = {}) {
  render(
    <AssistantLauncher
      beacon={beaconState({
        pendingCount: 0,
        liveTurns: [],
        repliedAgentName: null,
        lastAgentName: "Billing exceptions",
        ...input,
      })}
      onClick={vi.fn()}
    />,
  );

  return screen.getByRole("button", { name: /^Open the assistant/ });
}

/**
 * The beacon is on every page in the product, so what it says and when it
 * says it is the whole design. At rest it offers the agent it would ask; it
 * speaks up for a reply being written, a change waiting on the person, and a
 * reply that came while the panel was closed.
 */
describe("AssistantLauncher at rest", () => {
  it("offers the agent last asked and the key that opens it", () => {
    const button = renderBeacon();

    expect(button).toHaveAccessibleName("Open the assistant");
    expect(button).toHaveTextContent("Ask Billing exceptions");
    expect(button).toHaveTextContent("⌘J");
    expect(button).toHaveAttribute("aria-keyshortcuts", "Meta+J");
    expect(button.closest(".as-beacon")).toHaveAttribute("data-mode", "idle");
  });
});

describe("AssistantLauncher with changes waiting", () => {
  it("says how many changes need approval and offers to review them", () => {
    const button = renderBeacon({ pendingCount: 3 });

    expect(button).toHaveTextContent("3 changes need your approval");
    expect(button).toHaveTextContent("Review");
    expect(button.closest(".as-beacon")).toHaveAttribute("data-mode", "pending");
  });

  it("speaks of one change in the singular", () => {
    expect(renderBeacon({ pendingCount: 1 })).toHaveTextContent("1 change needs your approval");
  });

  // The visible text is capped; what a screen reader is told is not, because
  // "99+ changes" is worse than the number.
  it("caps a count that would stretch the corner, but tells a screen reader the real one", () => {
    const button = renderBeacon({ pendingCount: 150 });

    expect(button).toHaveTextContent("99+ changes");
    expect(button).toHaveAccessibleName("Open the assistant, 150 changes await your decision");
  });

  // Decisions lead: they are the only part that needs the person.
  it("puts waiting changes ahead of a reply being written", () => {
    const button = renderBeacon({ pendingCount: 2, liveTurns: [liveTurn("athr_1")] });

    expect(button).toHaveTextContent("2 changes need your approval");
    expect(button).not.toHaveTextContent("Which loads are stuck?");
    expect(button).toHaveAccessibleName(
      "Open the assistant, 2 changes await your decision, 1 reply is being written",
    );
  });
});

describe("AssistantLauncher while replies are being written", () => {
  it("says what one reply is answering and for how long", () => {
    const button = renderBeacon({ liveTurns: [liveTurn("athr_1")] });

    expect(button).toHaveTextContent("Which loads are stuck?");
    expect(button).toHaveTextContent(/\d+s/);
    expect(button).toHaveAttribute("data-writing", "true");
    expect(button).toHaveAccessibleName("Open the assistant, 1 reply is being written");
  });

  it("counts several conversations writing at once", () => {
    const button = renderBeacon({ liveTurns: [liveTurn("athr_1"), liveTurn("athr_2")] });

    expect(button).toHaveTextContent("2 writing");
    expect(button).toHaveAccessibleName("Open the assistant, 2 replies are being written");
  });

  it("goes quiet again once nothing is being written", () => {
    const button = renderBeacon();

    expect(button).not.toHaveAttribute("data-writing");
  });
});

describe("AssistantLauncher after a reply came in", () => {
  it("names who replied while the panel was closed", () => {
    const button = renderBeacon({ repliedAgentName: "Dispatch desk" });

    expect(button).toHaveTextContent("Dispatch desk replied");
    expect(button).toHaveAccessibleName("Open the assistant, Dispatch desk replied");
    expect(button.closest(".as-beacon")).toHaveAttribute("data-mode", "replied");
  });
});
