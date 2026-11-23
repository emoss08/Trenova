import { getFileCategory } from "@/components/documents/document-utils";
import { defineLabels } from "@trenova/shared/i18n/labels";
import type { Document } from "@trenova/shared/types/document";

/** The type filter's choices: every file, or one kind of file. */
export const DOCUMENT_KINDS = ["all", "pdf", "image", "document", "spreadsheet", "other"] as const;
export type DocumentKind = (typeof DOCUMENT_KINDS)[number];

export const DOCUMENT_KIND_LABELS: Record<DocumentKind, string> = defineLabels({
  all: "All types",
  pdf: "PDFs",
  image: "Images",
  document: "Documents",
  spreadsheet: "Spreadsheets",
  other: "Other files",
});

/** What one file is, in the type column, when it was not filed under a document type. */
const FILE_KIND_NAMES: Record<string, string> = defineLabels({
  pdf: "PDF",
  image: "Image",
  document: "Document",
  spreadsheet: "Spreadsheet",
  data: "Data file",
  default: "File",
});

export function documentKind(document: Pick<Document, "fileType" | "originalName">): DocumentKind {
  const category = getFileCategory(document.fileType, document.originalName);
  switch (category) {
    case "pdf":
    case "image":
    case "document":
    case "spreadsheet":
      return category;
    default:
      return "other";
  }
}

/** The type column: the document type it was filed under, or the kind of file it is. */
export function documentTypeName(document: Document): string {
  const filed = document.documentType?.name?.trim();
  if (filed) return filed;
  return (
    FILE_KIND_NAMES[getFileCategory(document.fileType, document.originalName)] ??
    FILE_KIND_NAMES.default
  );
}

/** Only current versions are listed; newest first. */
export function sortDocuments(documents: readonly Document[]): Document[] {
  return documents
    .filter((document) => document.isCurrentVersion !== false)
    .sort((a, b) => b.createdAt - a.createdAt);
}

/** How many current documents of each kind there are, for the type filter's counts. */
export function kindCounts(documents: readonly Document[]): Record<DocumentKind, number> {
  const counts = Object.fromEntries(DOCUMENT_KINDS.map((kind) => [kind, 0])) as Record<
    DocumentKind,
    number
  >;
  for (const document of documents) {
    counts.all += 1;
    counts[documentKind(document)] += 1;
  }
  return counts;
}

/**
 * The documents a search and a type leave. The search reads the name and the type here, and the
 * person who added it when their name is known, on top of what the server already matched.
 */
export function filterDocuments(
  documents: readonly Document[],
  kind: DocumentKind,
  search: string,
  uploaderName: (userId: string) => string | null,
): Document[] {
  const needle = search.trim().toLowerCase();
  return documents.filter((document) => {
    if (kind !== "all" && documentKind(document) !== kind) return false;
    if (needle === "") return true;
    const haystack = [
      document.originalName,
      documentTypeName(document),
      uploaderName(document.uploadedById) ?? "",
    ]
      .join(" ")
      .toLowerCase();
    return haystack.includes(needle);
  });
}

/** Whether the browser can show the file itself, rather than only download it. */
export function canPreviewDocument(document: Pick<Document, "fileType">): boolean {
  const type = document.fileType.toLowerCase();
  return type.startsWith("image/") || type === "application/pdf";
}
