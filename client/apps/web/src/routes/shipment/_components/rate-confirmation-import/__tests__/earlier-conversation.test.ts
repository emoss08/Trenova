import { importAssistantChatHistoryResponseSchema } from "@trenova/shared/types/document";
import { describe, expect, it } from "vitest";
import { earlierConversation } from "../earlier-conversation";

/*
 * GET /documents/:id/import-assistant/history/ reads the legacy
 * shipment_import_chat_* tables for one release. Active conversations were
 * carried into the page's thread by 20261231006560, so only a finished one is
 * shown from here; showing an active one would repeat the thread.
 */
function history(overrides: Record<string, unknown>) {
  return importAssistantChatHistoryResponseSchema.parse({
    documentId: "doc_1",
    conversationId: "sicc_1",
    status: "Completed",
    statusReason: "shipment_created",
    turnCount: 2,
    updatedAt: 20,
    messages: [
      { id: "m1", role: "user", text: "Help me finish this", createdAt: 10 },
      {
        id: "m2",
        role: "assistant",
        text: "",
        toolCalls: [{ name: "accept_field", status: "completed", input: "{}", output: "{}" }],
        createdAt: 11,
      },
      { id: "m3", role: "assistant", text: "Done.", toolCalls: null, createdAt: 12 },
    ],
    ...overrides,
  });
}

describe("earlierConversation", () => {
  it("shows a finished conversation's words in order, read-only", () => {
    expect(earlierConversation(history({}))).toEqual({
      reason: "completed",
      messages: [
        { id: "m1", role: "user", text: "Help me finish this", createdAt: 10 },
        { id: "m3", role: "assistant", text: "Done.", createdAt: 12 },
      ],
    });
  });

  it("says a conversation ended by a re-extraction was superseded", () => {
    expect(
      earlierConversation(history({ status: "Superseded", statusReason: "reextract" }))?.reason,
    ).toBe("superseded");
  });

  it("shows nothing for an active conversation, which the thread already holds", () => {
    expect(earlierConversation(history({ status: "Active", statusReason: "" }))).toBeNull();
  });

  it("shows nothing when there is no history or nothing in it was said", () => {
    expect(earlierConversation(undefined)).toBeNull();
    expect(earlierConversation(history({ messages: null }))).toBeNull();
    expect(
      earlierConversation(
        history({ messages: [{ id: "m", role: "assistant", text: "  ", createdAt: 1 }] }),
      ),
    ).toBeNull();
  });
});
