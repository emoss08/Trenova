import { DocumentFileTypeIcon } from "@/components/documents/document-file-type-icon";
import { Button } from "@trenova/shared/components/ui/button";
import { Spinner } from "@trenova/shared/components/ui/spinner";
import { formatFileSize } from "@trenova/shared/lib/utils";
import { cn } from "@trenova/shared/lib/utils";
import type { UploadState } from "@/hooks/use-upload-with-progress";
import { RotateCcwIcon, XIcon } from "lucide-react";
import { AnimatePresence, m } from "motion/react";
import { useEffect, useState } from "react";

function useObjectUrl(file: File | null): string | null {
  const [url, setUrl] = useState<string | null>(null);

  useEffect(() => {
    if (!file) {
      setUrl(null);
      return;
    }
    const objectUrl = URL.createObjectURL(file);
    setUrl(objectUrl);
    return () => {
      URL.revokeObjectURL(objectUrl);
    };
  }, [file]);

  return url;
}

function AttachmentThumbnail({ upload }: { upload: UploadState }) {
  const isImage = upload.file.type.startsWith("image/");
  const objectUrl = useObjectUrl(isImage ? upload.file : null);
  const [imageFailed, setImageFailed] = useState(false);

  useEffect(() => {
    setImageFailed(false);
  }, [objectUrl]);

  if (!isImage || imageFailed) {
    return (
      <DocumentFileTypeIcon fileType={upload.file.type} fileName={upload.file.name} size="sm" />
    );
  }

  return (
    <span className="flex size-8 shrink-0 items-center justify-center overflow-hidden rounded bg-muted">
      {objectUrl ? (
        <img
          src={objectUrl}
          alt=""
          aria-hidden
          className="size-full object-cover"
          onError={() => setImageFailed(true)}
        />
      ) : (
        <Spinner className="size-3" />
      )}
    </span>
  );
}

function UploadChip({
  upload,
  onCancel,
  onRetry,
  onRemove,
}: {
  upload: UploadState;
  onCancel: () => void;
  onRetry: () => void;
  onRemove: () => void;
}) {
  const isActive = upload.status === "uploading" || upload.status === "pending";
  const isError = upload.status === "error";

  return (
    <m.div
      layout
      initial={{ opacity: 0, scale: 0.95 }}
      animate={{ opacity: 1, scale: 1 }}
      exit={{ opacity: 0, scale: 0.95 }}
      transition={{ duration: 0.15, ease: "easeOut" }}
      className={cn(
        "relative flex w-52 items-center gap-2 overflow-hidden rounded-md border border-border bg-card px-2 py-1.5",
        isError && "border-red-500/50 bg-red-500/5",
      )}
    >
      <AttachmentThumbnail upload={upload} />
      <span className="min-w-0 flex-1">
        <span className="block truncate text-xs font-medium">{upload.file.name}</span>
        <span className={cn("block text-2xs", isError ? "text-red-500" : "text-muted-foreground")}>
          {isError
            ? (upload.error ?? "Upload failed")
            : isActive
              ? `${Math.round(upload.progress)}%`
              : formatFileSize(upload.file.size)}
        </span>
      </span>
      {isError && (
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          className="size-5 shrink-0"
          onClick={onRetry}
          aria-label={`Retry uploading ${upload.file.name}`}
        >
          <RotateCcwIcon className="size-3" />
        </Button>
      )}
      <Button
        type="button"
        variant="ghost"
        size="icon-xs"
        className="size-5 shrink-0 text-muted-foreground"
        onClick={isActive ? onCancel : onRemove}
        aria-label={`Remove ${upload.file.name}`}
      >
        <XIcon className="size-3" />
      </Button>
      {isActive && (
        <span className="absolute inset-x-0 bottom-0 h-0.5 bg-muted">
          <span
            className="block h-full bg-brand transition-[width] duration-200"
            style={{ width: `${Math.max(4, upload.progress)}%` }}
          />
        </span>
      )}
    </m.div>
  );
}

export function ComposerAttachments({
  uploads,
  onCancel,
  onRetry,
  onRemove,
}: {
  uploads: UploadState[];
  onCancel: (id: string) => void;
  onRetry: (id: string) => void;
  onRemove: (id: string) => void;
}) {
  if (uploads.length === 0) return null;

  return (
    <div className="mb-2 flex flex-wrap gap-1.5">
      <AnimatePresence initial={false}>
        {uploads.map((upload) => (
          <UploadChip
            key={upload.id}
            upload={upload}
            onCancel={() => onCancel(upload.id)}
            onRetry={() => onRetry(upload.id)}
            onRemove={() => onRemove(upload.id)}
          />
        ))}
      </AnimatePresence>
    </div>
  );
}
