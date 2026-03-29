import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { Document } from "@/types/document";
import { BrainCircuitIcon, DownloadIcon, EyeIcon, HistoryIcon, Trash2Icon } from "lucide-react";
import { useEffect, useState } from "react";
import { DocumentThumbnail } from "./document-thumbnail";

interface DocumentCardProps {
  document: Document;
  onPreview?: (document: Document) => void;
  onDownload?: (document: Document) => void;
  onDelete?: (document: Document) => void;
  onInspect?: (document: Document) => void;
  onVersions?: (document: Document) => void;
  isDeleting?: boolean;
  className?: string;
  isSelected?: boolean;
  onSelect?: (documentId: string) => void;
  documentTypeName?: string;
}

function formatFileSize(bytes: number): string {
  if (bytes === 0) return "0 B";
  const k = 1024;
  const sizes = ["B", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${parseFloat((bytes / k ** i).toFixed(1))} ${sizes[i] ?? ""}`;
}

function formatDate(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function DocumentCard({
  document,
  onPreview,
  onDownload,
  onDelete,
  onInspect,
  onVersions,
  isDeleting = false,
  className,
  isSelected = false,
  onSelect,
  documentTypeName,
}: DocumentCardProps) {
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const isImage = document.fileType.toLowerCase().startsWith("image/");
  const canPreview = isImage || document.fileType === "application/pdf";

  useEffect(() => {
    if (document.previewStatus !== "Ready" || !document.previewStoragePath) {
      return;
    }

    void apiService.documentService.getPreviewUrl(document.id).then(setPreviewUrl);
  }, [document.id, document.previewStatus, document.previewStoragePath]);

  const handleCheckboxChange = () => {
    if (onSelect) {
      onSelect(document.id);
    }
  };

  return (
    <div
      className={cn(
        "group relative flex items-center gap-3 rounded-lg border bg-card p-3 transition-colors hover:bg-accent/50",
        isDeleting && "pointer-events-none opacity-50",
        isSelected && "ring-1 ring-primary/30",
        className,
      )}
    >
      {onSelect && (
        <div
          className={cn(
            "hidden transition-all duration-200 group-hover:block",
            isSelected && "block",
          )}
        >
          <Checkbox
            checked={isSelected}
            onCheckedChange={handleCheckboxChange}
            className="size-5"
            aria-label={`Select ${document.originalName}`}
          />
        </div>
      )}

      <DocumentThumbnail
        fileType={document.fileType}
        fileName={document.originalName}
        previewStatus={document.previewStatus}
        previewUrl={previewUrl ?? undefined}
        size="md"
      />

      <div className="min-w-0 flex-1">
        <p
          className="truncate text-sm font-medium"
          title={document.originalName}
        >
          {document.originalName}
        </p>
        <p className="text-xs text-muted-foreground">
          {formatFileSize(document.fileSize)} • {formatDate(document.createdAt)}
          {documentTypeName && <> • {documentTypeName}</>}
        </p>
        <div className="mt-1 flex flex-wrap gap-1">
          {document.detectedKind && document.detectedKind !== "Other" && (
            <Badge variant="info" className="h-5 px-1.5 py-0 text-[10px]">
              {document.detectedKind}
            </Badge>
          )}
          {document.versionNumber > 1 && (
            <Badge variant="secondary" className="h-5 px-1.5 py-0 text-[10px]">
              v{document.versionNumber}
            </Badge>
          )}
          {document.contentStatus === "Extracting" && (
            <Badge variant="warning" className="h-5 px-1.5 py-0 text-[10px]">
              Extracting text
            </Badge>
          )}
          {document.contentStatus === "Failed" && (
            <Badge variant="outline" className="h-5 px-1.5 py-0 text-[10px]">
              Extraction failed
            </Badge>
          )}
          {document.shipmentDraftStatus === "Ready" && (
            <Badge variant="teal" className="h-5 px-1.5 py-0 text-[10px]">
              Shipment draft ready
            </Badge>
          )}
        </div>
      </div>

      <div className="flex shrink-0 items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
        {canPreview && onPreview && (
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onPreview(document)}
            aria-label="Preview document"
          >
            <EyeIcon className="size-4" />
          </Button>
        )}
        {onDownload && (
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onDownload(document)}
            aria-label="Download document"
          >
            <DownloadIcon className="size-4" />
          </Button>
        )}
        {onInspect && (
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onInspect(document)}
            aria-label="Inspect document intelligence"
          >
            <BrainCircuitIcon className="size-4" />
          </Button>
        )}
        {onVersions && (
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onVersions(document)}
            aria-label="View document versions"
          >
            <HistoryIcon className="size-4" />
          </Button>
        )}
        {onDelete && (
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onDelete(document)}
            disabled={isDeleting}
            aria-label="Delete document"
          >
            <Trash2Icon className="size-4 text-destructive" />
          </Button>
        )}
      </div>
    </div>
  );
}
