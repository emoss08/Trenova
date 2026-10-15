import { useT } from "@trenova/shared/i18n/use-t";
import { PdfFrame } from "@/components/elements/pdf-frame";
import { documentContentEndpoint, documentContentUrl } from "@/services/document";
import { DocumentBodySkeleton } from "./document-preview-panel-skeleton";

type DocumentPreviewPanelProps = {
  documentId: string;
  fileName?: string;
};

/**
 * The rate confirmation beside the draft: its name over the document, shown by the browser's
 * own PDF viewer, which brings paging, zoom and search.
 */
export default function DocumentPreviewPanel({ documentId, fileName }: DocumentPreviewPanelProps) {
  const t = useT();
  const name = fileName || t("Document");
  const isPdf = fileName?.toLowerCase().endsWith(".pdf") ?? true;

  if (!isPdf) {
    return (
      <div className="bg-muted/20 flex h-full items-center justify-center p-4">
        <img
          src={documentContentUrl(documentId, "view")}
          alt={name}
          className="max-h-full max-w-full object-contain"
        />
      </div>
    );
  }

  return (
    <div className="bg-muted/20 flex h-full flex-col">
      <div className="bg-background/80 flex shrink-0 items-center border-b px-3 py-1.5">
        <span className="text-muted-foreground truncate text-xs">{name}</span>
      </div>
      <PdfFrame
        endpoint={documentContentEndpoint(documentId, "view")}
        title={name}
        loading={<DocumentBodySkeleton />}
        className="min-h-0 flex-1"
      />
    </div>
  );
}
