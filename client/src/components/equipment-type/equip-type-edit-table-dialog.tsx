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

import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { cn } from "@/lib/utils";
import { equipmentTypeSchema } from "@/lib/validations/EquipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import {
  EquipmentType,
  EquipmentTypeFormValues as FormValues,
} from "@/types/equipment";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { EquipTypeForm } from "./equip-type-table-dialog";

function EquipTypeEditForm({
  equipType,
  onOpenChange,
}: {
  equipType: EquipmentType;
  onOpenChange: (open: boolean) => void;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { handleSubmit, control } = useForm<FormValues>({
    resolver: yupResolver(equipmentTypeSchema),
    defaultValues: {
      status: equipType.status,
      name: equipType.name,
      description: equipType?.description || "",
      costPerMile: equipType?.costPerMile || "",
      equipmentClass: equipType.equipmentClass,
      exemptFromTolls: equipType.exemptFromTolls,
      fixedCost: equipType?.fixedCost || "",
      height: equipType?.height || "",
      length: equipType?.length || "",
      idlingFuelUsage: equipType?.idlingFuelUsage || "",
      weight: equipType?.weight || "",
      variableCost: equipType?.variableCost || "",
      width: equipType?.width || "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/equipment_types/${equipType.id}/`,
      successMessage: "Equipment Type updated successfully.",
      queryKeysToInvalidate: ["equipment-type-table-data"],
      closeModal: true,
      errorMessage: "Failed to update equip. type.",
    },
    () => setIsSubmitting(false),
  );

  const onSubmit = (values: FormValues) => {
    setIsSubmitting(true);
    mutation.mutate(values);
  };

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex flex-col h-full overflow-y-auto"
    >
      <EquipTypeForm control={control} />
      <SheetFooter className="mb-12">
        <Button
          type="reset"
          variant="secondary"
          onClick={() => onOpenChange(false)}
          className="w-full"
        >
          Cancel
        </Button>
        <Button
          type="submit"
          isLoading={isSubmitting}
          loadingText="Saving Changes..."
          className="w-full"
        >
          Save Changes
        </Button>
      </SheetFooter>
    </form>
  );
}

export function EquipTypeEditSheet({ onOpenChange, open }: TableSheetProps) {
  const [equipType] = useTableStore.use("currentRecord");

  if (!equipType) return null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle>{equipType && equipType.name}</SheetTitle>
          <SheetDescription>
            Last updated on {equipType && formatDate(equipType.modified)}
          </SheetDescription>
        </SheetHeader>
        {equipType && (
          <EquipTypeEditForm
            equipType={equipType}
            onOpenChange={onOpenChange}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}
