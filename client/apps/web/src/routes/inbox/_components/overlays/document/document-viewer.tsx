import "../../../_styles/triage-overlays.css";
import type { InboundAttachment, InboundMessageDetail } from "@/lib/graphql/inbox";
import { inbox } from "@/lib/queries/inbox";
import { senderName } from "@/lib/triage/message-state";
import { PdfFrame } from "@/components/elements/pdf-frame";
import { documentContentEndpoint, documentContentUrl } from "@/services/document";
import { useQuery } from "@tanstack/react-query";
import { CheckIcon, Download01Icon, XCloseIcon } from "@trenova/shared/components/icons";
import { Button } from "@trenova/shared/components/ui/button";
import { Dialog, DialogContent, DialogTitle } from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { cn, formatFileSize } from "@trenova/shared/lib/utils";
import { useState } from "react";
import { inlineErrorText } from "../../inline-error";
import { useTriageActions } from "../../use-triage-actions";

type Preview = "pdf" | "image" | "none";

function previewOf(attachment: InboundAttachment): Preview {
  if (attachment.documentId === null) {
    return "none";
  }
  const type = attachment.contentType.toLowerCase();
  if (type === "application/pdf" || attachment.fileName.toLowerCase().endsWith(".pdf")) {
    return "pdf";
  }

  return type.startsWith("image/") ? "image" : "none";
}

function extensionOf(fileName: string): string {
  const dot = fileName.lastIndexOf(".");
  return dot === -1 ? "" : fileName.slice(dot + 1);
}

type PageRow = { number: number; label: string };

function pagesOf(attachment: InboundAttachment): PageRow[] {
  if (attachment.pages.length > 0) {
    return attachment.pages.map((page) => ({ number: page.number, label: page.label }));
  }
  const count = Math.max(1, attachment.pageCount ?? attachment.reading?.pageCount ?? 1);

  return Array.from({ length: count }, (_, index) => ({ number: index + 1, label: "" }));
}

/**
 * One file of a message, opened: its pages down the side, the page itself in
 * the browser's own PDF viewer, and what Nova read from it. The file can be
 * filed on the matched shipment from here, or says where it already was.
 */
export function DocumentViewer({
  messageId,
  attachmentId,
  onClose,
}: {
  messageId: string;
  attachmentId: string;
  onClose: () => void;
}) {
  const t = useT();
  const message = useQuery(inbox.message(messageId)).data;
  const attachment = message?.attachments.find((row) => row.id === attachmentId);

  return (
    <Dialog open onOpenChange={(open) => (open ? undefined : onClose())}>
      <DialogContent
        size="2xl"
        showCloseButton={false}
        className="grid h-[min(720px,calc(100dvh-3rem))] grid-rows-[auto_minmax(0,1fr)] gap-0 overflow-hidden p-0"
      >
        {message === undefined || attachment === undefined ? (
          <>
            <DialogTitle className="sr-only">{t("Document")}</DialogTitle>
            <div className="flex flex-col gap-3 p-4">
              <Skeleton className="h-8 w-1/2" />
              <Skeleton className="h-full min-h-80 w-full" />
            </div>
          </>
        ) : (
          <ViewerBody message={message} attachment={attachment} onClose={onClose} />
        )}
      </DialogContent>
    </Dialog>
  );
}

function ViewerBody({
  message,
  attachment,
  onClose,
}: {
  message: InboundMessageDetail;
  attachment: InboundAttachment;
  onClose: () => void;
}) {
  const t = useT();
  const actions = useTriageActions();
  const [page, setPage] = useState(1);
  const [failed, setFailed] = useState(false);
  const [hovered, setHovered] = useState<number | null>(null);
  const [filing, setFiling] = useState(false);
  const [refusal, setRefusal] = useState<string | null>(null);

  const kind = failed ? "none" : previewOf(attachment);
  const pages = pagesOf(attachment);
  const reading = attachment.reading;
  const fields = reading?.fields ?? [];
  const extension = extensionOf(attachment.fileName);
  const shipment = message.matchedShipment;
  const filedOn = attachment.filedShipment ?? null;
  const current = pages.find((row) => row.number === page) ?? pages[0];

  const file = () => {
    if (shipment === null) {
      return;
    }
    setFiling(true);
    setRefusal(null);
    actions.file(message.id, { attachmentId: attachment.id, shipmentId: shipment.id }).then(
      () => {
        setFiling(false);
        onClose();
      },
      (error: unknown) => {
        setFiling(false);
        setRefusal(inlineErrorText(t, error));
      },
    );
  };

  return (
    <>
      <header className="border-border-subtle flex items-center gap-2.5 border-b py-2.5 pr-2.5 pl-3.5">
        <span
          aria-hidden
          className={cn(
            "border-border bg-sunken flex h-8 w-7 shrink-0 items-end justify-center rounded-xs border pb-1 font-mono text-3xs font-semibold uppercase",
            extension.toLowerCase() === "pdf" ? "text-danger" : "text-info",
          )}
        >
          {extension}
        </span>
        <div className="flex min-w-0 flex-col">
          <DialogTitle className="truncate text-base font-semibold">
            {attachment.fileName}
          </DialogTitle>
          <span className="text-foreground-subtle truncate text-sm">
            {t(
              "{0} · {1, plural, one {# page} other {# pages}} · from {2}",
              formatFileSize(attachment.byteSize),
              pages.length,
              senderName(message),
            )}
          </span>
        </div>
        <span className="flex-1" />
        {attachment.documentId !== null && (
          <Button
            variant="ghost"
            size="sm"
            nativeButton={false}
            render={
              <a
                href={documentContentUrl(attachment.documentId, "download")}
                download={attachment.fileName}
              />
            }
          >
            <Download01Icon className="size-3.5" />
            {t("Download")}
          </Button>
        )}
        <Button variant="ghost" size="icon-sm" aria-label={t("Close")} onClick={onClose}>
          <XCloseIcon className="size-3.5" />
        </Button>
      </header>

      <div className="grid min-h-0 grid-cols-[96px_minmax(0,1fr)_280px] max-md:grid-cols-[minmax(0,1fr)] max-md:grid-rows-[minmax(0,1fr)_auto]">
        <nav
          aria-label={t("Pages")}
          className="border-border-subtle flex flex-col gap-2 overflow-y-auto border-r p-3 max-md:hidden"
        >
          {pages.map((row) => (
            <button
              key={row.number}
              type="button"
              aria-current={row.number === page ? "page" : undefined}
              onClick={() => setPage(row.number)}
              className={cn(
                "ui-focus-ring flex flex-col items-center gap-1 rounded-lg p-1",
                row.number === page && "bg-surface-hover",
              )}
            >
              <span
                className={cn(
                  "tb-page-thumb text-foreground-subtle flex h-16.5 w-13 items-end justify-center rounded-xs border pb-1 font-mono text-2xs",
                  row.number === page ? "border-foreground" : "border-border",
                )}
              >
                {row.number}
              </span>
              {row.label !== "" && (
                <em className="text-foreground-subtle text-center text-2xs leading-tight not-italic">
                  {row.label}
                </em>
              )}
            </button>
          ))}
        </nav>

        <div className="bg-sunken grid min-h-0 place-items-center overflow-auto p-5">
          {kind === "pdf" && attachment.documentId !== null ? (
            <PdfFrame
              endpoint={documentContentEndpoint(attachment.documentId, "view")}
              title={attachment.fileName}
              page={page}
              className="ring-border aspect-[8.5/11] w-full max-w-160 rounded-xs ring-1"
            />
          ) : kind === "image" && attachment.documentId !== null ? (
            <img
              src={documentContentUrl(attachment.documentId, "view")}
              alt={attachment.fileName}
              onError={() => setFailed(true)}
              className="ring-border max-h-full max-w-full rounded-xs object-contain ring-1"
            />
          ) : (
            <div
              key={page}
              className="tb-paper border-border text-foreground-subtle animate-rise flex aspect-[8.5/11] w-full max-w-105 flex-col items-center justify-center gap-1 rounded-xs border font-mono text-xs"
            >
              <span className="text-foreground-muted text-sm">
                {current.label !== "" ? current.label : t("Page {0}", current.number)}
              </span>
              <span>
                {attachment.documentId === null
                  ? t("This file was not kept, so there is no preview")
                  : t("No preview for this file")}
              </span>
            </div>
          )}
        </div>

        <aside
          aria-label={t("What Nova read")}
          className="border-border-subtle flex flex-col gap-2.5 overflow-y-auto border-l px-4 py-3.5 max-md:border-t max-md:border-l-0"
        >
          <div className="text-foreground-subtle flex items-center gap-1.5 text-xs">
            <span>{t("What Nova read")}</span>
            {reading !== null && reading !== undefined && (
              <span className="ml-auto font-mono tabular-nums">
                {t("{0}% sure", Math.round(reading.confidence * 100))}
              </span>
            )}
          </div>
          {reading !== null && reading !== undefined && reading.documentType !== "" && (
            <p className="text-foreground text-sm font-medium">{reading.documentType}</p>
          )}
          {fields.length > 0 ? (
            <dl className="flex flex-col">
              {fields.map((field, index) => (
                <div
                  key={`${field.label}-${index}`}
                  onPointerEnter={() => setHovered(index)}
                  onPointerLeave={() => setHovered(null)}
                  className={cn(
                    "border-border-subtle -mx-1 flex justify-between gap-3 rounded-xs border-b px-1 py-1.75 text-sm last:border-b-0",
                    hovered === index && "bg-surface-hover",
                  )}
                >
                  <dt className="text-foreground-subtle">{field.label}</dt>
                  <dd className={cn("text-right", field.tone === "Warning" && "text-warning")}>
                    {field.value}
                  </dd>
                </div>
              ))}
            </dl>
          ) : (
            <p className="text-foreground-subtle text-sm">
              {reading === null || reading === undefined
                ? t("Nova has not read this file.")
                : t("Nothing structured on this file.")}
            </p>
          )}
          {filedOn !== null ? (
            <div className="bg-success-subtle text-success-subtle-foreground animate-confirm flex items-center gap-2 rounded-lg px-3 py-2.5 text-sm">
              <span className="bg-success text-foreground-on-solid flex size-5 items-center justify-center rounded-full">
                <CheckIcon aria-hidden className="size-2.75" strokeWidth={2.6} />
              </span>
              <span>
                {attachment.filedAs !== null && attachment.filedAs !== ""
                  ? t("Filed on {0} as {1}", filedOn.proNumber, attachment.filedAs)
                  : t("Filed on {0}", filedOn.proNumber)}
              </span>
            </div>
          ) : shipment !== null && attachment.documentId !== null ? (
            <Button
              size="sm"
              className="self-start"
              isLoading={filing}
              loadingText={t("Filing…")}
              onClick={file}
            >
              {t("File on {0}", shipment.proNumber)}
            </Button>
          ) : (
            <p className="text-foreground-subtle text-xs">
              {shipment === null
                ? t("Link the message to a shipment to file this.")
                : t("This file was not kept, so it can't be filed.")}
            </p>
          )}
          {refusal !== null && (
            <p role="alert" className="text-danger text-sm">
              {refusal}
            </p>
          )}
        </aside>
      </div>
    </>
  );
}
