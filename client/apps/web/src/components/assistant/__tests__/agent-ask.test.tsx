import type { AgentChoice } from "@/lib/graphql/agent-definition";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AgentAsk } from "../agent-ask";

vi.mock("@/components/assistant/agent-picker", () => ({
  AgentPicker: ({ agent }: { agent: { name: string } }) => <span>{agent.name}</span>,
}));

const agent = {
  id: "agdef_billing",
  name: "Billing Specialist",
  starters: [
    { label: "Oldest blocked", prompt: "Which billing items are stuck the longest?" },
    { label: "Past due", prompt: "Which invoices are past due?" },
  ],
} as unknown as AgentChoice;

function renderAsk(onAsk = vi.fn()) {
  render(<AgentAsk agent={agent} onAgentChange={() => undefined} recentIds={[]} onAsk={onAsk} />);
  return onAsk;
}

/** Long enough for the first starter to be written out in full. */
const WRITTEN_MS = 2400;

/** Moves the clock a little at a time with a flush between, as a browser would. */
function pass(ms: number) {
  for (let elapsed = 0; elapsed < ms; elapsed += 20) {
    act(() => {
      vi.advanceTimersByTime(20);
    });
  }
}

/**
 * The starter questions are written into the empty box rather than listed
 * under it, and Tab takes the one on screen as the person's own: it fills
 * the field without sending, so it can still be changed, and typing a first
 * letter of one's own puts the suggestion away.
 */
describe("the ask box's typed starters", () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it("writes the agent's first question into the empty box and takes it on Tab", () => {
    const onAsk = renderAsk();
    const field = screen.getByRole("textbox", { name: "Ask Billing Specialist…" });

    pass(WRITTEN_MS);
    const ghost = document.querySelector('[data-slot="starter-ghost"]');
    expect(ghost).toHaveTextContent("Which billing items are stuck the longest?");
    expect(screen.getByText("to ask this")).toBeInTheDocument();

    fireEvent.keyDown(field, { key: "Tab" });
    expect(field).toHaveValue("Which billing items are stuck the longest?");
    expect(onAsk).not.toHaveBeenCalled();
    expect(document.querySelector('[data-slot="starter-ghost"]')).toBeNull();
  });

  it("leaves Tab alone while the question is still being written or the box has words", () => {
    renderAsk();
    const field = screen.getByRole("textbox", { name: "Ask Billing Specialist…" });

    pass(500);
    const partial = document.querySelector('[data-slot="starter-ghost"]')?.textContent ?? "";
    expect(partial.length).toBeGreaterThan(0);
    expect(partial.length).toBeLessThan("Which billing items are stuck the longest?".length);
    const held = fireEvent.keyDown(field, { key: "Tab" });
    expect(held).toBe(true);
    expect(field).toHaveValue("");

    fireEvent.change(field, { target: { value: "Where" } });
    expect(document.querySelector('[data-slot="starter-ghost"]')).toBeNull();
    pass(WRITTEN_MS);
    expect(document.querySelector('[data-slot="starter-ghost"]')).toBeNull();
  });
});
