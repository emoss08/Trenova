/**
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
import { useCustomMutation } from "@/hooks/useCustomMutation";
import { formatToUserTimezone } from "@/lib/date";
import { shipmentTypeSchema } from "@/lib/validations/ShipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
    ShipmentTypeFormValues as FormValues,
    ShipmentType,
} from "@/types/shipment";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { ShipmentTypeForm } from "./shipment-type-table-dialog";
import { Badge } from "./ui/badge";
import {
    Credenza,
    CredenzaBody,
    CredenzaClose,
    CredenzaContent,
    CredenzaDescription,
    CredenzaFooter,
    CredenzaHeader,
    CredenzaTitle,
} from "./ui/credenza";

function ShipmentTypeEditForm({
  shipmentType,
}: {
  shipmentType: ShipmentType;
}) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(shipmentTypeSchema),
    defaultValues: shipmentType,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/shipment-types/${shipmentType.id}/`,
    successMessage: "Shipment Type updated successfully.",
    queryKeysToInvalidate: "shipmentTypes",
    closeModal: true,
    reset,
    errorMessage: "Failed to update shipment type.",
  });

  const onSubmit = (values: FormValues) => mutation.mutate(values);

  return (
    <CredenzaBody>
      <form
        onSubmit={handleSubmit(onSubmit)}
        className="flex h-full flex-col overflow-y-auto"
      >
        <ShipmentTypeForm control={control} />
        <CredenzaFooter>
          <CredenzaClose asChild>
            <Button variant="outline" type="button">
              Cancel
            </Button>
          </CredenzaClose>
          <Button type="submit" isLoading={mutation.isPending}>
            Save Changes
          </Button>
        </CredenzaFooter>
      </form>
    </CredenzaBody>
  );
}

export function ShipmentTypeEditDialog({
  onOpenChange,
  open,
}: TableSheetProps) {
  const [shipmentType] = useTableStore.use("currentRecord") as ShipmentType[];

  if (!shipmentType) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span>{shipmentType.code}</span>
            <Badge className="ml-5" variant="purple">
              {shipmentType.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(shipmentType.updatedAt)}
        </CredenzaDescription>
        <ShipmentTypeEditForm shipmentType={shipmentType} />
      </CredenzaContent>
    </Credenza>
  );
}
