import { deskSearchHref } from "@/lib/desk-search-href";
import type { DeskSearchResult } from "@/types/assistant";
import { MessageChatSquareIcon } from "@trenova/shared/components/icons";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import type { PalettePage } from "./palette-model";

/** What the palette holds of the person's conversations for the query. */
export interface ConversationHits {
  pages: readonly PalettePage[];
  loading: boolean;
  error: boolean;
}

function kindLabel(kind: DeskSearchResult["kind"], t: TranslateFn): string {
  switch (kind) {
    case "chat":
      return t("Conversation");
    case "chap":
      return t("Chapter");
    case "msg":
      return t("Message");
    case "art":
      return t("Artifact");
    default:
      return t("Decision");
  }
}

/**
 * A Desk search hit as a row the palette opens: the conversation, chapter,
 * message, artifact or decision found, under the conversation it belongs to.
 */
export function conversationPage(result: DeskSearchResult, t: TranslateFn): PalettePage {
  const title = result.title || result.threadTitle || t("Untitled conversation");
  const trail =
    result.threadTitle !== "" && result.threadTitle !== title
      ? [kindLabel(result.kind, t), result.threadTitle]
      : [kindLabel(result.kind, t)];

  return {
    id: `conversation:${result.kind}:${result.id}`,
    title,
    trail,
    href: deskSearchHref(result),
    icon: MessageChatSquareIcon,
    keywords: [],
    module: "desk",
  };
}
