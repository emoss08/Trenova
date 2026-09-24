import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { dictationScope, FakeRecognition, refusedMicrophone } from "./__tests__/dictation-fakes";
import { Composer } from "./composer";
import type { DictationScope } from "./use-dictation";

afterEach(cleanup);

beforeEach(() => {
  FakeRecognition.instances = [];
});

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

/** A composer that owns its draft, as the thread does, so dictation can be read back from the box. */
function DictationHarness({ scope }: { scope: DictationScope }) {
  const [draft, setDraft] = useState("");

  return (
    <Composer
      onSend={() => {}}
      onStop={() => {}}
      active={false}
      placeholder="Message Dispatch desk…"
      draft={draft}
      onDraftChange={setDraft}
      dictationScope={scope}
    />
  );
}

/**
 * "I click on the microphone and nothing happens." Every way a click could
 * end in silence now ends in something a person can see: words in the box,
 * a reason under it, or a control that says up front why it cannot be used.
 */
describe("Composer dictation", () => {
  it("keeps the mic in the row where this browser cannot dictate, disabled rather than dead", () => {
    renderComposer({ dictationScope: dictationScope({ recognition: null }) });

    expect(screen.getByRole("button", { name: "Dictate" })).toHaveAttribute(
      "aria-disabled",
      "true",
    );
  });

  it("writes what is heard into the box as it is heard, and stops on Esc", () => {
    render(<DictationHarness scope={dictationScope()} />);

    fireEvent.click(screen.getByRole("button", { name: "Dictate" }));
    act(() =>
      FakeRecognition.latest().emitResults([{ transcript: "where is load", isFinal: false }]),
    );

    expect(screen.getByRole("textbox")).toHaveValue("where is load");
    expect(screen.getByRole("button", { name: "Stop dictating" })).toHaveAttribute(
      "aria-pressed",
      "true",
    );

    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Escape" });
    expect(FakeRecognition.latest().stop).toHaveBeenCalled();
  });

  it("says why under the box when the microphone is refused", async () => {
    render(
      <DictationHarness
        scope={dictationScope({ getUserMedia: refusedMicrophone("NotAllowedError").getUserMedia })}
      />,
    );

    fireEvent.click(screen.getByRole("button", { name: "Dictate" }));

    expect(
      await screen.findByText("Microphone blocked. Allow it in the address bar."),
    ).toBeInTheDocument();
    expect(FakeRecognition.instances).toHaveLength(0);
  });

  it("stops writing into the box once the person types", () => {
    render(<DictationHarness scope={dictationScope()} />);

    fireEvent.click(screen.getByRole("button", { name: "Dictate" }));
    act(() => FakeRecognition.latest().emitResults([{ transcript: "hello", isFinal: false }]));
    fireEvent.change(screen.getByRole("textbox"), { target: { value: "hello, typed" } });
    act(() => FakeRecognition.latest().emitResults([{ transcript: "hello world", isFinal: true }]));

    expect(FakeRecognition.latest().abort).toHaveBeenCalled();
    expect(screen.getByRole("textbox")).toHaveValue("hello, typed");
  });
});

/**
 * The footer used to list every key at once and wrapped into ragged lines
 * in the corner panel. It now says one thing, for what is happening.
 */
describe("Composer hints", () => {
  it("teaches the slash and the at sign on an empty, focused box, and nothing about sending", () => {
    renderComposer({ onSearchMentions: async () => [] });

    fireEvent.focus(screen.getByRole("textbox"));

    expect(screen.getByText(/for commands/)).toBeInTheDocument();
    expect(screen.getByText(/for records/)).toBeInTheDocument();
    expect(screen.queryByText(/to send/)).toBeNull();
  });

  it("says how to send once there is something to send", () => {
    renderComposer({ draft: "Where is PRO 1234?" });

    fireEvent.focus(screen.getByRole("textbox"));

    expect(screen.getByText(/to send/)).toBeInTheDocument();
    expect(screen.queryByText(/for commands/)).toBeNull();
  });

  it("says nothing while the box is not in use", () => {
    renderComposer({ draft: "Where is PRO 1234?" });

    expect(screen.queryByText(/to send/)).toBeNull();
    expect(screen.queryByText(/for commands/)).toBeNull();
  });

  it("keeps the full list of keys behind the shortcuts button", async () => {
    renderComposer();

    fireEvent.click(screen.getByRole("button", { name: "Keyboard shortcuts" }));

    expect(await screen.findByText("New line")).toBeInTheDocument();
  });
});

describe("Composer while a reply is being written", () => {
  it("turns Send into Stop and says the next message can wait in the box", () => {
    const onStop = vi.fn();
    renderComposer({ active: true, draft: "and then?", onStop });

    expect(screen.getByText("Replying. You can draft your next message.")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Send" })).toBeNull();

    fireEvent.click(screen.getByRole("button", { name: "Stop the reply" }));
    expect(onStop).toHaveBeenCalledTimes(1);
  });

  it("holds Enter while the reply is being written", async () => {
    const { onSend } = renderComposer({ active: true, draft: "and then?" });

    fireEvent.keyDown(screen.getByRole("textbox"), { key: "Enter" });

    await waitFor(() => expect(onSend).not.toHaveBeenCalled());
  });
});
