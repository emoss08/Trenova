import { documentContentUrl } from "@/services/document";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatFileSize } from "@trenova/shared/lib/utils";
import type { DocumentPreviewStatus } from "@trenova/shared/types/document";
import type { ReactNode } from "react";
import { DocumentThumbnail } from "./document-thumbnail";

/** What a pane knows about a stored document. */
export type PaneDocument = {
  id: string;
  originalName: string;
  fileType: string | null;
  /** Bytes; 0 when unknown, and then no size is shown. */
  fileSize: number;
  /** Where the server's preview stands; the read that owns the document polls while `Pending`. */
  previewStatus: DocumentPreviewStatus;
};

/**
 * A stored document read in a pane: the server's preview at the document's own proportions
 * (the file's icon when there is none, a spinner while it is drawn), its full name, its size and
 * opening it. The preview opens the document as the link does. `children` sits under the name
 * (review and signature badges). A pane whose footer already opens the document passes
 * `openLink={false}`; the preview still opens it. Without a document it says `empty` in the box.
 */
export function DocumentPanePreview({
  document,
  empty,
  openLink = true,
  children,
  className,
}: {
  document: PaneDocument | null;
  empty?: ReactNode;
  openLink?: boolean;
  children?: ReactNode;
  className?: string;
}) {
  const t = useT();
  const box = "rounded-md border border-border-subtle bg-sunken";

  if (!document) {
    return empty ? (
      <p className={cn(box, "px-4 py-5 text-center text-xs text-muted-foreground", className)}>
        {empty}
      </p>
    ) : null;
  }

  const viewUrl = documentContentUrl(document.id, "view");

  return (
    <div className={cn(box, "flex flex-col items-center gap-2.5 px-4 pt-5 pb-3.5", className)}>
      <a
        href={viewUrl}
        target="_blank"
        rel="noopener noreferrer"
        aria-label={t("Open {0}", document.originalName)}
        className="block max-w-full rounded-xs ui-focus-ring"
      >
        <DocumentThumbnail
          fileType={document.fileType}
          fileName={document.originalName}
          previewStatus={document.previewStatus}
          previewUrl={documentContentUrl(document.id, "preview")}
          size="lg"
          fit="natural"
          className="h-40 w-30 rounded-xs border bg-card"
          imageClassName="h-auto max-h-40 w-auto max-w-full rounded-xs border bg-card"
        />
      </a>
      <p className="max-w-full text-center font-mono text-xs wrap-anywhere text-muted-foreground">
        {document.originalName}
      </p>
      {children}
      {document.fileSize > 0 || openLink ? (
        <p className="flex items-center gap-2 text-xs text-muted-foreground">
          {document.fileSize > 0 ? (
            <span className="tabular-nums">{formatFileSize(document.fileSize)}</span>
          ) : null}
          {openLink ? (
            <a
              href={viewUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="rounded-sm ui-focus-ring text-xs font-medium text-brand hover:underline"
            >
              {t("Open the document")}
            </a>
          ) : null}
        </p>
      ) : null}
    </div>
  );
}
