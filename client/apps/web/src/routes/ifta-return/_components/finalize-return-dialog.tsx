import { useT } from "@trenova/shared/i18n/use-t";
import { finalizeIftaReturn } from "@/lib/graphql/ifta-return";
import {
  linesMissingRates,
  quarterLabel,
  type IftaPeriodKey,
  type IftaReturnView,
} from "@/lib/ifta-return";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogMedia,
  AlertDialogTitle,
} from "@trenova/shared/components/ui/alert-dialog";
import { Badge } from "@trenova/shared/components/ui/badge";
import { pluralize } from "@trenova/shared/lib/utils";
import { IFTA_FUEL_TYPE_LABELS } from "@trenova/shared/types/fuel-ifta-enums";
import { LockIcon } from "lucide-react";
import { toast } from "sonner";
import { handleIftaReturnError, invalidateIftaReturn } from "./queries";

type FinalizeReturnDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  ret: IftaReturnView;
  period: IftaPeriodKey;
};

export function FinalizeReturnDialog({
  open,
  onOpenChange,
  ret,
  period,
}: FinalizeReturnDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const missing = linesMissingRates(ret.lines);

  const { mutate, isPending } = useMutation({
    mutationFn: () => finalizeIftaReturn(ret.id, ret.version),
    onSuccess: async () => {
      toast.success(t("Return finalized"), {
        description: t("The worksheet is locked. Reopening it takes a reason, which is audited."),
      });
      await invalidateIftaReturn(queryClient, period);
      onOpenChange(false);
    },
    onError: (error) => handleIftaReturnError(error, queryClient, period),
  });

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogMedia className="bg-info/10 text-info">
            <LockIcon />
          </AlertDialogMedia>
          <AlertDialogTitle>{t("Finalize the {0} return?", quarterLabel(period))}</AlertDialogTitle>
          <AlertDialogDescription>
            <span className="block">
              {t(
                "The return is recomputed one last time and then locked: its figures stop moving with the miles, fuel and rates on file, so it is the record you file from. Reopening it afterwards takes a reason, which is kept with the return.",
              )}
            </span>
            {missing.length > 0 ? (
              <span className="mt-2 block">
                {t(
                  "{0} member {1} has no published rate, so finalizing is refused. Publish the missing rates, recompute, then finalize.",
                  missing.length,
                  pluralize("line", missing.length),
                )}
              </span>
            ) : null}
          </AlertDialogDescription>
        </AlertDialogHeader>
        {missing.length > 0 ? (
          <ul className="mt-3 flex max-h-40 flex-col gap-1 overflow-y-auto rounded-md border p-2">
            {missing.map((line) => (
              <li key={line.id} className="flex items-center gap-2 text-xs">
                <Badge variant="inactive">{t("No rate")}</Badge>
                <span className="font-medium">{line.jurisdiction.code}</span>
                <span className="text-muted-foreground">{line.jurisdiction.name}</span>
                <span className="text-muted-foreground">
                  · {IFTA_FUEL_TYPE_LABELS[line.fuelType]}
                </span>
              </li>
            ))}
          </ul>
        ) : null}
        <AlertDialogFooter className="mt-4">
          <AlertDialogCancel disabled={isPending}>{t("Keep it a draft")}</AlertDialogCancel>
          <AlertDialogAction onClick={() => mutate()} disabled={isPending || missing.length > 0}>
            {isPending ? t("Finalizing...") : t("Finalize return")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
