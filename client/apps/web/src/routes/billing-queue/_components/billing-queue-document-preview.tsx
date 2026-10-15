import { useT } from "@trenova/shared/i18n/use-t";
import { PdfFrame } from "@/components/elements/pdf-frame";
import { documentContentEndpoint, documentContentUrl } from "@/services/document";
import { FileSearch02Icon } from "@trenova/shared/components/icons";
import { BillingQueueDocumentPreviewSkeleton } from "./billing-queue-document-preview-skeleton";

export default function BillingQueueDocumentPreview({
  documentId,
  fileName,
}: {
  documentId: string | null;
  fileName?: string | null;
}) {
  const t = useT();

  if (!documentId) {
    return (
      <div className="bg-muted/20 text-muted-foreground flex h-full flex-col items-center justify-center gap-2 p-4">
        <FileSearch02Icon className="size-10" />
        <p className="text-center text-sm">{t("Select a document to preview it here")}</p>
      </div>
    );
  }

  const isPdf = fileName?.toLowerCase().endsWith(".pdf") ?? true;

  if (!isPdf) {
    return (
      <div className="bg-muted/20 flex h-full items-center justify-center p-4">
        <img
          src={documentContentUrl(documentId, "view")}
          alt={fileName || t("Document")}
          className="max-h-full max-w-full object-contain"
        />
      </div>
    );
  }

  return (
    <PdfFrame
      endpoint={documentContentEndpoint(documentId, "view")}
      title={fileName || t("Document")}
      loading={<BillingQueueDocumentPreviewSkeleton />}
      className="h-full"
    />
  );
}
