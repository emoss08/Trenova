import { apiService } from "@/services/api";
import type { AssistantMessageAttachment as MessageAttachment } from "@/types/assistant";
import type { DocumentContent } from "@trenova/shared/types/document";
import { useQueries } from "@tanstack/react-query";
import { useMemo } from "react";

/** The average OCR confidence under which a scan counts as mostly unreadable, as the server judges it. */
const POOR_READING_CONFIDENCE = 0.5;

/** How often a file still being read is checked again. */
const READING_POLL_MS = 5000;

/**
 * Whether reading finished but made out little of a file: the pages read by
 * OCR were read with low confidence on average, as a blurred or skewed photo
 * is. Text a file carried natively is always legible, and a reading that
 * failed says nothing about the file itself.
 */
export function poorlyRead(content: Pick<DocumentContent, "status" | "pages">): boolean {
  if (content.status !== "Extracted" && content.status !== "Indexed") return false;
  const scanned = (content.pages ?? []).filter((page) => page.sourceKind === "ocr");
  if (scanned.length === 0) return false;
  const average = scanned.reduce((sum, page) => sum + page.ocrConfidence, 0) / scanned.length;
  return average < POOR_READING_CONFIDENCE;
}

function scannable(file: MessageAttachment): boolean {
  const type = file.contentType ?? "";
  return (
    type.startsWith("image/") ||
    type === "application/pdf" ||
    /\.(pdf|png|jpe?g|heic|tiff?)$/iu.test(file.fileName)
  );
}

/**
 * The attached files reading made out little of, by document id. A file
 * marked so when the question was sent counts at once; a scan still being
 * read when it went is checked until reading finishes, since a photo taken
 * on the spot is usually still being read as it is sent.
 */
export function usePoorlyReadFiles(files: readonly MessageAttachment[]): ReadonlySet<string> {
  const checked = useMemo(
    () => files.filter((file) => !file.poorlyRead && scannable(file)),
    [files],
  );
  const results = useQueries({
    queries: checked.map((file) => ({
      queryKey: ["document-content", file.documentId],
      queryFn: async () => {
        try {
          return await apiService.documentService.getContent(file.documentId);
        } catch {
          return null;
        }
      },
      staleTime: Number.POSITIVE_INFINITY,
      refetchInterval: (query: { state: { data: DocumentContent | null | undefined } }) => {
        const status = query.state.data?.status;
        return status === "Pending" || status === "Extracting" ? READING_POLL_MS : false;
      },
    })),
  });

  const flags = results.map((result) => (result.data ? poorlyRead(result.data) : false)).join();
  return useMemo(() => {
    const out = new Set(files.filter((file) => file.poorlyRead).map((file) => file.documentId));
    const marks = flags.split(",");
    checked.forEach((file, index) => {
      if (marks[index] === "true") out.add(file.documentId);
    });
    return out;
  }, [files, checked, flags]);
}
