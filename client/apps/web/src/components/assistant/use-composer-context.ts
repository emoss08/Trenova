import { useDocumentUpload } from "@/hooks/use-document-upload";
import { apiService } from "@/services/api";
import type { AssistantEntityRef } from "@/types/assistant";
import type { UploadState } from "@/types/upload";
import { useCallback, useMemo, useState } from "react";
import type { ComposerAttachment, MentionCandidate } from "./composer";

/** The resource an attachment is uploaded against; the server checks it. */
export const ATTACHMENT_RESOURCE_TYPE = "assistant_thread";

/** How many records the mention search offers at once. */
const MENTION_RESULT_LIMIT = 8;

function toComposerAttachment(
  upload: UploadState,
  documents: ReadonlyMap<string, { id: string; contentType: string }>,
): ComposerAttachment {
  const document = documents.get(upload.id);
  const status: ComposerAttachment["status"] =
    upload.status === "error" || upload.status === "quarantined"
      ? "error"
      : document
        ? "ready"
        : "uploading";

  return {
    id: upload.id,
    name: upload.file.name,
    size: upload.file.size,
    status,
    progress: upload.progress,
    documentId: document?.id,
    contentType: document?.contentType ?? upload.file.type,
    error: upload.error,
  };
}

/**
 * What a message carries besides its words, for one thread: the files being
 * uploaded to it, and the records named from the composer. Files go up as
 * they are picked, with the profile that has document intelligence read
 * them, so by the time the message leaves the assistant can read the file.
 */
export function useComposerContext(threadId: string) {
  const [documents, setDocuments] = useState<Map<string, { id: string; contentType: string }>>(
    () => new Map(),
  );
  const [mentions, setMentions] = useState<AssistantEntityRef[]>([]);

  const { uploads, uploadFiles, cancelUpload, removeUpload, clearAll } = useDocumentUpload({
    resourceId: threadId,
    resourceType: ATTACHMENT_RESOURCE_TYPE,
    processingProfile: "assistant_attachment",
    invalidateQueryKey: ["assistant", "attachments", threadId],
    onSuccess: (document, upload) => {
      setDocuments((current) => {
        const next = new Map(current);
        next.set(upload.id, { id: document.id, contentType: document.fileType });
        return next;
      });
    },
  });

  const attachments = useMemo(
    () => uploads.map((upload) => toComposerAttachment(upload, documents)),
    [documents, uploads],
  );

  const attachFiles = useCallback((files: File[]) => uploadFiles(files), [uploadFiles]);

  const removeAttachment = useCallback(
    (id: string) => {
      const upload = uploads.find((item) => item.id === id);
      if (upload && upload.status !== "success" && upload.status !== "error") {
        cancelUpload(id);
      }
      removeUpload(id);
      setDocuments((current) => {
        if (!current.has(id)) {
          return current;
        }
        const next = new Map(current);
        next.delete(id);
        return next;
      });
    },
    [cancelUpload, removeUpload, uploads],
  );

  const clear = useCallback(() => {
    clearAll();
    setDocuments(new Map());
    setMentions([]);
  }, [clearAll]);

  const searchMentions = useCallback(async (query: string): Promise<MentionCandidate[]> => {
    const response = await apiService.globalSearchService.search(query, MENTION_RESULT_LIMIT);

    return response.groups.flatMap((group) =>
      group.hits.map<MentionCandidate>((hit) => ({
        type: hit.entityType,
        id: hit.id,
        label: hit.title,
        subtitle: hit.subtitle,
      })),
    );
  }, []);

  return {
    attachments,
    attachFiles,
    removeAttachment,
    mentions,
    setMentions,
    searchMentions,
    clear,
  };
}
