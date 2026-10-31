import { useT } from "@trenova/shared/i18n/use-t";
import { PdfFrame } from "@/components/elements/pdf-frame";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { BillingQueueDocumentPreviewSkeleton } from "./billing-queue-document-preview-skeleton";

/**
 * The one PDF approving the item would send, built by the same planning approval
 * runs: the invoice it would make, then each document in its packet. Nothing is
 * saved by looking.
 */
export function BillingQueuePacketPreview({
  itemId,
  open,
  onOpenChange,
}: {
  itemId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent size="xl" className="flex h-[88vh] flex-col">
        <DialogHeader>
          <DialogTitle>{t("Invoice packet")}</DialogTitle>
          <DialogDescription>
            {t("What approving sends: the invoice, then each document in its packet.")}
          </DialogDescription>
        </DialogHeader>
        {open ? (
          <PdfFrame
            endpoint={`/billing-queue/${encodeURIComponent(itemId)}/packet-preview/`}
            title={t("Invoice packet")}
            loading={<BillingQueueDocumentPreviewSkeleton />}
            className="min-h-0 flex-1"
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}
