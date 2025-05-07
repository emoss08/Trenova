import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Icon } from "@/components/ui/icons";
import { STOP_DIALOG_NOTICE_KEY } from "@/constants/env";
import { type TableSheetProps } from "@/types/data-table";
import { faInfoCircle, faXmark } from "@fortawesome/pro-solid-svg-icons";
import { useLocalStorage } from "@uidotdev/usehooks";
import { memo, useCallback } from "react";
import { StopDialogForm } from "./stop-dialog-form";

type StopDialogProps = TableSheetProps & {
  moveIdx: number;
  stopIdx: number;
};

export function StopDialog({
  open,
  onOpenChange,
  moveIdx,
  stopIdx,
}: StopDialogProps) {
  const isEditing = stopIdx !== undefined;

  // * If isEditing is true, fetch the stop data from the server.
  // * Otherwise, populate form with default stop information.

  console.info("stop-dialog debug information", {
    open,
    moveIdx,
    stopIdx,
    isEditing,
  });

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEditing ? "Edit Stop" : "Add Stop"}</DialogTitle>
          <DialogDescription>
            {isEditing
              ? "Edit the stop details for this shipment."
              : "Add a new stop to the shipment."}
          </DialogDescription>
        </DialogHeader>
        <StopDialogNotice />
        <DialogBody>
          <StopDialogForm moveIdx={moveIdx} stopIdx={stopIdx} />
        </DialogBody>
        {/* <DialogFooter>
          <Button type="button" variant="outline" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="button" onClick={handleSave}>
            {isEditing ? "Update" : "Add"}
          </Button>
        </DialogFooter> */}
      </DialogContent>
    </Dialog>
  );
}

const StopDialogNotice = memo(function StopDialogNotice() {
  const [noticeVisible, setNoticeVisible] = useLocalStorage(
    STOP_DIALOG_NOTICE_KEY,
    true,
  );

  const handleClose = useCallback(() => {
    setNoticeVisible(false);
  }, [setNoticeVisible]);

  if (!noticeVisible) return null;

  return (
    <div className="bg-blue-500/20 px-4 py-3 text-blue-500">
      <div className="flex gap-2">
        <div className="flex grow gap-3">
          <Icon
            icon={faInfoCircle}
            className="mt-0.5 shrink-0 text-blue-500"
            aria-hidden="true"
          />
          <div className="flex grow flex-col justify-between gap-2 md:flex-row">
            <span className="text-sm">
              All times are displayed in your local timezone. Please ensure
              location details are accurate for proper routing.
            </span>
          </div>
        </div>
        <Button
          variant="ghost"
          className="group -my-1.5 -me-2 size-8 shrink-0 p-0 hover:bg-transparent"
          onClick={handleClose}
          aria-label="Close banner"
        >
          <Icon
            icon={faXmark}
            className="opacity-60 transition-opacity group-hover:opacity-100"
            aria-hidden="true"
          />
        </Button>
      </div>
    </div>
  );
});
