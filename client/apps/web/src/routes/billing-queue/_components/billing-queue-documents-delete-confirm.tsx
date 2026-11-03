import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Button } from "@trenova/shared/components/ui/button";
import { useT } from "@trenova/shared/i18n/use-t";
import { useState } from "react";

/**
 * Asks before documents are deleted, one or several, since a delete cannot be
 * taken back from the board. The delete holds its button until the server answers;
 * the dialog closes once it has.
 */
export function DocumentsDeleteConfirm({
  count,
  open,
  onOpenChange,
  onConfirm,
}: {
  count: number;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => Promise<void>;
}) {
  const t = useT();
  const [deleting, setDeleting] = useState(false);

  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (!deleting) onOpenChange(next);
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {t("Delete {0, plural, one {# document} other {# documents}}?", count)}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "{0, plural, one {It is removed from the shipment, and from the invoice packet if it is in it.} other {They are removed from the shipment, and from the invoice packet if they are in it.}}",
              count,
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <Button
            type="button"
            variant="outline"
            disabled={deleting}
            onClick={() => onOpenChange(false)}
          >
            {t("Cancel")}
          </Button>
          <Button
            type="button"
            variant="destructive"
            isLoading={deleting}
            loadingText={t("Deleting...")}
            onClick={() => {
              setDeleting(true);
              void onConfirm()
                .then(() => onOpenChange(false))
                .catch(() => {
                  // The delete has already reported why it failed.
                })
                .finally(() => setDeleting(false));
            }}
          >
            {t("Delete")}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
