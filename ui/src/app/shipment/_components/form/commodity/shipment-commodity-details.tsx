import { Button } from "@/components/ui/button";
import { Icon } from "@/components/ui/icons";
import { COMMODITY_DELETE_DIALOG_KEY } from "@/constants/env";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import { ShipmentCommodity } from "@/types/shipment";
import { faPlus } from "@fortawesome/pro-solid-svg-icons";
import { useState } from "react";
import { useFieldArray, useFormContext } from "react-hook-form";
import { CommodityDeleteDialog } from "./commodity-delete-dialog";
import { CommodityDialog } from "./commodity-dialog";
import { CommodityList } from "./commodity-list";

export function ShipmentCommodityDetails({
  className,
}: {
  className?: string;
}) {
  const [commodityDialogOpen, setCommodityDialogOpen] =
    useState<boolean>(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState<boolean>(false);
  const [deletingIndex, setDeletingIndex] = useState<number | null>(null);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);

  const { control } = useFormContext<ShipmentSchema>();
  const {
    fields: commodities,
    update,
    remove,
  } = useFieldArray({
    control,
    name: "commodities",
  });

  const handleAddCommodity = () => {
    setCommodityDialogOpen(true);
  };

  const handleEdit = (index: number) => {
    setEditingIndex(index);
    setCommodityDialogOpen(true);
  };

  const handleDelete = (index: number) => {
    // Always check localStorage directly
    const showDialog =
      localStorage.getItem(COMMODITY_DELETE_DIALOG_KEY) !== "false";

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
        localStorage.setItem(COMMODITY_DELETE_DIALOG_KEY, "false");
      }

      setDeleteDialogOpen(false);
      setDeletingIndex(null);
    }
  };

  const handleDialogClose = () => {
    setCommodityDialogOpen(false);
    setEditingIndex(null);
  };

  return (
    <>
      <div
        className={cn(
          "flex flex-col gap-2 border-t border-bg-sidebar-border py-4",
          className,
        )}
      >
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-1">
            <h3 className="text-sm font-medium">Commodities</h3>
            <span className="text-2xs text-muted-foreground">
              ({commodities?.length ?? 0})
            </span>
          </div>
          <Button
            type="button"
            variant="outline"
            size="xs"
            onClick={handleAddCommodity}
          >
            <Icon icon={faPlus} className="size-4" />
            Add Commodity
          </Button>
        </div>
        <CommodityList
          commodities={commodities as ShipmentCommodity[]}
          handleEdit={handleEdit}
          handleDelete={handleDelete}
        />
      </div>
      {commodityDialogOpen && (
        <CommodityDialog
          open={commodityDialogOpen}
          onOpenChange={handleDialogClose}
          isEditing={editingIndex !== null}
          update={update}
          index={editingIndex ?? commodities.length}
          remove={remove}
        />
      )}
      {deleteDialogOpen && (
        <CommodityDeleteDialog
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
