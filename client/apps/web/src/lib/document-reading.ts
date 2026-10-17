import type { DocumentContent } from "@trenova/shared/types/document";

/**
 * How long a model's reading of a document is waited for once its text is in,
 * as the server judges it (documentcontent.ModelReadingWindowSeconds). A
 * reading still marked pending past it was lost along the way.
 */
export const MODEL_READING_WINDOW_SECONDS = 15 * 60;

type ReadingState = Pick<DocumentContent, "status" | "structuredData" | "lastExtractedAt">;

/** Whether the model's reading of a document whose text is in is still under way. */
export function modelReadingPending(content: ReadingState, nowSeconds: number): boolean {
  if (content.structuredData?.aiDiagnostics?.acceptanceStatus !== "pending") return false;
  if (content.lastExtractedAt == null) return true;
  return nowSeconds - content.lastExtractedAt < MODEL_READING_WINDOW_SECONDS;
}

/**
 * Whether a document's reading has not finished: its text is not in, or the
 * model is still reading it. A document that could not be read is done.
 */
export function stillBeingRead(content: ReadingState, nowSeconds: number): boolean {
  if (content.status === "Pending" || content.status === "Extracting") return true;
  return content.status !== "Failed" && modelReadingPending(content, nowSeconds);
}
