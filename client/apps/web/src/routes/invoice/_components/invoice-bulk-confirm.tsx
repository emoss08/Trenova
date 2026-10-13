import { useT } from "@trenova/shared/i18n/use-t";
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";

export type InvoiceBulkKind = "post" | "send";

/**
 * Asked before posting or sending a selection of invoices. Both reach outside
 * the app (the ledger, the customer's inbox), so the person sees how many
 * will go and how many of the selection were left out before anything does.
 */
export function InvoiceBulkConfirm({
  kind,
  count,
  skipped,
  isPending,
  onOpenChange,
  onConfirm,
}: {
  kind: InvoiceBulkKind;
  count: number;
  skipped: number;
  isPending: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  const t = useT();

  return (
    <AlertDialog open onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {kind === "post"
              ? t("{0, plural, one {Post # invoice?} other {Post # invoices?}}", count)
              : t("{0, plural, one {Send # invoice?} other {Send # invoices?}}", count)}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {kind === "post"
              ? t(
                  "Posting writes each invoice to the ledger and delivers it as its customer's billing profile says. A posted invoice can only be corrected by an adjustment.",
                )
              : t(
                  "Each invoice is emailed to the recipients on its customer's billing profile. Invoices that already went out are not sent again.",
                )}
            {skipped > 0
              ? ` ${
                  kind === "post"
                    ? t(
                        "{0, plural, one {# selected invoice is left out because it isn't a draft.} other {# selected invoices are left out because they aren't drafts.}}",
                        skipped,
                      )
                    : t(
                        "{0, plural, one {# selected invoice is left out because it isn't posted or already went out.} other {# selected invoices are left out because they aren't posted or already went out.}}",
                        skipped,
                      )
                }`
              : null}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel variant="outline" size="default" disabled={isPending}>
            {t("Cancel")}
          </AlertDialogCancel>
          <Button
            disabled={isPending}
            isLoading={isPending}
            loadingText={kind === "post" ? t("Posting...") : t("Sending...")}
            onClick={onConfirm}
          >
            {kind === "post" ? t("Post {0}", count) : t("Send {0}", count)}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
