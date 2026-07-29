import { DocumentFileTypeIcon } from "@/components/documents/document-file-type-icon";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { formatFileSize } from "@trenova/shared/lib/utils";
import { ChevronLeftIcon, ChevronRightIcon, DownloadIcon, ExternalLinkIcon } from "lucide-react";
import { useCallback, useEffect, useMemo, useState } from "react";
import { documentContentUrl } from "@/services/document";
import type { ShipmentCommentAttachment } from "@/types/shipment-comment";

function isImageAttachment(attachment: ShipmentCommentAttachment): boolean {
  return attachment.mimeType.startsWith("image/");
}

function attachmentViewUrl(attachment: ShipmentCommentAttachment): string {
  return documentContentUrl(attachment.documentId, "view");
}

function attachmentDownloadUrl(attachment: ShipmentCommentAttachment): string {
  return documentContentUrl(attachment.documentId, "download");
}

export function CommentAttachments({ attachments }: { attachments: ShipmentCommentAttachment[] }) {
  const [lightboxIndex, setLightboxIndex] = useState<number | null>(null);

  const images = useMemo(() => attachments.filter(isImageAttachment), [attachments]);
  const files = useMemo(
    () => attachments.filter((attachment) => !isImageAttachment(attachment)),
    [attachments],
  );

  if (attachments.length === 0) return null;

  return (
    <div className="mt-2 space-y-2">
      {images.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {images.map((image, index) => (
            <button
              key={image.documentId}
              type="button"
              className="group/attachment relative size-24 overflow-hidden rounded-lg border border-border bg-muted transition-shadow hover:shadow-md"
              onClick={() => setLightboxIndex(index)}
              aria-label={`Open ${image.originalName ?? image.fileName}`}
            >
              <img
                src={attachmentViewUrl(image)}
                alt={image.originalName ?? image.fileName}
                loading="lazy"
                className="size-full object-cover transition-transform duration-200 group-hover/attachment:scale-105"
              />
            </button>
          ))}
        </div>
      )}
      {files.length > 0 && (
        <div className="flex flex-wrap gap-1.5">
          {files.map((file) => (
            <a
              key={file.documentId}
              href={attachmentDownloadUrl(file)}
              target="_blank"
              rel="noopener noreferrer"
              className="flex max-w-60 items-center gap-2 rounded-md border border-border bg-card px-2 py-1.5 transition-colors hover:bg-muted"
            >
              <DocumentFileTypeIcon fileType={file.mimeType} fileName={file.fileName} size="sm" />
              <span className="min-w-0 flex-1">
                <span className="block truncate text-xs font-medium">
                  {file.originalName ?? file.fileName}
                </span>
                <span className="block text-2xs text-muted-foreground">
                  {formatFileSize(file.fileSize)}
                </span>
              </span>
            </a>
          ))}
        </div>
      )}
      <AttachmentLightbox
        images={images}
        index={lightboxIndex}
        onClose={() => setLightboxIndex(null)}
        onNavigate={setLightboxIndex}
      />
    </div>
  );
}

function AttachmentLightbox({
  images,
  index,
  onClose,
  onNavigate,
}: {
  images: ShipmentCommentAttachment[];
  index: number | null;
  onClose: () => void;
  onNavigate: (index: number) => void;
}) {
  const current = index != null ? images[index] : null;

  const goPrevious = useCallback(() => {
    if (index == null || images.length < 2) return;
    onNavigate((index - 1 + images.length) % images.length);
  }, [index, images.length, onNavigate]);

  const goNext = useCallback(() => {
    if (index == null || images.length < 2) return;
    onNavigate((index + 1) % images.length);
  }, [index, images.length, onNavigate]);

  useEffect(() => {
    if (index == null) return;
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "ArrowLeft") goPrevious();
      if (event.key === "ArrowRight") goNext();
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [index, goPrevious, goNext]);

  return (
    <Dialog open={current != null} onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-3xl">
        {current && (
          <>
            <DialogHeader>
              <DialogTitle className="truncate pr-8 text-sm">
                {current.originalName ?? current.fileName}
              </DialogTitle>
              <DialogDescription className="sr-only">Image attachment preview</DialogDescription>
            </DialogHeader>
            <div className="relative flex max-h-[70vh] items-center justify-center overflow-hidden rounded-md bg-muted/40">
              <img
                src={attachmentViewUrl(current)}
                alt={current.originalName ?? current.fileName}
                className="max-h-[70vh] max-w-full object-contain"
              />
              {images.length > 1 && (
                <>
                  <Button
                    type="button"
                    variant="secondary"
                    size="icon-sm"
                    className="absolute left-2 rounded-full"
                    onClick={goPrevious}
                    aria-label="Previous image"
                  >
                    <ChevronLeftIcon className="size-4" />
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    size="icon-sm"
                    className="absolute right-2 rounded-full"
                    onClick={goNext}
                    aria-label="Next image"
                  >
                    <ChevronRightIcon className="size-4" />
                  </Button>
                </>
              )}
            </div>
            <div className="flex items-center justify-between">
              <span className="text-2xs text-muted-foreground">
                {formatFileSize(current.fileSize)}
                {images.length > 1 && index != null && ` · ${index + 1} of ${images.length}`}
              </span>
              <div className="flex items-center gap-1">
                <Button
                  type="button"
                  variant="outline"
                  size="xs"
                  onClick={() => window.open(attachmentDownloadUrl(current), "_blank", "noopener")}
                >
                  <DownloadIcon className="mr-1 size-3" />
                  Download
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="xs"
                  onClick={() => window.open(attachmentViewUrl(current), "_blank", "noopener")}
                >
                  <ExternalLinkIcon className="mr-1 size-3" />
                  Open
                </Button>
              </div>
            </div>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
