import { useT } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { DocumentPreviewStatus } from "@trenova/shared/types/document";
import {
  File04Icon,
  File06Icon,
  FileSpreadsheetIcon,
  Image01Icon,
  SpinnerIcon,
  VideoRecorderIcon,
} from "@trenova/shared/components/icons";
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
  const t = useT();

  const iconType = fileType.toLowerCase();
  const [imageError, setImageError] = useState(false);
  const canHaveThumbnail = supportsThumbnail(fileType);
  const showThumbnail = canHaveThumbnail && previewStatus === "Ready" && previewUrl && !imageError;
  const isGenerating = canHaveThumbnail && previewStatus === "Pending";

  if (showThumbnail) {
    return (
      <div
        className={cn("bg-muted relative overflow-hidden rounded-md", sizeClasses[size], className)}
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
          "bg-muted flex items-center justify-center rounded-md",
          sizeClasses[size],
          className,
        )}
        title={t("Generating thumbnail...")}
      >
        <SpinnerIcon className={cn("text-muted-foreground animate-spin", iconSizes[size])} />
      </div>
    );
  }

  return (
    <div
      className={cn(
        "bg-muted flex items-center justify-center rounded-md",
        sizeClasses[size],
        className,
      )}
      title={previewStatus === "Failed" ? "Preview unavailable" : undefined}
    >
      {iconType.startsWith("image/") && (
        <Image01Icon className={cn("text-muted-foreground", iconSizes[size])} />
      )}
      {iconType.startsWith("video/") && (
        <VideoRecorderIcon className={cn("text-muted-foreground", iconSizes[size])} />
      )}
      {(iconType === "application/pdf" || iconType.includes("pdf")) && (
        <File06Icon className={cn("text-muted-foreground", iconSizes[size])} />
      )}
      {(iconType.includes("spreadsheet") ||
        iconType.includes("excel") ||
        iconType === "text/csv") && (
        <FileSpreadsheetIcon className={cn("text-muted-foreground", iconSizes[size])} />
      )}
      {(iconType.includes("document") ||
        iconType.includes("word") ||
        iconType === "text/plain") && (
        <File06Icon className={cn("text-muted-foreground", iconSizes[size])} />
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
        ) && <File04Icon className={cn("text-muted-foreground", iconSizes[size])} />}
    </div>
  );
}
