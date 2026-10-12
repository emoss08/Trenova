import { describeApiError } from "@/lib/api-error-message";
import type { InboundReplyAttachable, InboundReplyAttachableSource } from "@/lib/graphql/inbox";
import { queries } from "@/lib/queries";
import {
  attachableSections,
  fileFromAttachable,
  fitsBudget,
  type ReplyFile,
} from "@/lib/triage/reply-files";
import type { UploadState } from "@/types/upload";
import { useQuery } from "@tanstack/react-query";
import {
  CheckIcon,
  PaperclipIcon,
  RefreshCw01Icon,
  Upload01Icon,
  XCloseIcon,
} from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@trenova/shared/components/ui/popover";
import { Progress } from "@trenova/shared/components/ui/progress";
import type { TranslateFn } from "@trenova/shared/i18n/runtime";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatFileSize } from "@trenova/shared/lib/utils";
import { type ReactNode, useRef, useState } from "react";
import { FileBadge } from "../file-badge";

function sourceHeading(t: TranslateFn, source: InboundReplyAttachableSource): string {
  switch (source) {
    case "Message":
      return t("On this message");
    case "Shipment":
      return t("On the shipment");
    case "Customer":
      return t("On the customer");
    case "Carrier":
      return t("On the carrier");
  }
}

const ROW = "animate-rise flex min-h-8 min-w-0 items-center gap-2 py-1 pr-1 pl-2";

/**
 * The files a reply carries, one line each in a single box, so a reply with a
 * handful of files stays short. The summary closes the box: the count and,
 * while writing, how much of the email's budget they spend.
 */
export function ReplyFileList({ summary, children }: { summary?: ReactNode; children: ReactNode }) {
  const t = useT();

  return (
    <div className="border-border min-w-0 overflow-hidden rounded-lg border">
      <ul aria-label={t("Files")} className="divide-border-subtle flex flex-col divide-y">
        {children}
      </ul>
      {summary !== undefined && (
        <div className="border-border-subtle text-foreground-subtle flex h-7 items-center justify-between gap-2 border-t px-2 text-xs tabular-nums">
          {summary}
        </div>
      )}
    </div>
  );
}

function RemoveButton({ label, onClick }: { label: string; onClick: () => void }) {
  return (
    <Button size="icon-xs" variant="ghost" aria-label={label} onClick={onClick}>
      <XCloseIcon aria-hidden className="size-3" />
    </Button>
  );
}

/** A file the reply carries: its kind, name and size, and a way to take it off. */
export function ReplyFileChip({
  file,
  onRemove,
}: {
  file: ReplyFile;
  /** Absent while the reply is sending: a file on its way cannot be taken back. */
  onRemove?: (documentId: string) => void;
}) {
  const t = useT();

  return (
    <li title={file.fileName} className={cn(ROW, onRemove === undefined && "pr-2")}>
      <FileBadge fileName={file.fileName} compact />
      <span className="min-w-0 flex-1 truncate text-sm">{file.fileName}</span>
      <span className="text-foreground-subtle shrink-0 text-xs tabular-nums">
        {formatFileSize(file.byteSize)}
      </span>
      {onRemove !== undefined && (
        <RemoveButton
          label={t("Remove {0}", file.fileName)}
          onClick={() => onRemove(file.documentId)}
        />
      )}
    </li>
  );
}

/** A file on its way up from the computer, or the reason it did not make it. */
export function UploadingChip({
  upload,
  onCancel,
  onRetry,
}: {
  upload: UploadState;
  onCancel: (id: string) => void;
  onRetry: (id: string) => void;
}) {
  const t = useT();
  const failed = upload.status === "error";
  const reason = upload.error ?? t("The file could not be uploaded");

  return (
    <li title={failed ? reason : upload.file.name} className={ROW}>
      <FileBadge fileName={upload.file.name} dimmed compact />
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-sm">{upload.file.name}</span>
        {failed && <span className="text-danger truncate text-xs">{reason}</span>}
      </span>
      {failed ? (
        <Button
          size="icon-xs"
          variant="ghost"
          aria-label={t("Try again")}
          onClick={() => onRetry(upload.id)}
        >
          <RefreshCw01Icon aria-hidden className="size-3" />
        </Button>
      ) : (
        <Progress
          size="sm"
          showLabel
          value={upload.progress}
          className="w-20 shrink-0"
          aria-label={t("Uploading {0}", upload.file.name)}
        />
      )}
      <RemoveButton
        label={failed ? t("Remove {0}", upload.file.name) : t("Cancel upload")}
        onClick={() => onCancel(upload.id)}
      />
    </li>
  );
}

/** A file refused before it was sent anywhere, because it would not fit in the email. */
export function RefusedFileChip({
  name,
  reason,
  onDismiss,
}: {
  name: string;
  reason: string;
  onDismiss: () => void;
}) {
  const t = useT();

  return (
    <li title={reason} className={ROW}>
      <FileBadge fileName={name} dimmed compact />
      <span className="flex min-w-0 flex-1 flex-col">
        <span className="truncate text-sm">{name}</span>
        <span className="text-danger line-clamp-2 text-xs">{reason}</span>
      </span>
      <RemoveButton label={t("Remove {0}", name)} onClick={onDismiss} />
    </li>
  );
}

function AttachableRow({
  document,
  attached,
  fits,
  limitLabel,
  onToggle,
}: {
  document: InboundReplyAttachable;
  attached: boolean;
  fits: boolean;
  limitLabel: string;
  onToggle: (document: InboundReplyAttachable) => void;
}) {
  const t = useT();
  const blocked = !attached && !fits;
  const size = formatFileSize(document.byteSize);

  return (
    <button
      type="button"
      role="menuitemcheckbox"
      aria-checked={attached}
      disabled={blocked}
      onClick={() => onToggle(document)}
      className="hover:bg-surface-hover ui-focus-ring flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left disabled:cursor-not-allowed disabled:opacity-50"
    >
      <FileBadge fileName={document.fileName} />
      <span className="flex min-w-0 flex-1 flex-col leading-snug">
        <span className="truncate text-sm font-medium">{document.fileName}</span>
        <span className="text-foreground-subtle truncate text-xs tabular-nums">
          {blocked
            ? t("{0} · over the {1} a reply can carry", size, limitLabel)
            : document.documentType === ""
              ? size
              : t("{0} · {1}", document.documentType, size)}
        </span>
      </span>
      <CheckIcon
        aria-hidden
        className={cn("size-3.5 shrink-0", attached ? "text-foreground" : "invisible")}
      />
    </button>
  );
}

/**
 * Attach: the documents already stored on the message, its shipment, customer
 * and carrier, grouped by where they are, and a file from the computer, which
 * is uploaded to the shipment (or the message) first so it stays on file.
 */
export function AttachMenu({
  messageId,
  files,
  limitBytes,
  disabled,
  onToggle,
  onUpload,
}: {
  messageId: string;
  files: readonly ReplyFile[];
  limitBytes: number | null;
  disabled: boolean;
  onToggle: (file: ReplyFile) => void;
  onUpload: (files: File[]) => void;
}) {
  const t = useT();
  const [open, setOpen] = useState(false);
  const input = useRef<HTMLInputElement>(null);
  const attachables = useQuery({ ...queries.inbox.replyAttachables(messageId), enabled: open });
  const sections = attachableSections(attachables.data?.documents ?? []);
  const attached = new Set(files.map((file) => file.documentId));
  const limit = limitBytes ?? attachables.data?.limitBytes ?? null;
  const limitLabel = limit === null ? "" : formatFileSize(limit);

  return (
    <>
      <input
        ref={input}
        type="file"
        multiple
        hidden
        aria-hidden
        tabIndex={-1}
        onChange={(event) => {
          const chosen = Array.from(event.target.files ?? []);
          event.target.value = "";
          if (chosen.length > 0) {
            onUpload(chosen);
          }
        }}
      />
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger
          disabled={disabled}
          render={<Button size="sm" variant="ghost" className="data-popup-open:bg-surface-hover" />}
        >
          <PaperclipIcon aria-hidden className="size-3" />
          {t("Attach")}
        </PopoverTrigger>
        <PopoverContent
          side="top"
          align="start"
          className="w-84 gap-0 p-1"
          aria-label={t("Attach files")}
        >
          <div role="menu" className="flex max-h-80 flex-col overflow-y-auto">
            {attachables.isLoading && (
              <span className="text-foreground-subtle px-2 py-1.5 text-sm">{t("Loading…")}</span>
            )}
            {attachables.isError && (
              <span className="text-danger px-2 py-1.5 text-sm">
                {t("Couldn't load the files: {0}", describeApiError(attachables.error))}
              </span>
            )}
            {attachables.isSuccess && sections.length === 0 && (
              <span className="text-foreground-subtle px-2 py-1.5 text-sm">
                {t("Nothing is stored on this message or its records yet.")}
              </span>
            )}
            {sections.map((section) => (
              <div key={section.source} role="group" className="flex flex-col">
                <div className="text-foreground-subtle px-2 pt-1.5 pb-1 text-xs">
                  {sourceHeading(t, section.source)}
                </div>
                {section.documents.map((document) => {
                  const isAttached = attached.has(document.id);
                  return (
                    <AttachableRow
                      key={document.id}
                      document={document}
                      attached={isAttached}
                      fits={limit === null || fitsBudget(files, document.byteSize, limit)}
                      limitLabel={limitLabel}
                      onToggle={(picked) => onToggle(fileFromAttachable(picked))}
                    />
                  );
                })}
              </div>
            ))}
          </div>
          <div className="border-border-subtle mt-1 border-t pt-1">
            <button
              type="button"
              onClick={() => {
                setOpen(false);
                input.current?.click();
              }}
              className="hover:bg-surface-hover ui-focus-ring flex h-7 w-full items-center gap-2 rounded-md px-2 text-sm"
            >
              <Upload01Icon aria-hidden className="size-3" />
              {t("Upload from computer")}
            </button>
          </div>
        </PopoverContent>
      </Popover>
    </>
  );
}
