/*
 * COPYRIGHT(c) 2024 Trenova
 *
 * This file is part of Trenova.
 *
 * The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatDate } from "@/lib/date";
import { shipmentTypeSchema } from "@/lib/validations/ShipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import {
  ShipmentTypeFormValues as FormValues,
  ShipmentType,
} from "@/types/shipment";
import { TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
import { useForm } from "react-hook-form";
import { ShipmentTypeForm } from "./ship-type-table-dialog";

function ShipmentTypeEditForm({
  shipmentType,
}: {
  shipmentType: ShipmentType;
}) {
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);

  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(shipmentTypeSchema),
    defaultValues: {
      status: shipmentType.status,
      code: shipmentType.code,
      description: shipmentType?.description || "",
    },
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/shipment_types/${shipmentType.id}/`,
      successMessage: "Shipment Type updated successfully.",
      queryKeysToInvalidate: ["shipment-type-table-data"],
      closeModal: true,
      errorMessage: "Failed to update shipment type.",
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
      className="flex h-full flex-col overflow-y-auto"
    >
      <ShipmentTypeForm control={control} />
      <DialogFooter className="mt-6">
        <Button type="submit" isLoading={isSubmitting}>
          Save
        </Button>
      </DialogFooter>
    </form>
  );
}

export function ShipmentTypeEditDialog({
  onOpenChange,
  open,
}: TableSheetProps) {
  const [shipmentType] = useTableStore.use("currentRecord") as ShipmentType[];

  if (!shipmentType) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{shipmentType && shipmentType.code}</DialogTitle>
        </DialogHeader>
        <DialogDescription>
          Last updated on&nbsp;
          {shipmentType && formatDate(shipmentType.modified)}
        </DialogDescription>
        {shipmentType && <ShipmentTypeEditForm shipmentType={shipmentType} />}
      </DialogContent>
    </Dialog>
  );
}
