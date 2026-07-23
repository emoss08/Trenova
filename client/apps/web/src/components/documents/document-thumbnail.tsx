import { cn } from "@/lib/utils";
import type { DocumentPreviewStatus } from "@/types/document";
import {
  FileIcon,
  FileSpreadsheetIcon,
  FileTextIcon,
  ImageIcon,
  LoaderCircleIcon,
  VideoIcon,
} from "lucide-react";
import { useState } from "react";

interface DocumentThumbnailProps {
  fileType: string;
  fileName: string;
  previewStatus: DocumentPreviewStatus;
  previewUrl?: string;
  className?: string;
  size?: "sm" | "md" | "lg";
}

const sizeClasses = {
  sm: "size-8",
  md: "size-12",
  lg: "size-16",
};

const iconSizes = {
  sm: "size-4",
  md: "size-6",
  lg: "size-8",
};

function supportsThumbnail(fileType: string): boolean {
  const type = fileType.toLowerCase();
  return type.startsWith("image/") || type === "application/pdf";
}

export function DocumentThumbnail({
  fileType,
  fileName,
  previewStatus,
  previewUrl,
  className,
  size = "md",
}: DocumentThumbnailProps) {
  const iconType = fileType.toLowerCase();
  const [imageError, setImageError] = useState(false);
  const canHaveThumbnail = supportsThumbnail(fileType);
  const showThumbnail =
    canHaveThumbnail && previewStatus === "Ready" && previewUrl && !imageError;
  const isGenerating =
    canHaveThumbnail && previewStatus === "Pending";

  if (showThumbnail) {
    return (
      <div
        className={cn(
          "relative overflow-hidden rounded-md bg-muted",
          sizeClasses[size],
          className,
        )}
      >
        <img
          src={previewUrl}
          alt={fileName}
          className="size-full object-cover"
          onError={() => setImageError(true)}
        />
      </div>
    );
  }

  if (isGenerating) {
    return (
      <div
        className={cn(
          "flex items-center justify-center rounded-md bg-muted",
          sizeClasses[size],
          className,
        )}
        title="Generating thumbnail..."
      >
        <LoaderCircleIcon
          className={cn("animate-spin text-muted-foreground", iconSizes[size])}
        />
      </div>
    );
  }

  return (
    <div
      className={cn(
        "flex items-center justify-center rounded-md bg-muted",
        sizeClasses[size],
        className,
      )}
      title={previewStatus === "Failed" ? "Preview unavailable" : undefined}
    >
      {iconType.startsWith("image/") && (
        <ImageIcon className={cn("text-muted-foreground", iconSizes[size])} />
      )}
      {iconType.startsWith("video/") && (
        <VideoIcon className={cn("text-muted-foreground", iconSizes[size])} />
      )}
      {(iconType === "application/pdf" || iconType.includes("pdf")) && (
        <FileTextIcon
          className={cn("text-muted-foreground", iconSizes[size])}
        />
      )}
      {(iconType.includes("spreadsheet") ||
        iconType.includes("excel") ||
        iconType === "text/csv") && (
        <FileSpreadsheetIcon
          className={cn("text-muted-foreground", iconSizes[size])}
        />
      )}
      {(iconType.includes("document") ||
        iconType.includes("word") ||
        iconType === "text/plain") && (
        <FileTextIcon
          className={cn("text-muted-foreground", iconSizes[size])}
        />
      )}
      {!iconType.startsWith("image/") &&
        !iconType.startsWith("video/") &&
        !(iconType === "application/pdf" || iconType.includes("pdf")) &&
        !(
          iconType.includes("spreadsheet") ||
          iconType.includes("excel") ||
          iconType === "text/csv"
        ) &&
        !(
          iconType.includes("document") ||
          iconType.includes("word") ||
          iconType === "text/plain"
        ) && (
          <FileIcon className={cn("text-muted-foreground", iconSizes[size])} />
        )}
    </div>
  );
}
