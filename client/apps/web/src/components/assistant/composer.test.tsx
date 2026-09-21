import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { Composer } from "./composer";

afterEach(cleanup);

const suggestions = [
  { label: "Where is a shipment right now?", prompt: "Where is PRO S12345 right now?" },
  { label: "What is picking up today?", prompt: "Which shipments pick up today?" },
];

function renderComposer(overrides: Partial<React.ComponentProps<typeof Composer>> = {}) {
  const onSend = vi.fn();
  const onDraftChange = vi.fn();
  const view = render(
    <Composer
      onSend={onSend}
      onStop={() => {}}
      active={false}
      placeholder="Message Dispatch desk…"
      draft=""
      onDraftChange={onDraftChange}
      suggestions={suggestions}
      {...overrides}
    />,
  );

  return { ...view, onSend, onDraftChange };
}

/**
 * The starter questions used to sit in the toolbar under the box as chips on
 * every turn. They now come when asked for: a draft that opens with a slash
 * lists them, narrowed by what follows, and Enter sends the highlighted one.
 */
describe("Composer slash commands", () => {
  it("lists the starter questions when the draft opens with a slash", () => {
    renderComposer({ draft: "/" });

    expect(screen.getByRole("listbox", { name: /starter questions/i })).toBeInTheDocument();
    expect(screen.getAllByRole("option")).toHaveLength(2);
  });

  it("narrows the list by what follows the slash", () => {
    renderComposer({ draft: "/pick" });

    const options = screen.getAllByRole("option");
    expect(options).toHaveLength(1);
    expect(options[0]).toHaveTextContent("What is picking up today?");
  });

  it("sends the highlighted question on Enter instead of the slash text", () => {
    const { onSend } = renderComposer({ draft: "/pick" });

    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });

    expect(onSend).toHaveBeenCalledWith("Which shipments pick up today?");
  });

  it("shows no list for an ordinary draft", () => {
    renderComposer({ draft: "what about 12/24?" });

    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });
});
