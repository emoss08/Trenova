import { Button } from "@/components/ui/button";
import { ScrollArea } from "@/components/ui/scroll-area";
import { TextShimmer } from "@/components/ui/text-shimmer";
import { apiService } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import { FileTextIcon, ZoomInIcon, ZoomOutIcon } from "lucide-react";
import { useCallback, useRef, useState } from "react";
import { Document, Page, pdfjs } from "react-pdf";
import "react-pdf/dist/Page/AnnotationLayer.css";
import "react-pdf/dist/Page/TextLayer.css";

pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  "pdfjs-dist/build/pdf.worker.min.mjs",
  import.meta.url,
).toString();

type DocumentPreviewPanelProps = {
  documentId: string;
  fileName?: string;
};

export default function DocumentPreviewPanel({ documentId, fileName }: DocumentPreviewPanelProps) {
  const [numPages, setNumPages] = useState<number>(0);
  const [scale, setScale] = useState(1.0);
  const containerRef = useRef<HTMLDivElement>(null);

  const { data: viewUrl, isLoading } = useQuery({
    queryKey: ["document-view-url", documentId],
    queryFn: () => apiService.documentService.getViewUrl(documentId),
    enabled: !!documentId,
    staleTime: 4 * 60 * 1000,
  });

  const onDocumentLoadSuccess = useCallback(({ numPages: n }: { numPages: number }) => {
    setNumPages(n);
  }, []);

  const isPdf = fileName?.toLowerCase().endsWith(".pdf") ?? true;

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

  if (!isPdf) {
    return (
      <div className="flex h-full items-center justify-center bg-muted/20 p-4">
        <img
          src={viewUrl}
          alt={fileName || "Rate Confirmation"}
          className="max-h-full max-w-full object-contain"
        />
      </div>
    );
  }

  const pageNumbers = Array.from({ length: numPages }, (_, i) => i + 1);

  return (
    <div className="flex h-full flex-col bg-muted/20">
      {/* Toolbar */}
      <div className="flex shrink-0 items-center justify-between border-b bg-background/80 px-3 py-1.5">
        <span className="truncate text-xs text-muted-foreground">
          {fileName || "Document"}
          {numPages > 0 && (
            <span className="ml-1.5 text-muted-foreground/50">{numPages} pages</span>
          )}
        </span>
        <div className="flex items-center gap-1">
          <Button
            variant="ghost"
            size="icon-xs"
            onClick={() => setScale((s) => Math.max(0.5, s - 0.15))}
            disabled={scale <= 0.5}
          >
            <ZoomOutIcon className="size-3.5" />
          </Button>
          <span className="w-10 text-center text-2xs text-muted-foreground tabular-nums">
            {Math.round(scale * 100)}%
          </span>
          <Button
            variant="ghost"
            size="icon-xs"
            onClick={() => setScale((s) => Math.min(2.5, s + 0.15))}
            disabled={scale >= 2.5}
          >
            <ZoomInIcon className="size-3.5" />
          </Button>
        </div>
      </div>

      {/* All pages stacked */}
      <ScrollArea className="flex-1">
        <div ref={containerRef} className="flex flex-col items-center gap-3 p-4">
          <Document
            file={viewUrl}
            onLoadSuccess={onDocumentLoadSuccess}
            loading={
              <div className="flex items-center justify-center py-20">
                <TextShimmer as="span" className="text-xs" duration={1.5}>
                  Rendering document
                </TextShimmer>
              </div>
            }
            error={
              <div className="flex items-center justify-center py-20 text-xs text-muted-foreground">
                Failed to load PDF
              </div>
            }
          >
            {pageNumbers.map((pageNum) => (
              <div key={pageNum} className="relative mb-2">
                {numPages > 1 && (
                  <div className="mb-1 text-center text-2xs text-muted-foreground/40">
                    Page {pageNum}
                  </div>
                )}
                <Page
                  pageNumber={pageNum}
                  scale={scale}
                  loading={
                    <div className="flex h-40 items-center justify-center">
                      <TextShimmer as="span" className="text-2xs" duration={1.5}>
                        Loading page
                      </TextShimmer>
                    </div>
                  }
                  className="shadow-sm"
                />
              </div>
            ))}
          </Document>
        </div>
      </ScrollArea>
    </div>
  );
}
