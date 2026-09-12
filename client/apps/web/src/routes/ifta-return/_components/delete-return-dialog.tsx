import { useT } from "@trenova/shared/i18n/use-t";
import { deleteIftaReturn } from "@/lib/graphql/ifta-return";
import { quarterLabel, type IftaPeriodKey, type IftaReturnView } from "@/lib/ifta-return";
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
import { Trash2Icon } from "lucide-react";
import { toast } from "sonner";
import { handleIftaReturnError, invalidateIftaReturn } from "./queries";

type DeleteReturnDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  ret: IftaReturnView;
  period: IftaPeriodKey;
};

export function DeleteReturnDialog({ open, onOpenChange, ret, period }: DeleteReturnDialogProps) {
  const t = useT();

  const queryClient = useQueryClient();
  const isDraft = ret.status === "Draft";

  const { mutate, isPending } = useMutation({
    mutationFn: () => deleteIftaReturn(ret.id, ret.version),
    onSuccess: async () => {
      toast.success(t("Draft deleted"), {
        description: t(
          "Nothing else changed: the miles, fuel and rates it was built from are kept.",
        ),
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
          <AlertDialogMedia className="bg-destructive/10 text-destructive">
            <Trash2Icon />
          </AlertDialogMedia>
          <AlertDialogTitle>{t("Delete the {0} draft?", quarterLabel(period))}</AlertDialogTitle>
          <AlertDialogDescription>
            <span className="block">
              {t(
                "The worksheet and every jurisdiction line on it are removed outright. The miles, fuel purchases and rates it was built from are untouched, so generating the quarter again rebuilds it from the same data.",
              )}
            </span>
            {isDraft ? null : (
              <span className="mt-2 block">
                {t("Only a draft can be deleted. This return is {0}.", ret.status.toLowerCase())}
              </span>
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isPending}>{t("Keep the draft")}</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={() => mutate()}
            disabled={isPending || !isDraft}
          >
            {isPending ? t("Deleting...") : t("Delete draft")}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
