import { formulaTemplate } from "@/lib/queries/formula-template";
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
import { useT } from "@trenova/shared/i18n/use-t";
import type { FormulaTemplate } from "@trenova/shared/types/formula-template";
import { useQuery } from "@tanstack/react-query";
import { UsageRows } from "../studio-top-bar";

/**
 * The pause before a material edit takes an approved or in-review template
 * back to Draft: what it does, and everything that rates with it now.
 */
export function SaveDemoteDialog({
  open,
  template,
  onCancel,
  onConfirm,
}: {
  open: boolean;
  template: FormulaTemplate | null;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  const t = useT();
  const { data: usage } = useQuery({
    ...formulaTemplate.usage(template?.id ?? ""),
    enabled: open && !!template?.id,
    staleTime: 60_000,
  });
  const version = template?.currentVersionNumber || 1;
  const inReview = template?.status === "InReview";

  return (
    <AlertDialog
      open={open}
      onOpenChange={(next) => {
        if (!next) onCancel();
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{t("Save and return to Draft?")}</AlertDialogTitle>
          <AlertDialogDescription>
            {inReview
              ? t(
                  "This changes what the reviewer is looking at, so the review in progress is cancelled and the template returns to Draft.",
                )
              : t(
                  "This changes what the template computes. v{0} keeps rating shipments until the new version is approved.",
                  version,
                )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        {usage && usage.usages.length > 0 && (
          <div className="bg-sunken rounded-md px-3 py-2">
            <UsageRows usage={usage} />
          </div>
        )}
        <AlertDialogFooter>
          <AlertDialogCancel variant="outline" size="default">
            {t("Cancel")}
          </AlertDialogCancel>
          <AlertDialogAction size="default" onClick={onConfirm}>
            {t("Save and return to Draft")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
