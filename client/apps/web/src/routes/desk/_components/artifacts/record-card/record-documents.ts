import { getFileCategory } from "@/components/documents/document-utils";
import { documentReviewState, type DocumentReviewState } from "@/lib/document-review";
import { hasPendingDocumentWork } from "@/lib/resource-documents";
import type { TranslateFn } from "@trenova/shared/i18n/use-t";
import { formatFileSize } from "@trenova/shared/lib/utils";
import type { Document } from "@trenova/shared/types/document";
import { cardDay } from "./record-card-values";

/*
A record's documents as every card's document rail reads them: which to show,
in what order, and how each is named. Pure, so a card for any record with
documents draws them the same way.
*/

/** One of a record's documents as a card's document rail draws it. */
export type DocumentTile = {
  id: string;
  name: string;
  /** What the document is to its record: its type, such as "Proof of delivery". */
  type: string;
  /** The type's short code, for the tile's corner. */
  code: string;
  /** The file's media type, for its mark when there is no preview. */
  fileType: string;
  /** What the file is: "pdf", "image", "spreadsheet", "document", "data" or "default". */
  category: string;
  /** "PDF · 240 KB · Oct 12". */
  meta: string;
  /** Whether the server has drawn a preview to show. */
  preview: boolean;
  /** Whether the server is still working on it. */
  pending: boolean;
  /** Where its review stands: approved, rejected, awaiting a reviewer, or never asked. */
  review: Exclude<DocumentReviewState, "inactive">;
};

const CATEGORY_NAMES: Readonly<Record<string, (t: TranslateFn) => string>> = {
  pdf: (t) => t("PDF"),
  image: (t) => t("Image"),
  spreadsheet: (t) => t("Spreadsheet"),
  document: (t) => t("Document"),
  data: (t) => t("Data"),
};

function reviewOf(document: Document): Exclude<DocumentReviewState, "inactive"> {
  const state = documentReviewState(document);
  return state === "inactive" ? "unreviewed" : state;
}

/** How a document's review reads on its tile; "" when nobody was asked to review it. */
export function documentReviewLabel(review: DocumentTile["review"], t: TranslateFn): string {
  switch (review) {
    case "approved":
      return t("Approved");
    case "rejected":
      return t("Rejected");
    case "awaiting":
      return t("Awaiting review");
    case "unreviewed":
      return "";
  }
}

/**
 * A record's documents as tiles: the current version of each, newest first,
 * an older version being history the documents tab keeps. A document
 * whose type is known is named for it, its file for what it is.
 */
export function documentTiles(documents: readonly Document[], t: TranslateFn): DocumentTile[] {
  return documents
    .filter((document) => document.isCurrentVersion)
    .sort((a, b) => b.createdAt - a.createdAt)
    .map((document) => {
      const category = getFileCategory(document.fileType, document.originalName);
      const name = Object.hasOwn(CATEGORY_NAMES, category) ? CATEGORY_NAMES[category](t) : "";
      return {
        id: document.id,
        name: document.originalName,
        type: document.documentType?.name ?? "",
        code: document.documentType?.code ?? "",
        fileType: document.fileType,
        category,
        meta: [name, formatFileSize(document.fileSize), cardDay(document.createdAt)]
          .filter(Boolean)
          .join(" · "),
        preview: document.previewStatus === "Ready",
        pending: hasPendingDocumentWork(document),
        review: reviewOf(document),
      };
    });
}
