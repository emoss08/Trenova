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
  /** The file's MIME type; null when the read did not carry it. */
  fileType: string | null;
  fileName: string;
  /**
   * Where the server's preview stands. Absent or null when the read did not carry it: the
   * preview is then tried, and a file with none falls back to its icon.
   */
  previewStatus?: DocumentPreviewStatus | null;
  previewUrl?: string | null;
  /** The frame: its size when it holds an icon or a spinner, and the cropped image's box. */
  className?: string;
  size?: "sm" | "md" | "lg";
  /**
   * `cover` crops the preview to the frame (a card's square). `natural` draws it at its own
   * proportions, sized by `imageClassName`, so a landscape license and a letter page each read
   * as themselves; the frame shows the file's icon until the image has loaded.
   */
  fit?: "cover" | "natural";
  imageClassName?: string;
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

/** Whether the server draws a preview for a file of this type: images and PDFs. */
export function supportsThumbnail(fileType: string): boolean {
  const type = fileType.toLowerCase();
  return type.startsWith("image/") || type === "application/pdf";
}

function FileTypeIcon({ fileType, className }: { fileType: string | null; className: string }) {
  const type = (fileType ?? "").toLowerCase();
  if (type.startsWith("image/")) return <Image01Icon aria-hidden className={className} />;
  if (type.startsWith("video/")) return <VideoRecorderIcon aria-hidden className={className} />;
  if (type.includes("spreadsheet") || type.includes("excel") || type === "text/csv") {
    return <FileSpreadsheetIcon aria-hidden className={className} />;
  }
  if (
    type.includes("pdf") ||
    type.includes("document") ||
    type.includes("word") ||
    type === "text/plain"
  ) {
    return <File06Icon aria-hidden className={className} />;
  }
  return <File04Icon aria-hidden className={className} />;
}

export function DocumentThumbnail({
  fileType,
  fileName,
  previewStatus,
  previewUrl,
  className,
  size = "md",
  fit = "cover",
  imageClassName,
}: DocumentThumbnailProps) {
  const t = useT();
  const [failedUrl, setFailedUrl] = useState<string | null>(null);
  const [loadedUrl, setLoadedUrl] = useState<string | null>(null);

  const statusKnown = previewStatus !== undefined && previewStatus !== null;
  const canHaveThumbnail = fileType === null || supportsThumbnail(fileType);
  const src =
    canHaveThumbnail && previewUrl && (!statusKnown || previewStatus === "Ready")
      ? previewUrl
      : null;
  const showImage = src !== null && failedUrl !== src;
  const isGenerating = canHaveThumbnail && previewStatus === "Pending";

  const frameClass = cn(
    "flex items-center justify-center rounded-md bg-muted",
    sizeClasses[size],
    className,
  );

  if (showImage && fit === "cover") {
    return (
      <div
        className={cn("relative overflow-hidden rounded-md bg-muted", sizeClasses[size], className)}
      >
        <img
          src={src}
          alt={fileName}
          decoding="async"
          className="size-full object-cover object-top"
          onError={() => setFailedUrl(src)}
        />
      </div>
    );
  }

  if (showImage && fit === "natural") {
    const loaded = loadedUrl === src;
    return (
      <>
        <img
          src={src}
          alt={fileName}
          decoding="async"
          className={cn("block rounded-md", imageClassName, !loaded && "hidden")}
          onLoad={() => setLoadedUrl(src)}
          onError={() => setFailedUrl(src)}
        />
        {loaded ? null : (
          <div className={frameClass}>
            <FileTypeIcon
              fileType={fileType}
              className={cn("text-muted-foreground", iconSizes[size])}
            />
          </div>
        )}
      </>
    );
  }

  if (isGenerating) {
    return (
      <div className={frameClass} title={t("Generating thumbnail...")}>
        <SpinnerIcon className={cn("animate-spin text-muted-foreground", iconSizes[size])} />
      </div>
    );
  }

  return (
    <div
      className={frameClass}
      title={previewStatus === "Failed" ? t("Preview unavailable") : undefined}
    >
      <FileTypeIcon fileType={fileType} className={cn("text-muted-foreground", iconSizes[size])} />
    </div>
  );
}
