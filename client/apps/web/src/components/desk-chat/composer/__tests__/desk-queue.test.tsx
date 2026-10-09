import type { ConversationQueue } from "@/components/assistant/use-conversation-queue";
import type { QueuedMessage } from "@/types/assistant";
import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DeskQueue } from "../desk-queue";

afterEach(cleanup);

function item(overrides: Partial<QueuedMessage>): QueuedMessage {
  return {
    id: "aqm_1",
    threadId: "athr_1",
    content: "Then bill it.",
    request: { mentions: [], attachmentDocumentIds: [] },
    position: 1,
    steer: false,
    version: 0,
    createdAt: 0,
    ...overrides,
  };
}

function queueOf(items: QueuedMessage[]): ConversationQueue {
  return {
    items,
    add: vi.fn(async () => true),
    edit: vi.fn(async () => true),
    remove: vi.fn(async () => undefined),
    move: vi.fn(async () => undefined),
    sendNow: vi.fn(async () => undefined),
  };
}

describe("DeskQueue", () => {
  it("shows nothing when nothing waits", () => {
    const { container } = render(<DeskQueue queue={queueOf([])} busy />);

    expect(container).toBeEmptyDOMElement();
  });

  it("lists the waiting messages in order and lets one be moved, sent or removed", () => {
    const first = item({ id: "aqm_1", content: "Then bill it." });
    const second = item({ id: "aqm_2", content: "Email Acme.", position: 2 });
    const queue = queueOf([first, second]);
    render(<DeskQueue queue={queue} busy />);

    const rows = screen.getAllByRole("listitem");
    expect(rows.map((row) => row.querySelector("p")?.textContent)).toEqual([
      "Then bill it.",
      "Email Acme.",
    ]);
    expect(within(rows[0]).getByRole("button", { name: "Move the message up" })).toBeDisabled();
    expect(within(rows[1]).getByRole("button", { name: "Move the message down" })).toBeDisabled();

    fireEvent.click(within(rows[1]).getByRole("button", { name: "Move the message up" }));
    expect(queue.move).toHaveBeenCalledWith(second, -1);
    fireEvent.click(within(rows[0]).getByRole("button", { name: "Steer the reply with it now" }));
    expect(queue.sendNow).toHaveBeenCalledWith(first);
    fireEvent.click(within(rows[1]).getByRole("button", { name: "Remove the waiting message" }));
    expect(queue.remove).toHaveBeenCalledWith(second);
  });

  it("leaves a message handed to the reply under way alone", () => {
    render(<DeskQueue queue={queueOf([item({ steer: true })])} busy />);

    const row = screen.getByRole("listitem");
    expect(
      within(row).getByTitle("Steering: the agent reads it at its next step"),
    ).toBeInTheDocument();
    expect(within(row).queryByRole("button")).toBeNull();
  });

  it("offers no steer for a message with files while a reply runs", () => {
    render(
      <DeskQueue
        queue={queueOf([item({ request: { mentions: [], attachmentDocumentIds: ["doc_1"] } })])}
        busy
      />,
    );

    expect(screen.queryByRole("button", { name: "Steer the reply with it now" })).toBeNull();
    expect(screen.getByText("1 file")).toBeInTheDocument();
  });

  it("says a queue held after a stopped reply is waiting, and sends the next on request", () => {
    const first = item({});
    const queue = queueOf([first, item({ id: "aqm_2", position: 2 })]);
    render(<DeskQueue queue={queue} busy={false} />);

    expect(screen.getByText("2 messages are waiting")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Send next" }));
    expect(queue.sendNow).toHaveBeenCalledWith(first);
  });

  it("edits a waiting message in place, saving on Enter and keeping it on Escape", () => {
    const waiting = item({});
    const queue = queueOf([waiting]);
    render(<DeskQueue queue={queue} busy />);

    fireEvent.click(screen.getByRole("button", { name: "Edit the waiting message" }));
    const box = screen.getByRole("textbox", { name: "Edit the waiting message" });
    fireEvent.change(box, { target: { value: "Then bill it and email Acme." } });
    fireEvent.keyDown(box, { key: "Enter" });

    expect(queue.edit).toHaveBeenCalledWith(waiting, "Then bill it and email Acme.");
  });

  it("folds a long queue away and brings a newly added message into view", () => {
    const many = Array.from({ length: 8 }, (_, index) =>
      item({ id: `aqm_${index}`, position: index + 1 }),
    );
    const { rerender } = render(<DeskQueue queue={queueOf(many.slice(0, 7))} busy />);
    const list = screen.getByRole("list");
    Object.defineProperty(list, "scrollHeight", { configurable: true, value: 640 });

    rerender(<DeskQueue queue={queueOf(many)} busy />);
    expect(list.scrollTop).toBe(640);

    fireEvent.click(screen.getByRole("button", { name: "Hide the waiting messages" }));
    expect(screen.queryByRole("list")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Show the waiting messages" }));
    expect(screen.getByRole("list")).toBeInTheDocument();
  });
});
