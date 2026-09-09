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
  const queryClient = useQueryClient();
  const isDraft = ret.status === "Draft";

  const { mutate, isPending } = useMutation({
    mutationFn: () => deleteIftaReturn(ret.id, ret.version),
    onSuccess: async () => {
      toast.success("Draft deleted", {
        description: "Nothing else changed: the miles, fuel and rates it was built from are kept.",
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
          <AlertDialogTitle>Delete the {quarterLabel(period)} draft?</AlertDialogTitle>
          <AlertDialogDescription>
            <span className="block">
              The worksheet and every jurisdiction line on it are removed outright. The miles, fuel
              purchases and rates it was built from are untouched, so generating the quarter again
              rebuilds it from the same data.
            </span>
            {isDraft ? null : (
              <span className="mt-2 block">
                Only a draft can be deleted. This return is {ret.status.toLowerCase()}.
              </span>
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isPending}>Keep the draft</AlertDialogCancel>
          <AlertDialogAction
            variant="destructive"
            onClick={() => mutate()}
            disabled={isPending || !isDraft}
          >
            {isPending ? "Deleting..." : "Delete draft"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
