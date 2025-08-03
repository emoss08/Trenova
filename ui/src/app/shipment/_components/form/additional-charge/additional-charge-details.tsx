/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

import { ADDITIONAL_CHARGE_DELETE_DIALOG_KEY } from "@/constants/env";
import { type ShipmentSchema } from "@/lib/schemas/shipment-schema";
import { cn } from "@/lib/utils";
import { useState } from "react";
import { useFieldArray, useFormContext } from "react-hook-form";
import { AdditionalChargeDeleteDialog } from "./additional-charge-delete-dialog";
import { AdditionalChargeDialog } from "./additional-charge-dialog";
import { AdditionalChargeList } from "./additional-charge-list";
import { AdditionalChargeListHeader } from "./additional-charge-list-header";

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
    <AdditionalChargeDetailsInner
      className={cn(
        "flex flex-col gap-2 border-t border-bg-sidebar-border py-4",
        className,
      )}
    >
      <AdditionalChargeListHeader
        additionalCharges={additionalCharges}
        handleAddAdditionalCharge={handleAddAdditionalCharge}
      />
      <AdditionalChargeList
        additionalCharges={additionalCharges}
        handleEdit={handleEdit}
        handleDelete={handleDelete}
      />
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
    </AdditionalChargeDetailsInner>
  );
}

function AdditionalChargeDetailsInner({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={className}>{children}</div>;
}
