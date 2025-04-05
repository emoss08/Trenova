import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { ADDITIONAL_CHARGE_DELETE_DIALOG_KEY } from "@/constants/env";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import { AdditionalCharge } from "@/types/shipment";
import { faPlus } from "@fortawesome/pro-regular-svg-icons";
import { memo, useState } from "react";
import { useFieldArray, useFormContext } from "react-hook-form";
import { AdditionalChargeDeleteDialog } from "./additional-charge-delete-dialog";
import { AdditionalChargeList } from "./additional-charge-list";
import { AdditionalChargeDialog } from "./additional-charge-dialog";

export default function AdditionalChargeDetails({
  className,
}: {
  className?: string;
}) {
  const [additionalChargeDialogOpen, setAdditionalChargeDialogOpen] =
    useState<boolean>(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState<boolean>(false);
  const [deletingIndex, setDeletingIndex] = useState<number | null>(null);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);

  const { control } = useFormContext<ShipmentSchema>();
  const {
    fields: additionalCharges,
    update,
    remove,
  } = useFieldArray({
    control,
    name: "additionalCharges",
  });

  const handleAddAdditionalCharge = () => {
    setAdditionalChargeDialogOpen(true);
  };

  const handleEdit = (index: number) => {
    setEditingIndex(index);
    setAdditionalChargeDialogOpen(true);
  };

  const handleDelete = (index: number) => {
    const showDialog =
      localStorage.getItem(ADDITIONAL_CHARGE_DELETE_DIALOG_KEY) !== "false";

    if (showDialog) {
      setDeletingIndex(index);
      setDeleteDialogOpen(true);
    } else {
      remove(index);
    }
  };

  const handleConfirmDelete = (doNotShowAgain: boolean) => {
    if (deletingIndex !== null) {
      remove(deletingIndex);

      if (doNotShowAgain) {
        localStorage.setItem(ADDITIONAL_CHARGE_DELETE_DIALOG_KEY, "false");
      }

      setDeleteDialogOpen(false);
      setDeletingIndex(null);
    }
  };

  const handleDialogClose = () => {
    setAdditionalChargeDialogOpen(false);
    setEditingIndex(null);
  };

  return (
    <>
      <div className={cn("flex flex-col gap-2 py-4", className)}>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1">
            <h3 className="text-sm font-medium">Additional Charges</h3>
            <span className="text-2xs text-muted-foreground">
              ({additionalCharges?.length ?? 0})
            </span>
          </div>
          <AddAdditionalChargeButton onClick={handleAddAdditionalCharge} />
        </div>
        <AdditionalChargeList
          additionalCharges={additionalCharges as AdditionalCharge[]}
          handleEdit={handleEdit}
          handleDelete={handleDelete}
        />
      </div>
      {additionalChargeDialogOpen && (
        <AdditionalChargeDialog
          open={additionalChargeDialogOpen}
          onOpenChange={handleDialogClose}
          isEditing={editingIndex !== null}
          update={update}
          index={editingIndex ?? additionalCharges.length}
          remove={remove}
        />
      )}
      {deleteDialogOpen && (
        <AdditionalChargeDeleteDialog
          open={deleteDialogOpen}
          onOpenChange={(open) => {
            setDeleteDialogOpen(open);
            if (!open) {
              setDeletingIndex(null);
            }
          }}
          handleDelete={handleConfirmDelete}
        />
      )}
    </>
  );
}

const AddAdditionalChargeButton = memo(function AddAdditionalChargeButton({
  onClick,
}: {
  onClick: () => void;
}) {
  return (
    <Button type="button" variant="outline" size="xs" onClick={onClick}>
      <Icon icon={faPlus} className="size-4" />
      Add Additional Charge
    </Button>
  );
});

AddAdditionalChargeButton.displayName = "AddAdditionalChargeButton";
