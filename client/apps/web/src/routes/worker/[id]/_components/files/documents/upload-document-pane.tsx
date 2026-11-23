import { DocumentFileTypeIcon } from "@/components/documents/document-file-type-icon";
import { formatFileSize } from "@/components/documents/document-upload-zone";
import { WorkPane, WorkPaneBody, WorkPaneCancel, WorkPaneFooter } from "@/components/work-pane";
import { useDocumentUpload } from "@/hooks/use-document-upload";
import {
  DOCUMENT_UPLOAD_ACCEPT,
  DOCUMENT_UPLOAD_MAX_SIZE,
  sortUploadFiles,
  type RejectedFile,
} from "@/lib/document-upload-files";
import { quarantinedUploadMessage } from "@/lib/upload-refusal";
import type { UploadState } from "@/types/upload";
import { RefreshCcw01Icon, UploadCloud01Icon, XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { useRichT } from "@trenova/shared/i18n/rich";
import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import { cn } from "@trenova/shared/lib/utils";
import type { Document } from "@trenova/shared/types/document";
import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import { WORKER_DOCUMENT_RESOURCE } from "./use-worker-documents";

/** Upload states the session is done with; anything else is still moving. */
const SETTLED = new Set<UploadState["status"]>(["success", "error", "quarantined"]);

type Chosen = { key: string; file: File };

export function uploadStatusLine(upload: UploadState, t: TranslateFn): string {
  switch (upload.status) {
    case "uploading":
      return t("{0} · uploading {1}%", formatFileSize(upload.file.size), upload.progress);
    case "checking":
      return t("Checking for viruses");
    case "processing":
      return t("Compressing");
    case "uploaded":
    case "verifying":
    case "completing":
      return t("Finishing the upload");
    case "paused":
      return t("Paused, waiting for the connection");
    case "retrying":
    case "pending":
      return t("Waiting to upload");
    case "quarantined":
      return upload.error ?? quarantinedUploadMessage();
    case "error":
      return upload.error ?? t("The upload failed");
    default:
      return t("{0} · on file", formatFileSize(upload.file.size));
  }
}

export type UploadDocumentPaneProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workerId: string;
  workerName: string;
  /** Uploading a new version of one document: its lineage, and one file only. */
  lineageId: string | null;
  canUpload: boolean;
  /** Files dropped straight onto the table, handed to the pane as it opens. */
  dropped: readonly File[];
  onDroppedTaken: () => void;
  /** Told each document an upload became, so its row can flash when it lands. */
  onUploaded: (document: Document) => void;
  onDone: () => void;
};

/**
 * Uploads documents to the worker's file beside the table: choose or drop files, see each as a
 * chip, then upload them together. Each chip shows its progress and can be cancelled while it
 * runs, retried when it fails, or taken off. The pane closes once every file has landed.
 */
export function UploadDocumentPane({
  open,
  onOpenChange,
  workerId,
  workerName,
  lineageId,
  canUpload,
  dropped,
  onDroppedTaken,
  onUploaded,
  onDone,
}: UploadDocumentPaneProps) {
  const t = useT();
  const rt = useRichT();
  const inputId = useId();
  const inputRef = useRef<HTMLInputElement>(null);
  const [chosen, setChosen] = useState<Chosen[]>([]);
  const [rejected, setRejected] = useState<RejectedFile[]>([]);
  const [over, setOver] = useState(false);
  const [missing, setMissing] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const single = lineageId !== null;

  const { uploads, uploadFiles, cancelUpload, retryUpload, removeUpload, clearCompleted } =
    useDocumentUpload({
      resourceId: workerId,
      resourceType: WORKER_DOCUMENT_RESOURCE,
      uploadMetadata: lineageId ? { lineageId } : undefined,
      onSuccess: (document) => onUploaded(document),
    });

  const take = (files: Iterable<File>) => {
    const sorted = sortUploadFiles(files, { multiple: !single });
    setRejected(sorted.rejected);
    if (sorted.accepted.length === 0) return;
    setMissing(false);
    const next = sorted.accepted.map((file) => ({
      key: `${file.name}:${file.size}:${file.lastModified}:${Math.random()}`,
      file,
    }));
    setChosen((current) => (single ? next.slice(0, 1) : [...current, ...next]));
  };

  useEffect(() => {
    if (!open || dropped.length === 0) return;
    take(dropped);
    onDroppedTaken();
    // `take` reads only state setters and the pane's mode.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, dropped, onDroppedTaken]);

  useEffect(() => {
    if (open) return;
    setChosen([]);
    setRejected([]);
    setMissing(false);
    setSubmitted(false);
  }, [open]);

  const settled = uploads.length > 0 && uploads.every((upload) => SETTLED.has(upload.status));
  const allLanded = settled && uploads.every((upload) => upload.status === "success");
  useEffect(() => {
    if (!submitted || !allLanded) return;
    clearCompleted();
    setSubmitted(false);
    onDone();
  }, [submitted, allLanded, clearCompleted, onDone]);

  const running = uploads.some((upload) => !SETTLED.has(upload.status));
  const upload = () => {
    if (chosen.length === 0) {
      setMissing(true);
      return;
    }
    uploadFiles(chosen.map((item) => item.file));
    setChosen([]);
    setSubmitted(true);
  };

  return (
    <WorkPane
      open={open}
      onOpenChange={onOpenChange}
      title={single ? t("Upload a new version") : t("Upload documents")}
      description={
        single
          ? t("The file becomes the document's current version; the old one stays in its history.")
          : t("Adds them to {0}'s file.", workerName)
      }
    >
      <WorkPaneBody className="text-sm">
        <div
          role="group"
          aria-label={t("Files to upload")}
          onDragOver={(event) => {
            event.preventDefault();
            setOver(true);
          }}
          onDragLeave={() => setOver(false)}
          onDrop={(event) => {
            event.preventDefault();
            setOver(false);
            if (canUpload) take(Array.from(event.dataTransfer.files));
          }}
          className={cn(
            "flex flex-col items-center gap-1 rounded-md border border-dashed border-border-strong px-4 py-5 text-center transition-colors",
            over && "border-ring bg-brand-subtle",
            missing && "border-danger-border",
          )}
        >
          <UploadCloud01Icon className="mb-1 size-5 text-muted-foreground" />
          <p className="text-sm text-muted-foreground">
            {rt(
              single
                ? "<b>Drop the new version here</b> or <pick>choose a file</pick>."
                : "<b>Drop documents here</b> or <pick>choose files</pick>.",
              {
                b: (chunk) => <b className="font-medium text-foreground">{chunk}</b>,
                pick: (chunk) => (
                  <button
                    type="button"
                    disabled={!canUpload}
                    onClick={() => inputRef.current?.click()}
                    className="rounded-sm ui-focus-ring text-sm text-brand hover:underline"
                  >
                    {chunk}
                  </button>
                ),
              },
            )}
          </p>
          <p className="text-xs text-muted-foreground">
            {t(
              "PDF, images, Word, Excel and text files, up to {0} each.",
              formatFileSize(DOCUMENT_UPLOAD_MAX_SIZE),
            )}
          </p>
          <input
            ref={inputRef}
            id={inputId}
            type="file"
            className="sr-only"
            tabIndex={-1}
            accept={DOCUMENT_UPLOAD_ACCEPT}
            multiple={!single}
            onChange={(event) => {
              if (event.target.files) take(Array.from(event.target.files));
              event.target.value = "";
            }}
          />
        </div>
        {missing ? (
          <p role="alert" className="-mt-2 text-xs text-danger-foreground">
            {t("Add a file before you upload.")}
          </p>
        ) : null}
        {rejected.length > 0 ? (
          <ul role="alert" className="-mt-2 flex flex-col gap-0.5 text-xs text-danger-foreground">
            {rejected.map(({ file, reason }) => (
              <li key={`${file.name}:${file.size}`}>
                {reason === "size"
                  ? t(
                      "{0} is larger than {1}.",
                      file.name,
                      formatFileSize(DOCUMENT_UPLOAD_MAX_SIZE),
                    )
                  : t("{0} is not a kind of file that can be uploaded.", file.name)}
              </li>
            ))}
          </ul>
        ) : null}
        {chosen.length > 0 || uploads.length > 0 ? (
          <ul className="flex flex-col gap-1.5" aria-label={t("Files")}>
            {uploads.map((item) => (
              <FileChip
                key={item.id}
                file={item.file}
                line={uploadStatusLine(item, t)}
                tone={
                  item.status === "error" || item.status === "quarantined"
                    ? "danger"
                    : item.status === "paused"
                      ? "warning"
                      : "none"
                }
                progress={SETTLED.has(item.status) ? null : item.progress}
                actions={
                  <>
                    {item.status === "error" ? (
                      <Button
                        type="button"
                        size="icon-xs"
                        variant="ghost"
                        aria-label={t("Try {0} again", item.file.name)}
                        onClick={() => retryUpload(item.id)}
                      >
                        <RefreshCcw01Icon className="size-3.5" />
                      </Button>
                    ) : null}
                    <Button
                      type="button"
                      size="icon-xs"
                      variant="ghost"
                      aria-label={
                        SETTLED.has(item.status)
                          ? t("Take {0} off the list", item.file.name)
                          : t("Cancel the upload of {0}", item.file.name)
                      }
                      onClick={() =>
                        SETTLED.has(item.status) ? removeUpload(item.id) : cancelUpload(item.id)
                      }
                    >
                      <XCloseIcon className="size-3.5" />
                    </Button>
                  </>
                }
              />
            ))}
            {chosen.map((item) => (
              <FileChip
                key={item.key}
                file={item.file}
                line={t("{0} · ready", formatFileSize(item.file.size))}
                tone="none"
                progress={null}
                actions={
                  <Button
                    type="button"
                    size="icon-xs"
                    variant="ghost"
                    aria-label={t("Take {0} off the list", item.file.name)}
                    onClick={() =>
                      setChosen((current) => current.filter((entry) => entry.key !== item.key))
                    }
                  >
                    <XCloseIcon className="size-3.5" />
                  </Button>
                }
              />
            ))}
          </ul>
        ) : null}
      </WorkPaneBody>
      <WorkPaneFooter>
        <WorkPaneCancel>{running || settled ? t("Close") : t("Cancel")}</WorkPaneCancel>
        <Button
          type="button"
          size="sm"
          disabled={!canUpload || (chosen.length === 0 && running)}
          isLoading={running && chosen.length === 0}
          onClick={upload}
        >
          {chosen.length > 1 ? t("Upload {0} files", chosen.length) : t("Upload")}
        </Button>
      </WorkPaneFooter>
    </WorkPane>
  );
}

function FileChip({
  file,
  line,
  tone,
  progress,
  actions,
}: {
  file: File;
  line: string;
  tone: "none" | "warning" | "danger";
  progress: number | null;
  actions: ReactNode;
}) {
  return (
    <li className="grid grid-cols-[2rem_minmax(0,1fr)_auto] items-center gap-2.5 rounded-md border border-border-subtle bg-card p-2 motion-safe:animate-expand-in">
      <DocumentFileTypeIcon
        fileType={file.type || "application/octet-stream"}
        fileName={file.name}
        size="sm"
      />
      <span className="min-w-0">
        <span className="block truncate text-sm font-medium" title={file.name}>
          {file.name}
        </span>
        <span
          className={cn(
            "block truncate text-xs tabular-nums",
            tone === "danger"
              ? "text-danger-foreground"
              : tone === "warning"
                ? "text-warning-foreground"
                : "text-muted-foreground",
          )}
        >
          {line}
        </span>
        {progress !== null ? (
          <span className="mt-1 block h-1 overflow-hidden rounded-full bg-surface-active">
            <span
              className="block h-full bg-brand transition-[width] duration-300 ease-settle"
              style={{ width: `${progress}%` }}
            />
          </span>
        ) : null}
      </span>
      <span className="flex items-center">{actions}</span>
    </li>
  );
}
