import { useT } from "@trenova/shared/i18n/use-t";
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
            {t("{0, plural, one {Duplicate # row?} other {Duplicate # rows?}}", rowCount)}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t(
              "{0, plural, one {Are you sure you want to duplicate # row? This action cannot be undone.} other {Are you sure you want to duplicate # rows? This action cannot be undone.}}",
              rowCount,
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
