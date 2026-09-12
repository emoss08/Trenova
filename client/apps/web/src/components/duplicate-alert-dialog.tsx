import { useT } from "@trenova/shared/i18n/use-t";
import { pluralize } from "@trenova/shared/lib/utils";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";

export function DuplicateAlertDialog({
  open,
  onOpenChange,
  rowCount,
  onConfirm,
  isLoading,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  rowCount: number;
  onConfirm: () => void;
  isLoading: boolean;
}) {
  const t = useT();

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle className="text-lg font-semibold">
            {t("Duplicate {0} {1}?", rowCount, pluralize("row", rowCount))}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "Are you sure you want to duplicate {0} {1}? This action cannot be undone.",
              rowCount,
              pluralize("row", rowCount),
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel variant="outline" size="default">
            {t("Cancel")}
          </AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            size="default"
            onClick={onConfirm}
            disabled={isLoading}
            isLoading={isLoading}
          >
            {t("Duplicate")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
