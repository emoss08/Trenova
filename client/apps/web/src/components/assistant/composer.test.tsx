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
    // The four commands, then the agent's two questions.
    expect(screen.getAllByRole("option")).toHaveLength(6);
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

    expect(onSend).toHaveBeenCalledWith("Which shipments pick up today?", {
      attachments: [],
      mentions: [],
    });
  });

  it("shows no list for an ordinary draft", () => {
    renderComposer({ draft: "what about 12/24?" });

    expect(screen.queryByRole("listbox")).not.toBeInTheDocument();
  });
});

/**
 * A slash command with slots is filled in the box: choosing `/status` leaves
 * `/status ` in the draft with the slot's hint beside it, and Enter sends the
 * command's full question once the slot has something in it.
 */
describe("Composer slash commands with slots", () => {
  it("lists the commands ahead of the questions and fills the draft with the chosen one", () => {
    const { onDraftChange, onSend } = renderComposer({ draft: "/" });

    const options = screen.getAllByRole("option");
    expect(options[0]).toHaveTextContent("/status");

    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });

    expect(onSend).not.toHaveBeenCalled();
    expect(onDraftChange).toHaveBeenCalledWith("/status ");
  });

  it("shows the empty slot as a hint and sends the filled question on Enter", () => {
    const empty = renderComposer({ draft: "/status " });
    expect(screen.getByText(/PRO or shipment number/)).toBeInTheDocument();
    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });
    expect(empty.onSend).not.toHaveBeenCalled();
    cleanup();

    const filled = renderComposer({ draft: "/status S12345" });
    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });
    expect(filled.onSend).toHaveBeenCalledWith(
      "What is the status of shipment S12345 right now, and is anything holding it up?",
      expect.objectContaining({ attachments: [], mentions: [] }),
    );
  });
});

describe("Composer attachments", () => {
  it("shows each file as a chip and holds the send until every file is ready", () => {
    const onRemoveAttachment = vi.fn();
    const { onSend } = renderComposer({
      draft: "Read this",
      attachments: [
        { id: "u1", name: "rate-con.pdf", size: 2048, status: "uploading", progress: 40 },
      ],
      onAttachFiles: () => {},
      onRemoveAttachment,
    });

    expect(screen.getByText("rate-con.pdf")).toBeInTheDocument();
    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });
    expect(onSend).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole("button", { name: /remove rate-con.pdf/i }));
    expect(onRemoveAttachment).toHaveBeenCalledWith("u1");
  });

  it("sends the ready documents with the message", () => {
    const { onSend } = renderComposer({
      draft: "Read this",
      attachments: [
        {
          id: "u1",
          name: "rate-con.pdf",
          size: 2048,
          status: "ready",
          progress: 100,
          documentId: "doc_1",
          contentType: "application/pdf",
        },
      ],
      onAttachFiles: () => {},
      onRemoveAttachment: () => {},
    });

    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });

    expect(onSend).toHaveBeenCalledWith("Read this", {
      attachments: [
        {
          documentId: "doc_1",
          fileName: "rate-con.pdf",
          contentType: "application/pdf",
          fileSize: 2048,
        },
      ],
      mentions: [],
    });
  });
});

describe("Composer mentions", () => {
  it("sends only the mentions still named in the text", () => {
    const { onSend } = renderComposer({
      draft: "Tell me about @Acme Foods",
      mentions: [
        { type: "customer", id: "cust_1", label: "Acme Foods" },
        { type: "worker", id: "wrk_1", label: "Maria Ortiz" },
      ],
      onMentionsChange: () => {},
      onSearchMentions: async () => [],
    });

    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });

    expect(onSend).toHaveBeenCalledWith("Tell me about @Acme Foods", {
      attachments: [],
      mentions: [{ type: "customer", id: "cust_1", label: "Acme Foods" }],
    });
  });
});
