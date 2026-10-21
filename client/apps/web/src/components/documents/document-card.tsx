import { useT } from "@trenova/shared/i18n/use-t";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Checkbox } from "@trenova/shared/components/ui/checkbox";
import { cn } from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import type { Document } from "@trenova/shared/types/document";
import {
  BrainCircuitIcon,
  CheckCircleIcon,
  ClockRewindIcon,
  Download01Icon,
  Edit05Icon,
  EyeIcon,
  SlashCircle01Icon,
  Trash01Icon,
} from "@trenova/shared/components/icons";
import { useEffect, useState } from "react";
import { canApproveDocument, canRejectDocument } from "@/lib/document-review";
import { documentNeedsSignature } from "@/lib/document-signature";
import { DocumentReviewBadge } from "./document-review-badge";
import { DocumentSignatureBadge } from "./document-signature-badge";
import { DocumentThumbnail } from "./document-thumbnail";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";

interface DocumentCardProps {
  document: Document;
  onPreview?: (document: Document) => void;
  onDownload?: (document: Document) => void;
  onDelete?: (document: Document) => void;
  onInspect?: (document: Document) => void;
  onVersions?: (document: Document) => void;
  onApprove?: (document: Document) => void;
  onReject?: (document: Document) => void;
  onRecordSignature?: (document: Document) => void;
  isDeleting?: boolean;
  isReviewing?: boolean;
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

export function DocumentCard({
  document,
  onPreview,
  onDownload,
  onDelete,
  onInspect,
  onVersions,
  onApprove,
  onReject,
  onRecordSignature,
  isDeleting = false,
  isReviewing = false,
  className,
  isSelected = false,
  onSelect,
  documentTypeName,
}: DocumentCardProps) {
  const t = useT();

  const [fetchedPreview, setFetchedPreview] = useState<{
    documentId: string;
    url: string | null;
  } | null>(null);
  const isPreviewReady = document.previewStatus === "Ready";
  const previewUrl =
    isPreviewReady && fetchedPreview?.documentId === document.id ? fetchedPreview.url : null;
  const isImage = document.fileType.toLowerCase().startsWith("image/");
  const canPreview = isImage || document.fileType === "application/pdf";

  useEffect(() => {
    if (!isPreviewReady) {
      return;
    }

    const documentId = document.id;
    void apiService.documentService
      .getPreviewUrl(documentId)
      .then((url) => setFetchedPreview({ documentId, url }));
  }, [document.id, isPreviewReady]);

  const handleCheckboxChange = () => {
    if (onSelect) {
      onSelect(document.id);
    }
  };

  return (
    <div
      className={cn(
        "group bg-card hover:bg-accent/50 relative flex items-center gap-3 rounded-lg border p-3 transition-colors",
        isDeleting && "pointer-events-none opacity-50",
        isSelected && "ring-primary/30 ring-1",
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
            aria-label={t("Select {0}", document.originalName)}
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
        <p className="truncate text-sm font-medium" title={document.originalName}>
          {document.originalName}
        </p>
        <p className="text-muted-foreground text-xs">
          {formatFileSize(document.fileSize)} • {formatUnixDateMedium(document.createdAt)}
          {documentTypeName && <> • {documentTypeName}</>}
        </p>
        <div className="mt-1 flex flex-wrap gap-1">
          <DocumentReviewBadge document={document} />
          {documentNeedsSignature(document) && <DocumentSignatureBadge document={document} />}
          {document.detectedKind && document.detectedKind !== "Other" && (
            <Badge variant="info" className="h-5 px-1.5 py-0 text-2xs">
              {document.detectedKind}
            </Badge>
          )}
          {document.versionNumber > 1 && onVersions && (
            <button
              type="button"
              onClick={(e) => {
                e.stopPropagation();
                onVersions(document);
              }}
            >
              <Badge
                variant="neutral"
                className="hover:bg-secondary/80 h-5 cursor-pointer px-1.5 py-0 text-2xs"
              >
                <ClockRewindIcon className="mr-0.5 size-3" />
                {t("v{0}", document.versionNumber)}
              </Badge>
            </button>
          )}
          {document.contentStatus === "Extracting" && (
            <Badge variant="warning" className="h-5 px-1.5 py-0 text-2xs">
              {t("Extracting text")}
            </Badge>
          )}
          {document.contentStatus === "Failed" && (
            <Badge variant="neutral" appearance="outline" className="h-5 px-1.5 py-0 text-2xs">
              {t("Extraction failed")}
            </Badge>
          )}
          {document.shipmentDraftStatus === "Ready" && (
            <Badge variant="success" className="h-5 px-1.5 py-0 text-2xs">
              {t("Shipment draft ready")}
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
            aria-label={t("Preview document")}
          >
            <EyeIcon className="size-4" />
          </Button>
        )}
        {onDownload && (
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onDownload(document)}
            aria-label={t("Download document")}
          >
            <Download01Icon className="size-4" />
          </Button>
        )}
        {onInspect && (
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onInspect(document)}
            aria-label={t("Inspect document extraction")}
          >
            <BrainCircuitIcon className="size-4" />
          </Button>
        )}
        {onVersions && (
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onVersions(document)}
            aria-label={t("View document versions")}
          >
            <ClockRewindIcon className="size-4" />
          </Button>
        )}
        {onApprove && canApproveDocument(document) && (
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onApprove(document)}
            disabled={isReviewing}
            aria-label={t("Approve document")}
          >
            <CheckCircleIcon className="size-4" />
          </Button>
        )}
        {onReject && canRejectDocument(document) && (
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onReject(document)}
            disabled={isReviewing}
            aria-label={t("Reject document")}
          >
            <SlashCircle01Icon className="size-4" />
          </Button>
        )}
        {onRecordSignature && documentNeedsSignature(document) && (
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onRecordSignature(document)}
            aria-label={t("Record signature")}
          >
            <Edit05Icon className="size-4" />
          </Button>
        )}
        {onDelete && (
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onDelete(document)}
            disabled={isDeleting}
            aria-label={t("Delete document")}
          >
            <Trash01Icon className="text-destructive size-4" />
          </Button>
        )}
      </div>
    </div>
  );
}
