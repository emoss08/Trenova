import { useDocumentUpload } from "@/hooks/use-document-upload";
import { sortUploadFiles, type RejectedFile } from "@/lib/document-upload-files";
import { quarantinedUploadMessage } from "@/lib/upload-refusal";
import { PENDING_DOCUMENTS_REFETCH_MS } from "@/lib/resource-documents";
import { documentContentUrl, documentService } from "@/services/document";
import { UploadCloud01Icon, XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Progress } from "@trenova/shared/components/ui/progress";
import type {
  DocumentPreviewStatus,
  DocumentProcessingProfile,
} from "@trenova/shared/types/document";
import { useQuery } from "@tanstack/react-query";
import { useLatestCallback } from "@trenova/shared/hooks/use-latest-callback";
import { useObjectUrl } from "@trenova/shared/hooks/use-object-url";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatFileSize } from "@trenova/shared/lib/utils";
import { useEffect, useId, useRef, useState, type DragEvent, type ReactNode } from "react";
import { DocumentThumbnail } from "./document-thumbnail";

/** A document the field holds: one already on the record, or the one just uploaded. */
export type AttachedFile = {
  id: string;
  name: string;
  /** Bytes; 0 when unknown, and then no size is shown. */
  size: number;
  /** The MIME type; null when the read did not carry it. */
  fileType: string | null;
  /**
   * Where the server's preview stands; null only when the record names a document its read did
   * not carry, and then the preview is tried.
   */
  previewStatus: DocumentPreviewStatus | null;
};

export const FILE_DROP_DEFAULT_ACCEPT = ".pdf,.jpg,.jpeg,.png,.webp";
const DEFAULT_MAX_FILE_SIZE = 50 * 1024 * 1024;

export type FileDropFieldProps = {
  /** The field's name ("Document"). */
  label: string;
  /** Muted words after the name ("needed before this credential can be verified"). */
  note?: string | null;
  value: AttachedFile | null;
  onChange: (file: AttachedFile | null) => void;
  /** The record the upload is filed against. */
  resourceId: string;
  resourceType: string;
  /** Extensions and MIME types, as an input's `accept`. Pass `acceptLabel` with it. */
  accept?: string;
  /** The accepted types in words ("PDF, JPG, PNG or WEBP."). */
  acceptLabel?: string;
  maxFileSize?: number;
  /** What the drop zone asks for ("Drop the document here"). */
  prompt?: string;
  uploadMetadata?: Record<string, string>;
  processingProfile?: DocumentProcessingProfile;
  /** A message from the form about this field (required, refused by the server). */
  error?: string | null;
  disabled?: boolean;
  /**
   * Told when a file starts and stops uploading, so the form holds its save until the document
   * is attached: a save mid-upload would go without it.
   */
  onUploadingChange?: (uploading: boolean) => void;
  className?: string;
};

type Problem = { kind: "rejected"; file: RejectedFile } | { kind: "failed"; message: string };

/**
 * One document on a form: a zone to drop it on or choose it from, its upload with progress and
 * a cancel, then the attached file with a remove. Removing only takes it off the form; the
 * uploaded document stays on the record's files.
 */
export function FileDropField({
  label,
  note,
  value,
  onChange,
  resourceId,
  resourceType,
  accept = FILE_DROP_DEFAULT_ACCEPT,
  acceptLabel,
  maxFileSize = DEFAULT_MAX_FILE_SIZE,
  prompt,
  uploadMetadata,
  processingProfile,
  error,
  disabled = false,
  onUploadingChange,
  className,
}: FileDropFieldProps) {
  const t = useT();
  const rt = useRichT();
  const labelId = useId();
  const messageId = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const [over, setOver] = useState(false);
  const [problem, setProblem] = useState<Problem | null>(null);
  const [picked, setPicked] = useState<{ documentId: string; file: File } | null>(null);

  const { uploads, uploadFiles, cancelUpload, removeUpload } = useDocumentUpload({
    resourceId,
    resourceType,
    uploadMetadata,
    processingProfile,
    onSuccess: (document, upload) => {
      removeUpload(upload.id);
      setPicked({ documentId: document.id, file: upload.file });
      onChange({
        id: document.id,
        name: document.originalName,
        size: document.fileSize,
        fileType: document.fileType,
        previewStatus: document.previewStatus,
      });
    },
    onError: (failure) => {
      setProblem({ kind: "failed", message: failure.message });
    },
  });
  const active = uploads.find(
    (upload) =>
      upload.status !== "success" && upload.status !== "error" && upload.status !== "quarantined",
  );
  const refused = uploads.find((upload) => upload.status === "quarantined");
  const uploading = active !== undefined;
  const tellUploading = useLatestCallback((value: boolean) => onUploadingChange?.(value));
  useEffect(() => {
    tellUploading(uploading);
  }, [uploading, tellUploading]);
  useEffect(() => () => tellUploading(false), [tellUploading]);

  const take = (files: Iterable<File>) => {
    if (disabled) return;
    const { accepted, rejected } = sortUploadFiles(files, { accept, maxFileSize, multiple: false });
    const file = accepted[0];
    if (!file) {
      setProblem(rejected[0] ? { kind: "rejected", file: rejected[0] } : null);
      return;
    }
    setProblem(null);
    for (const upload of uploads) removeUpload(upload.id);
    uploadFiles([file]);
  };

  const onDrop = (event: DragEvent<HTMLDivElement>) => {
    event.preventDefault();
    setOver(false);
    take(event.dataTransfer.files);
  };

  const message = error
    ? error
    : refused
      ? (refused.error ?? quarantinedUploadMessage())
      : problem?.kind === "rejected"
        ? problem.file.reason === "size"
          ? t("{0} is larger than {1}.", problem.file.file.name, formatFileSize(maxFileSize))
          : t("{0} is not a type this field takes.", problem.file.file.name)
        : problem?.kind === "failed"
          ? t("The upload failed: {0}", problem.message)
          : null;

  return (
    <div
      role="group"
      aria-labelledby={labelId}
      aria-describedby={message ? messageId : undefined}
      className={cn("flex flex-col gap-2", className)}
    >
      <p id={labelId} className="text-sm font-medium">
        {label}
        {note ? (
          <>
            {" "}
            <span className="text-xs font-normal text-muted-foreground">{note}</span>
          </>
        ) : null}
      </p>

      {value ? (
        <FileChip
          name={value.name}
          fileType={value.fileType}
          localFile={picked?.documentId === value.id ? picked.file : null}
          documentId={value.id}
          previewStatus={value.previewStatus}
          meta={value.size > 0 ? formatFileSize(value.size) : null}
          action={
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label={t("Remove {0}", value.name)}
              disabled={disabled}
              onClick={() => onChange(null)}
            >
              <XCloseIcon className="size-3.5" />
            </Button>
          }
        />
      ) : active ? (
        <FileChip
          name={active.file.name}
          fileType={active.file.type || null}
          localFile={active.file}
          documentId={null}
          previewStatus={null}
          meta={t("{0} · {1}%", formatFileSize(active.file.size), Math.round(active.progress))}
          progress={active.progress}
          action={
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              aria-label={t("Cancel the upload of {0}", active.file.name)}
              onClick={() => cancelUpload(active.id)}
            >
              <XCloseIcon className="size-3.5" />
            </Button>
          }
        />
      ) : (
        <div
          data-over={over || undefined}
          onDragOver={(event) => {
            event.preventDefault();
            if (!disabled) setOver(true);
          }}
          onDragLeave={() => setOver(false)}
          onDrop={onDrop}
          className={cn(
            "flex flex-col items-center gap-1.5 rounded-md border border-dashed border-border-strong bg-sunken px-4 py-4.5 text-center text-xs text-muted-foreground transition-colors",
            over && "border-ring bg-brand-subtle",
            message && !over && "border-danger-border",
            disabled && "pointer-events-none opacity-60",
          )}
        >
          <UploadCloud01Icon aria-hidden className="size-5 text-foreground-subtle" />
          <p>
            {rt(
              "<b>{0}</b> or <choose>choose a file</choose>.",
              {
                b: (chunks) => <b className="font-medium text-foreground">{chunks}</b>,
                choose: (chunks) => (
                  <button
                    type="button"
                    disabled={disabled}
                    onClick={() => inputRef.current?.click()}
                    className="rounded-xs ui-focus-ring font-medium text-brand hover:underline"
                  >
                    {chunks}
                  </button>
                ),
              },
              prompt ?? t("Drop the document here"),
            )}
          </p>
          <p>{acceptLabel ?? t("PDF, JPG, PNG or WEBP.")}</p>
          <input
            ref={inputRef}
            type="file"
            accept={accept}
            className="hidden"
            aria-label={t("Choose a file")}
            disabled={disabled}
            onChange={(event) => {
              if (event.target.files) take(event.target.files);
              event.target.value = "";
            }}
          />
        </div>
      )}

      {message ? (
        <p id={messageId} role="alert" className="text-xs text-danger-foreground">
          {message}
        </p>
      ) : null}
    </div>
  );
}

/**
 * The file the field holds. A file picked here shows itself (an image from its own bytes);
 * a stored document shows the server's preview, a spinner while it is drawn (read again until it
 * is), and anything without one its file-type icon.
 */
function FileChip({
  name,
  fileType,
  localFile,
  documentId,
  previewStatus,
  meta,
  progress,
  action,
}: {
  name: string;
  fileType: string | null;
  localFile: File | null;
  documentId: string | null;
  /** Null for a file still uploading, or a stored document whose read did not say. */
  previewStatus: DocumentPreviewStatus | null;
  meta: string | null;
  progress?: number;
  action: ReactNode;
}) {
  const localImage = localFile?.type.startsWith("image/") ? localFile : null;
  const localUrl = useObjectUrl(localImage);
  const waiting = !localImage && documentId !== null && previewStatus === "Pending";
  const { data: stored } = useQuery({
    queryKey: ["document-preview-status", documentId],
    queryFn: ({ signal }) => documentService.getById(documentId ?? "", { signal }),
    enabled: waiting,
    refetchInterval: (query) =>
      query.state.data?.previewStatus === "Pending" ? PENDING_DOCUMENTS_REFETCH_MS : false,
  });
  const status = waiting ? (stored?.previewStatus ?? previewStatus) : previewStatus;
  const previewUrl = localImage
    ? localUrl
    : documentId
      ? documentContentUrl(documentId, "preview")
      : null;

  return (
    <div className="grid animate-expand-in grid-cols-[2.5rem_minmax(0,1fr)_auto] items-center gap-2.5 rounded-md border bg-card py-2 pr-2 pl-2.5 motion-reduce:animate-none">
      <DocumentThumbnail
        fileType={fileType}
        fileName={name}
        previewStatus={localImage ? "Ready" : documentId ? status : "Unsupported"}
        previewUrl={previewUrl}
        size="sm"
        className="size-10 rounded-sm border border-border-subtle bg-sunken"
      />
      <div className="min-w-0">
        <div className="truncate text-sm font-medium" title={name}>
          {name}
        </div>
        {meta ? <div className="text-xs text-muted-foreground tabular-nums">{meta}</div> : null}
        {progress !== undefined ? <Progress size="sm" value={progress} className="mt-1.5" /> : null}
      </div>
      {action}
    </div>
  );
}
