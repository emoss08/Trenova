import { PdfViewer } from "@/components/elements/pdf-viewer";
import { TextShimmer } from "@/components/ui/text-shimmer";
import { apiService } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import { FileSearchIcon, FileTextIcon } from "lucide-react";

export default function BillingQueueDocumentPreview({
  documentId,
  fileName,
}: {
  documentId: string | null;
  fileName?: string | null;
}) {
  const { data: viewUrl, isLoading } = useQuery({
    queryKey: ["document-view-url", documentId],
    queryFn: () => apiService.documentService.getViewUrl(documentId!),
    enabled: !!documentId,
    staleTime: 4 * 60 * 1000,
  });

  if (!documentId) {
    return (
      <div className="flex h-full flex-col items-center justify-center gap-2 text-muted-foreground p-4 bg-muted/20">
        <FileSearchIcon className="size-10" />
        <p className="text-sm text-center">
          Select a document to preview it here
        </p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex h-full items-center justify-center bg-muted/20">
        <TextShimmer as="span" className="text-sm" duration={1.5}>
          Loading document preview
        </TextShimmer>
      </div>
    );
  }

  if (!viewUrl) {
    return (
      <div className="flex h-full items-center justify-center bg-muted/20">
        <div className="flex flex-col items-center gap-2 text-muted-foreground">
          <FileTextIcon className="size-6 opacity-40" />
          <span className="text-xs">Preview unavailable</span>
        </div>
      </div>
    );
  }

  const isPdf = fileName?.toLowerCase().endsWith(".pdf") ?? true;

  if (!isPdf) {
    return (
      <div className="flex h-full items-center justify-center bg-muted/20 p-4">
        <img
          src={viewUrl}
          alt={fileName || "Document"}
          className="max-h-full max-w-full object-contain"
        />
      </div>
    );
  }

  return <PdfViewer file={viewUrl} mode="scroll" className="h-full border-0" />;
}
