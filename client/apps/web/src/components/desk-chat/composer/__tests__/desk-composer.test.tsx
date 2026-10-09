import type { ComposerAttachment, ComposerPayload } from "@/components/assistant/composer-types";
import type { AssistantEntityRef } from "@/types/assistant";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { MemoryRouter } from "react-router";
import { registerCatalogSource, setLocale, translate } from "@trenova/shared/i18n/runtime";
import { afterAll, afterEach, beforeAll, describe, expect, it, vi } from "vitest";
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
  onSteer,
}: {
  busy?: boolean;
  files?: ComposerAttachment[];
  mentions?: AssistantEntityRef[];
  initial?: string;
  onSend: (content: string, payload: ComposerPayload) => void;
  onStop?: () => void;
  onSteer?: (content: string, payload: ComposerPayload, mode: "steer" | "queue") => void;
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
      onSteer={onSteer}
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

  return screen.getByRole("textbox", { name: translate("Message the Desk") });
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

/**
 * With somewhere to put it, what is typed while the agent works is not held:
 * the send gesture steers the reply under way, Option with it queues the
 * message for after the reply, and files always wait, because a reply under
 * way reads words, not documents.
 */
describe("DeskComposer steering while a reply is being written", () => {
  it("offers Stop while the box is empty", () => {
    renderComposer({ busy: true, initial: "", onSend: vi.fn(), onSteer: vi.fn() });

    expect(screen.getByRole("button", { name: "Stop the reply" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Steer the reply" })).toBeNull();
  });

  it("puts steer in Stop's place once something is typed, and steers on Enter", () => {
    const onSteer = vi.fn();
    const onSend = vi.fn();
    const box = renderComposer({
      busy: true,
      initial: "Actually the carrier is Werner",
      onSend,
      onSteer,
    });

    expect(screen.queryByRole("button", { name: "Stop the reply" })).toBeNull();
    expect(screen.getAllByRole("button", { name: "Steer the reply" })).toHaveLength(1);
    fireEvent.keyDown(box, { key: "Enter" });

    expect(onSteer).toHaveBeenCalledWith(
      "Actually the carrier is Werner",
      { attachments: [], mentions: [] },
      "steer",
    );
    expect(onSend).not.toHaveBeenCalled();
    expect(box).toHaveValue("");
  });

  it("queues on Option+Enter and from the queue button", () => {
    const onSteer = vi.fn();
    const box = renderComposer({ busy: true, initial: "Then bill it", onSend: vi.fn(), onSteer });

    fireEvent.keyDown(box, { key: "Enter", altKey: true });
    expect(onSteer).toHaveBeenLastCalledWith("Then bill it", expect.anything(), "queue");

    fireEvent.change(box, { target: { value: "And email Acme" } });
    fireEvent.click(screen.getByRole("button", { name: "Queue a follow-up" }));
    expect(onSteer).toHaveBeenLastCalledWith("And email Acme", expect.anything(), "queue");
  });

  it("queues a message with files, and offers no steer for it", () => {
    const onSteer = vi.fn();
    const box = renderComposer({
      busy: true,
      initial: "Read the POD",
      files: [ready],
      onSend: vi.fn(),
      onSteer,
    });

    expect(screen.queryByRole("button", { name: "Steer the reply" })).toBeNull();
    fireEvent.keyDown(box, { key: "Enter" });

    expect(onSteer).toHaveBeenCalledWith(
      "Read the POD",
      expect.objectContaining({ attachments: [expect.objectContaining({ documentId: "doc_1" })] }),
      "queue",
    );
  });

  it("offers nothing to send while the box is empty", () => {
    renderComposer({ busy: true, onSend: vi.fn(), onSteer: vi.fn() });

    expect(screen.queryByRole("button", { name: "Steer the reply" })).toBeNull();
    expect(screen.queryByRole("button", { name: "Queue a follow-up" })).toBeNull();
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

/**
 * A slash command is shown, and asks the agent, in the language on screen: a
 * person reading Spanish sees the command described in Spanish and sends the
 * Spanish question.
 */
describe("DeskComposer slash commands in another language", () => {
  beforeAll(async () => {
    await registerCatalogSource({
      es: async () => ({
        "Where a shipment is and what is holding it up": "Dónde está un envío y qué lo retiene",
        "What is the status of shipment {0} right now, and is anything holding it up?":
          "¿Cuál es el estado del envío {0} ahora mismo y hay algo que lo retenga?",
      }),
    });
    await setLocale("es");
  });

  afterAll(async () => {
    await setLocale("en");
  });

  it("describes the command and sends its question in Spanish", () => {
    const onSend = vi.fn();
    const box = renderComposer({ initial: "/status S12345", onSend });

    expect(screen.getByText("Dónde está un envío y qué lo retiene")).toBeInTheDocument();
    expect(
      screen.getByText(
        "¿Cuál es el estado del envío S12345 ahora mismo y hay algo que lo retenga?",
        {
          exact: false,
        },
      ),
    ).toBeInTheDocument();

    fireEvent.keyDown(box, { key: "Enter" });
    expect(onSend).toHaveBeenCalledWith(
      "¿Cuál es el estado del envío S12345 ahora mismo y hay algo que lo retenga?",
      { attachments: [], mentions: [] },
    );
  });
});
