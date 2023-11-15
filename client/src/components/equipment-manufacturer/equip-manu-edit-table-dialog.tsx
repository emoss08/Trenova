/*
 * COPYRIGHT(c) 2023 MONTA
 *
 * This file is part of Monta.
 *
 * The Monta software is licensed under the Business Source License 1.1. You are granted the right
 * to copy, modify, and redistribute the software, but only for non-production use or with a total
 * of less than three server instances. Starting from the Change Date (November 16, 2026), the
 * software will be made available under version 2 or later of the GNU General Public License.
 * If you use the software in violation of this license, your rights under the license will be
 * terminated automatically. The software is provided "as is," and the Licensor disclaims all
 * warranties and conditions. If you use this license's text or the "Business Source License" name
 * and trademark, you must comply with the Licensor's covenants, which include specifying the
 * Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
 * Grant, and not modifying the license in any other way.
 */

import React from "react";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { useCustomMutation } from "@/hooks/useCustomMutation";

import { TableSheetProps } from "@/types/tables";
import { useTableStore } from "@/stores/TableStore";
import { formatDate } from "@/lib/date";
import { toast } from "@/components/ui/use-toast";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { equipManufacturerSchema } from "@/lib/validations/EquipmentSchema";
import {
  EquipmentManufacturer,
  EquipmentManufacturerFormValues as FormValues,
} from "@/types/equipment";
import { EquipManuForm } from "@/components/equipment-manufacturer/eqiup-manu-table-dialog";

function EquipManuEditForm({
  equipManufacturer,
}: {
  equipManufacturer: EquipmentManufacturer;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(equipManufacturerSchema),
    defaultValues: {
      status: equipManufacturer.status,
      name: equipManufacturer.name,
      description: equipManufacturer.description,
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    toast,
    {
      method: "PUT",
      path: `/equipment_manufacturers/${equipManufacturer.id}/`,
      successMessage: "Equip. Manufacturer updated successfully.",
      queryKeysToInvalidate: ["equipment-manufacturer-table-data"],
      closeModal: true,
      errorMessage: "Failed to create update equip. manufacturer.",
    },
    () => setIsSubmitting(false),
    reset,
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)}>
      <EquipManuForm control={control} />
      <DialogFooter className="mt-6">
        <Button
          type="submit"
          isLoading={isSubmitting}
          loadingText="Saving Changes..."
        >
          Save
        </Button>
      </DialogFooter>
    </form>
  );
}

export function EquipMenuEditDialog({ onOpenChange, open }: TableSheetProps) {
  const [equipManufacturer] = useTableStore.use("currentRecord");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {equipManufacturer && equipManufacturer.name}
          </DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on{" "}
          {equipManufacturer && formatDate(equipManufacturer.modified)}
        </DialogDescription>
        {equipManufacturer && (
          <EquipManuEditForm equipManufacturer={equipManufacturer} />
        )}
      </DialogContent>
    </Dialog>
  );
}
