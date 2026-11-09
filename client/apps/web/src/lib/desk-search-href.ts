import { conversationPath } from "@/lib/conversation-path";
import type { DeskSearchResult } from "@/types/assistant";

/** Where a Desk search hit opens: its conversation, at the chapter or artifact it found. */
export function deskSearchHref(result: DeskSearchResult): string {
  switch (result.kind) {
    case "dec":
      return "/desk/decisions";
    case "art":
      return `${conversationPath(result.threadId)}?a=${encodeURIComponent(result.id)}`;
    case "chap":
      return `${conversationPath(result.threadId)}?c=${encodeURIComponent(result.id)}`;
    default:
      return conversationPath(result.threadId);
  }
}
