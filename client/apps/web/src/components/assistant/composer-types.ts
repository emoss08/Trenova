import type { AssistantEntityRef, AssistantMessageAttachment } from "@/types/assistant";

/** A file on its way to the message, or already there. */
export type ComposerAttachment = {
  id: string;
  name: string;
  size: number;
  status: "uploading" | "ready" | "error";
  progress: number;
  documentId?: string;
  contentType?: string;
  error?: string;
  /** The file as picked, for a preview of an image before it is sent. */
  file?: File;
};

/** A record the mention search offers. */
export type MentionCandidate = AssistantEntityRef & {
  subtitle?: string;
};

/** What leaves with the words. */
export type ComposerPayload = {
  attachments: AssistantMessageAttachment[];
  mentions: AssistantEntityRef[];
};

/** The mentions still named in the text; one whose @label was deleted is dropped. */
export function activeMentions(
  draft: string,
  mentions: readonly AssistantEntityRef[],
): AssistantEntityRef[] {
  return mentions.filter((mention) => mention.label !== "" && draft.includes("@" + mention.label));
}

/** The documents that are on the message, as the server takes them. */
export function readyAttachments(
  attachments: readonly ComposerAttachment[],
): AssistantMessageAttachment[] {
  return attachments.flatMap((attachment) =>
    attachment.status === "ready" && attachment.documentId
      ? [
          {
            documentId: attachment.documentId,
            fileName: attachment.name,
            contentType: attachment.contentType ?? "",
            fileSize: attachment.size,
          },
        ]
      : [],
  );
}
