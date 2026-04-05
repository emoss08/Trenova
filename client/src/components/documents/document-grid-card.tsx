import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { cn } from "@/lib/utils";
import { apiService } from "@/services/api";
import type { Document } from "@/types/document";
import {
  DownloadIcon,
  EllipsisVerticalIcon,
  EyeIcon,
  HistoryIcon,
  LoaderCircleIcon,
  ScanSearchIcon,
  Trash2Icon,
} from "lucide-react";
import { useEffect, useState } from "react";
import { LazyImage } from "../image";
import { DocumentFileTypeIcon } from "./document-file-type-icon";

interface DocumentGridCardProps {
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
  return `${parseFloat((bytes / k ** i).toFixed(1))} ${sizes[i]}`;
}

function formatDate(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

function supportsThumbnail(fileType: string): boolean {
  const type = fileType.toLowerCase();
  return type.startsWith("image/") || type === "application/pdf";
}

export function DocumentGridCard({
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
}: DocumentGridCardProps) {
  const [previewUrl, setPreviewUrl] = useState<string | null>(null);
  const [imageError, setImageError] = useState(false);
  const isImage = document.fileType.toLowerCase().startsWith("image/");
  const canPreview = isImage || document.fileType === "application/pdf";
  const canHaveThumbnail = supportsThumbnail(document.fileType);
  const showThumbnail =
    canHaveThumbnail &&
    previewUrl &&
    !imageError &&
    document.previewStatus === "Ready" &&
    document.previewStoragePath;
  const isGeneratingThumbnail = document.previewStatus === "Pending";
  const isPreviewUnavailable = document.previewStatus === "Failed";
  const hasActions = Boolean(
    (canPreview && onPreview) || onDownload || onInspect || onVersions || onDelete,
  );

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
        "group relative flex h-full min-w-[200px] flex-col overflow-hidden rounded-lg border bg-card",
        isDeleting && "pointer-events-none opacity-50",
        isSelected && "ring-1 ring-primary/20",
        className,
      )}
    >
      <div className="flex aspect-square items-center justify-center bg-muted/30 p-4">
        {showThumbnail ? (
          <LazyImage
            src={previewUrl ?? ""}
            alt={document.originalName}
            className="size-[250px] shadow-md"
            onError={() => setImageError(true)}
          />
        ) : isGeneratingThumbnail ? (
          <div
            className="flex flex-col items-center justify-center gap-2"
            title="Generating thumbnail..."
          >
            <LoaderCircleIcon className="size-8 animate-spin text-muted-foreground" />
            <span className="text-xs text-muted-foreground">Generating preview...</span>
          </div>
        ) : isPreviewUnavailable ? (
          <div
            className="flex flex-col items-center justify-center gap-2"
            title="Preview unavailable"
          >
            <DocumentFileTypeIcon
              fileType={document.fileType}
              fileName={document.originalName}
              size="xl"
            />
            <span className="text-xs text-muted-foreground">Preview unavailable</span>
          </div>
        ) : (
          <DocumentFileTypeIcon
            fileType={document.fileType}
            fileName={document.originalName}
            size="xl"
          />
        )}
      </div>

      <div className="flex flex-col gap-0.5 border-t px-3 py-2.5">
        <p className="truncate text-sm font-medium" title={document.originalName}>
          {document.originalName}
        </p>
        <p className="truncate text-xs text-muted-foreground">
          {formatFileSize(document.fileSize)} · {formatDate(document.createdAt)}
        </p>
        {documentTypeName && (
          <p className="truncate text-xs text-muted-foreground">{documentTypeName}</p>
        )}
        {document.versionNumber > 1 && onVersions && (
          <button
            type="button"
            className="mt-0.5 text-[10px] text-muted-foreground hover:text-foreground"
            onClick={(e) => {
              e.stopPropagation();
              onVersions(document);
            }}
          >
            v{document.versionNumber} · View history
          </button>
        )}
      </div>

      {onSelect && (
        <div
          className={cn(
            "absolute top-1.5 left-1.5 z-10 opacity-0 transition-opacity group-hover:opacity-100",
            isSelected && "opacity-100",
          )}
        >
          <Checkbox
            checked={isSelected}
            onCheckedChange={handleCheckboxChange}
            className="size-5 border-2 bg-background/80 backdrop-blur-sm"
            aria-label={`Select ${document.originalName}`}
          />
        </div>
      )}

      {hasActions && (
        <div className="absolute top-1.5 right-1.5 rounded-md border bg-background/80 p-0.5 opacity-0 shadow-sm backdrop-blur-sm transition-opacity group-hover:opacity-100">
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-xs"
                  className="rounded-sm"
                  aria-label="Document actions"
                  onClick={(event) => {
                    event.stopPropagation();
                  }}
                >
                  <EllipsisVerticalIcon className="size-3.5" />
                </Button>
              }
            />
            <DropdownMenuContent
              align="end"
              sideOffset={4}
              className="min-w-44"
              onClick={(event) => {
                event.stopPropagation();
              }}
            >
              {canPreview && onPreview && (
                <DropdownMenuItem
                  title="Preview"
                  description="Open the document preview"
                  startContent={<EyeIcon className="size-3.5" />}
                  onClick={() => onPreview(document)}
                />
              )}
              {onDownload && (
                <DropdownMenuItem
                  title="Download"
                  description="Save the original file"
                  startContent={<DownloadIcon className="size-3.5" />}
                  onClick={() => onDownload(document)}
                />
              )}
              {onInspect && (
                <DropdownMenuItem
                  title="Inspect"
                  description="Review extraction details"
                  startContent={<ScanSearchIcon className="size-3.5" />}
                  onClick={() => onInspect(document)}
                />
              )}
              {onVersions && (
                <DropdownMenuItem
                  title="Versions"
                  description="View document history"
                  startContent={<HistoryIcon className="size-3.5" />}
                  onClick={() => onVersions(document)}
                />
              )}
              {onDelete && (
                <DropdownMenuItem
                  title="Delete"
                  description="Remove this document"
                  color="danger"
                  startContent={<Trash2Icon className="size-3.5" />}
                  disabled={isDeleting}
                  onClick={() => onDelete(document)}
                />
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      )}
    </div>
  );
}
