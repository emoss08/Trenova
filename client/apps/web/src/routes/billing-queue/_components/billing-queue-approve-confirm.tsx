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
import type { BillingQueueItem } from "@trenova/shared/types/billing-queue";

/**
 * Asked before a bulk approval that includes items nobody is biller of:
 * approving those makes the person approving their biller, so they choose
 * between taking them on and approving only the items that already have one.
 */
export function BillingQueueApproveConfirm({
  items,
  isPending,
  onOpenChange,
  onApprove,
}: {
  items: BillingQueueItem[];
  isPending: boolean;
  onOpenChange: (open: boolean) => void;
  onApprove: (items: BillingQueueItem[], assignMe: boolean) => void;
}) {
  const t = useT();
  const withBiller = items.filter((item) => item.assignedBillerId);
  const unassigned = items.length - withBiller.length;

  return (
    <AlertDialog open onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {t(
              "{0, plural, one {# item has no biller} other {# items have no biller}}",
              unassigned,
            )}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "Approving them makes you their biller. You can approve only the items that already have a biller instead.",
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel variant="outline" size="default">
            {t("Back")}
          </AlertDialogCancel>
          {withBiller.length > 0 && (
            <Button
              variant="outline"
              disabled={isPending}
              onClick={() => onApprove(withBiller, false)}
            >
              {t("Only the {0} with a biller", withBiller.length)}
            </Button>
          )}
          <Button disabled={isPending} isLoading={isPending} onClick={() => onApprove(items, true)}>
            {t("Assign me and approve {0}", items.length)}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
