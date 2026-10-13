import { useT } from "@trenova/shared/i18n/use-t";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@trenova/shared/components/ui/table";
import { usePostInvoice } from "@/hooks/use-post-invoice";
import { formatMinor } from "@trenova/shared/lib/benefits";
import { formatUnixDateMedium } from "@trenova/shared/lib/date";
import { apiService } from "@/services/api";
import type { InvoicePostPreview } from "@trenova/shared/types/invoice";
import { useQuery } from "@tanstack/react-query";

export const invoicePostPreviewQueryKey = (invoiceId: string) =>
  ["invoice", invoiceId, "post-preview"] as const;

/**
 * Asked before an invoice is posted. The server plans the post with the code
 * that will run it, so the person sees the ledger entry it books and what else
 * it moves, or why it would be refused, before anything is written.
 */
export function InvoicePostDialog({
  invoiceId,
  open,
  onOpenChange,
}: {
  invoiceId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const t = useT();
  const { mutate: post, isPending } = usePostInvoice();

  const {
    data: preview,
    isLoading,
    isError,
    refetch,
  } = useQuery({
    queryKey: invoicePostPreviewQueryKey(invoiceId),
    queryFn: () => apiService.invoiceService.getPostPreview(invoiceId),
    enabled: open,
    staleTime: 0,
    gcTime: 0,
  });

  const refused = (preview?.refusals.length ?? 0) > 0;
  const canPost = !!preview && !refused && !preview.alreadyPosted;

  return (
    <Dialog open={open} onOpenChange={(next) => !isPending && onOpenChange(next)}>
      <DialogContent size="lg">
        <DialogHeader>
          <DialogTitle>{preview ? t("Post {0}?", preview.number) : t("Post invoice?")}</DialogTitle>
          <DialogDescription>
            {preview
              ? t(
                  "{0} to {1}. Once posted, it can only be corrected by an adjustment.",
                  formatMinor(preview.totalMinor, preview.currencyCode),
                  preview.billToName,
                )
              : t("Checking what posting this invoice would do...")}
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <div className="flex flex-col gap-2" aria-busy="true">
            <Skeleton className="h-7 w-full" />
            <Skeleton className="h-7 w-full" />
            <Skeleton className="h-7 w-2/3" />
          </div>
        ) : isError ? (
          <Alert size="sm" variant="destructive">
            <AlertTitle>{t("The post could not be checked")}</AlertTitle>
            <AlertDescription>
              <Button variant="link" size="xs" className="h-auto p-0" onClick={() => refetch()}>
                {t("Try again")}
              </Button>
            </AlertDescription>
          </Alert>
        ) : preview ? (
          <PostPreviewBody preview={preview} />
        ) : null}

        <DialogFooter>
          <Button variant="outline" disabled={isPending} onClick={() => onOpenChange(false)}>
            {t("Cancel")}
          </Button>
          <Button
            disabled={!canPost || isPending}
            isLoading={isPending}
            loadingText={t("Posting...")}
            onClick={() => post(invoiceId, { onSuccess: () => onOpenChange(false) })}
          >
            {t("Post invoice")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function PostPreviewBody({ preview }: { preview: InvoicePostPreview }) {
  const t = useT();

  if (preview.refusals.length > 0) {
    return (
      <Alert size="sm" variant="destructive">
        <AlertTitle>{t("This invoice can't be posted yet")}</AlertTitle>
        <AlertDescription>
          <ul className="list-disc pl-4">
            {preview.refusals.map((reason) => (
              <li key={reason}>{reason}</li>
            ))}
          </ul>
        </AlertDescription>
      </Alert>
    );
  }

  if (preview.alreadyPosted) {
    return (
      <Alert size="sm" variant="info">
        <AlertDescription>{t("This invoice is already posted.")}</AlertDescription>
      </Alert>
    );
  }

  const effects = postEffects(preview, t);

  return (
    <div className="flex flex-col gap-4">
      <section className="flex flex-col gap-2">
        <h3 className="text-sm font-medium">{t("Ledger entry")}</h3>
        {preview.journal ? (
          <>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("Account")}</TableHead>
                  <TableHead className="text-right">{t("Debit")}</TableHead>
                  <TableHead className="text-right">{t("Credit")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {preview.journal.lines.map((line, index) => (
                  <TableRow key={`${line.glAccountId}-${index}`}>
                    <TableCell>
                      <div className="flex flex-col">
                        <span>
                          {line.accountCode
                            ? `${line.accountCode} · ${line.accountName}`
                            : line.glAccountId}
                        </span>
                        {line.description ? (
                          <span className="text-muted-foreground text-xs">{line.description}</span>
                        ) : null}
                      </div>
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {line.debitMinor > 0
                        ? formatMinor(line.debitMinor, preview.currencyCode)
                        : ""}
                    </TableCell>
                    <TableCell className="text-right tabular-nums">
                      {line.creditMinor > 0
                        ? formatMinor(line.creditMinor, preview.currencyCode)
                        : ""}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <p className="text-muted-foreground text-xs">
              {preview.journal.requiresApproval
                ? t(
                    "Booked on {0} and held for approval before it reaches the ledger.",
                    formatUnixDateMedium(preview.journal.accountingDate),
                  )
                : t("Booked on {0}.", formatUnixDateMedium(preview.journal.accountingDate))}
            </p>
          </>
        ) : (
          <p className="text-muted-foreground text-xs">
            {t("Your accounting settings book no ledger entry when an invoice posts.")}
          </p>
        )}
      </section>

      {effects.length > 0 ? (
        <section className="flex flex-col gap-2">
          <h3 className="text-sm font-medium">{t("Posting also")}</h3>
          <ul className="text-muted-foreground flex list-disc flex-col gap-1 pl-4 text-sm">
            {effects.map((effect) => (
              <li key={effect}>{effect}</li>
            ))}
          </ul>
        </section>
      ) : null}

      {preview.edi && preview.edi.blockers.length > 0 ? (
        <Alert size="sm" variant="warning">
          <AlertTitle>{t("The 210 to {0} won't go", preview.edi.partnerName)}</AlertTitle>
          <AlertDescription>{preview.edi.blockers.join("; ")}</AlertDescription>
        </Alert>
      ) : null}
    </div>
  );
}

function postEffects(preview: InvoicePostPreview, t: ReturnType<typeof useT>): string[] {
  const effects: string[] = [];

  if (preview.shipmentsBilled.length > 0) {
    effects.push(
      preview.shipmentsBilled.length === 1
        ? t("Marks shipment {0} invoiced", preview.shipmentsBilled[0])
        : t("Marks {0} shipments invoiced", preview.shipmentsBilled.length),
    );
  }
  if (preview.queueItemPosted) {
    effects.push(t("Closes its billing queue item as posted"));
  }
  for (const sync of preview.accountingSync) {
    effects.push(
      sync.awaitsRelease
        ? t(
            "Queues it for {0} ({1}), where it waits to be released",
            sync.integration,
            sync.company,
          )
        : t("Syncs it to {0} ({1})", sync.integration, sync.company),
    );
  }
  if (preview.edi?.willSend) {
    effects.push(t("Sends an EDI 210 to {0}", preview.edi.partnerName));
  }

  return effects;
}
