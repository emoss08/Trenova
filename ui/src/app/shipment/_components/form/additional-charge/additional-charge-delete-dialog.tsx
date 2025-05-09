import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Checkbox } from "@/components/ui/checkbox";
import { TableSheetProps } from "@/types/data-table";
import { useState } from "react";

interface AdditionalChargeDeleteDialogProps extends TableSheetProps {
  handleDelete: (doNotShowAgain: boolean) => void;
}

export function AdditionalChargeDeleteDialog({
  open,
  onOpenChange,
  handleDelete,
}: AdditionalChargeDeleteDialogProps) {
  const [doNotShowAgain, setDoNotShowAgain] = useState(false);

  const handleDeleteClick = () => {
    handleDelete(doNotShowAgain);
  };

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent className="sm:max-w-md p-3">
        <AlertDialogHeader>
          <AlertDialogTitle>Are you sure?</AlertDialogTitle>
          <AlertDialogDescription>
            Once you delete this additional charge, there is no way to recover
            it. Please make sure you want to proceed with this action.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter className="flex flex-col sm:justify-between gap-4 p-0">
          <div className="flex items-center space-x-2">
            <Checkbox
              className="size-4"
              id="doNotShowAgain"
              checked={doNotShowAgain}
              onCheckedChange={(checked) =>
                setDoNotShowAgain(checked as boolean)
              }
            />
            <label
              htmlFor="doNotShowAgain"
              className="select-none text-sm font-medium mt-1 leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70"
            >
              Do not show this message again
            </label>
          </div>
          <div className="flex justify-end gap-2">
            <AlertDialogCancel onClick={() => onOpenChange(false)}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction onClick={handleDeleteClick}>
              Yes I&apos;m sure
            </AlertDialogAction>
          </div>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
