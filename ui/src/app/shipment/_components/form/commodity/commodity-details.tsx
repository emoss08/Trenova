import { COMMODITY_DELETE_DIALOG_KEY } from "@/constants/env";
import { ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import { useState } from "react";
import { useFieldArray, useFormContext } from "react-hook-form";
import { CommodityDeleteDialog } from "./commodity-delete-dialog";
import { CommodityDialog } from "./commodity-dialog";
import { CommodityList } from "./commodity-list";
import { CommodityListHeader } from "./commodity-list-header";

export default function ShipmentCommodityDetails({
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
    <CommodityDetailsInner
      className={cn(
        "flex flex-col gap-2 border-t border-bg-sidebar-border py-4",
        className,
      )}
    >
      <CommodityListHeader
        commodities={commodities}
        handleAddCommodity={handleAddCommodity}
      />
      <CommodityList
        commodities={commodities}
        handleEdit={handleEdit}
        handleDelete={handleDelete}
      />
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
    </CommodityDetailsInner>
  );
}

function CommodityDetailsInner({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={className}>{children}</div>;
}
