import { DocumentFileTypeIcon } from "@/components/documents/document-file-type-icon";
import type { InboundAttachment, InboundShipmentRef } from "@/lib/graphql/inbox";
import { apiService } from "@/services/api";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { formatFileSize } from "@trenova/shared/lib/utils";
import { ExternalLinkIcon, ScanTextIcon } from "lucide-react";
import { toast } from "sonner";

function attachmentKindLabel(
  t: (value: string) => string,
  kind: InboundAttachment["kind"],
): string {
  switch (kind) {
    case "RateConfirmation":
      return t("Rate confirmation");
    case "ProofOfDelivery":
      return t("Proof of delivery");
    case "Invoice":
      return t("Invoice");
    case "BillOfLading":
      return t("Bill of lading");
    case "Other":
      return t("Other");
    case "Unknown":
      return t("Not read");
  }
}

async function openDocument(documentId: string, failed: string) {
  try {
    const url = await apiService.documentService.getViewUrl(documentId);
    window.open(url, "_blank", "noopener,noreferrer");
  } catch {
    toast.error(failed);
  }
}

/**
 * The files that came with the message, and what became of each.
 *
 * A file the pipeline accepted became a document: it can be opened, read as
 * the desk read it, and filed on the matched shipment. A file it refused says
 * why, because an attachment that silently vanished is one the sender will
 * swear they sent.
 */
export function AttachmentList({
  attachments,
  shipment,
  busy,
  onReview,
  onAttach,
}: {
  attachments: InboundAttachment[];
  shipment: InboundShipmentRef | null;
  busy: boolean;
  onReview: (documentId: string) => void;
  onAttach: (documentId: string, shipmentId: string) => void;
}) {
  const t = useT();

  return (
    <ul className="grid gap-2 sm:grid-cols-2">
      {attachments.map((attachment) => {
        const documentId = attachment.documentId ?? null;
        const refused = attachment.failureText !== "";

        return (
          <li
            key={attachment.id}
            className="border-border bg-card flex flex-col gap-2 rounded-lg border p-3"
          >
            <div className="flex min-w-0 items-start gap-2.5">
              <DocumentFileTypeIcon
                fileType={attachment.contentType}
                fileName={attachment.fileName}
                size="sm"
              />
              <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                <span className="truncate text-sm" title={attachment.fileName}>
                  {attachment.fileName}
                </span>
                <span className="text-foreground-subtle flex items-center gap-1.5 text-xs">
                  <span className="tabular-nums">{formatFileSize(attachment.byteSize)}</span>
                  <span aria-hidden>·</span>
                  <span>{attachmentKindLabel(t, attachment.kind)}</span>
                </span>
              </div>
              {refused && (
                <Badge variant="danger" appearance="outline">
                  {t("Refused")}
                </Badge>
              )}
            </div>

            {refused ? (
              <p className="text-danger text-xs">{attachment.failureText}</p>
            ) : documentId === null ? (
              <p className="text-foreground-subtle text-xs">{t("Still being read")}</p>
            ) : (
              <div className="flex flex-wrap gap-1.5">
                <Button
                  size="xs"
                  variant="outline"
                  onClick={() => void openDocument(documentId, t("The file could not be opened"))}
                >
                  <ExternalLinkIcon className="size-3" />
                  {t("Open")}
                </Button>
                <Button size="xs" variant="outline" onClick={() => onReview(documentId)}>
                  <ScanTextIcon className="size-3" />
                  {t("What the desk read")}
                </Button>
                {shipment !== null && (
                  <Button
                    size="xs"
                    variant="ghost"
                    disabled={busy}
                    onClick={() => onAttach(documentId, shipment.id)}
                  >
                    {t("Attach to {0}", shipment.proNumber)}
                  </Button>
                )}
              </div>
            )}
          </li>
        );
      })}
    </ul>
  );
}
