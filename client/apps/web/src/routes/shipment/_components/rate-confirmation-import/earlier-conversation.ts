import type { ImportAssistantChatHistoryResponse } from "@trenova/shared/types/document";

export type EarlierMessage = {
  id: string;
  role: "user" | "assistant";
  text: string;
  createdAt: number;
};

export type EarlierConversation = {
  reason: "completed" | "superseded";
  messages: EarlierMessage[];
};

/**
 * A finished conversation from before the import assistant moved onto the
 * shared runtime, read from the legacy history for as long as it is kept.
 * Active conversations were carried into the page's own thread, so only a
 * finished one is shown: what was said, in order, and why it ended.
 */
export function earlierConversation(
  history: ImportAssistantChatHistoryResponse | undefined,
): EarlierConversation | null {
  if (!history || history.status === "Active") {
    return null;
  }

  const messages = history.messages
    .filter((message) => message.text.trim() !== "")
    .map((message) => ({
      id: message.id,
      role: message.role,
      text: message.text,
      createdAt: message.createdAt,
    }));
  if (messages.length === 0) {
    return null;
  }

  return {
    reason: history.status === "Superseded" ? "superseded" : "completed",
    messages,
  };
}
