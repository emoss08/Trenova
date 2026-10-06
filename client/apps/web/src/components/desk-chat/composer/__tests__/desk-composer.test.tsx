import type { ComposerAttachment, ComposerPayload } from "@/components/assistant/composer-types";
import type { AssistantEntityRef } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { MemoryRouter } from "react-router";
import { afterEach, describe, expect, it, vi } from "vitest";
import { useDeskAttachments } from "../desk-attachments";
import { DeskComposer } from "../desk-composer";

afterEach(cleanup);

const ready: ComposerAttachment = {
  id: "up_1",
  name: "rate-con.pdf",
  size: 2048,
  status: "ready",
  progress: 1,
  documentId: "doc_1",
  contentType: "application/pdf",
};
const uploading: ComposerAttachment = { ...ready, id: "up_2", status: "uploading", progress: 0.4 };

const shipment: AssistantEntityRef = { type: "shipment", id: "shp_1", label: "PRO-1001" };
const customer: AssistantEntityRef = { type: "customer", id: "cus_1", label: "Acme" };

function Harness({
  busy = false,
  files = [],
  mentions = [],
  initial = "",
  onSend,
  onStop,
}: {
  busy?: boolean;
  files?: ComposerAttachment[];
  mentions?: AssistantEntityRef[];
  initial?: string;
  onSend: (content: string, payload: ComposerPayload) => void;
  onStop?: () => void;
}) {
  const [value, setValue] = useState(initial);
  const attachments = useDeskAttachments({
    attachments: files,
    attachFiles: () => undefined,
    removeAttachment: () => undefined,
    retryAttachment: () => undefined,
  });

  return (
    <DeskComposer
      value={value}
      onChange={setValue}
      onSend={onSend}
      onStop={onStop}
      agent={null}
      busy={busy}
      attachments={attachments}
      mentions={mentions}
      onMentionsChange={() => undefined}
    />
  );
}

function renderComposer(props: Parameters<typeof Harness>[0]) {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <Harness {...props} />
      </MemoryRouter>
    </QueryClientProvider>,
  );

  return screen.getByRole("textbox", { name: "Message the Desk" });
}

/**
 * Ported from the assistant's old composer: the cases that hold for the one
 * composer both surfaces now share.
 */
describe("DeskComposer while a reply is being written", () => {
  it("turns Send into Stop", () => {
    const onStop = vi.fn();
    renderComposer({ busy: true, onSend: vi.fn(), onStop });

    expect(screen.queryByRole("button", { name: "Send" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Stop the reply" }));
    expect(onStop).toHaveBeenCalledTimes(1);
  });

  it("holds Enter, keeping what was typed for when the reply is done", () => {
    const onSend = vi.fn();
    const box = renderComposer({ busy: true, initial: "And the next one?", onSend });

    fireEvent.keyDown(box, { key: "Enter" });

    expect(onSend).not.toHaveBeenCalled();
    expect(box).toHaveValue("And the next one?");
  });
});

describe("DeskComposer sending", () => {
  it("sends on Enter and breaks a line on Shift+Enter", () => {
    const onSend = vi.fn();
    const box = renderComposer({ initial: "Where is PRO-1001?", onSend });

    fireEvent.keyDown(box, { key: "Enter", shiftKey: true });
    expect(onSend).not.toHaveBeenCalled();

    fireEvent.keyDown(box, { key: "Enter" });
    expect(onSend).toHaveBeenCalledWith("Where is PRO-1001?", { attachments: [], mentions: [] });
    expect(box).toHaveValue("");
  });

  it("sends the ready documents with the message", () => {
    const onSend = vi.fn();
    renderComposer({ initial: "What does this say?", files: [ready], onSend });

    fireEvent.click(screen.getByRole("button", { name: "Send" }));

    expect(onSend).toHaveBeenCalledWith("What does this say?", {
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

  it("holds the send until every file has uploaded", () => {
    const onSend = vi.fn();
    const box = renderComposer({ initial: "Read these", files: [ready, uploading], onSend });

    expect(screen.getByRole("button", { name: "Send" })).toBeDisabled();
    fireEvent.keyDown(box, { key: "Enter" });
    expect(onSend).not.toHaveBeenCalled();
  });

  it("sends only the records still named in the text", () => {
    const onSend = vi.fn();
    renderComposer({
      initial: "Is @PRO-1001 late?",
      mentions: [shipment, customer],
      onSend,
    });

    fireEvent.click(screen.getByRole("button", { name: "Send" }));

    expect(onSend).toHaveBeenCalledWith("Is @PRO-1001 late?", {
      attachments: [],
      mentions: [shipment],
    });
  });

  it("asks what is in a file sent without words", () => {
    const onSend = vi.fn();
    renderComposer({ files: [ready], onSend });

    fireEvent.click(screen.getByRole("button", { name: "Send" }));

    expect(onSend).toHaveBeenCalledWith("What's in this?", expect.anything());
  });
});
