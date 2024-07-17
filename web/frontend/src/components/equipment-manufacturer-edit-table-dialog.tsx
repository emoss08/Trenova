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



import { useCustomMutation } from "@/hooks/useCustomMutation";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { formatToUserTimezone } from "@/lib/date";
import { equipManufacturerSchema } from "@/lib/validations/EquipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import {
    EquipmentManufacturer,
    EquipmentManufacturerFormValues as FormValues,
} from "@/types/equipment";
import { TableSheetProps } from "@/types/tables";
import { EquipManuForm } from "./eqiupment-manufacturer-table-dialog";
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

function EquipManuEditForm({
  equipManufacturer,
}: {
  equipManufacturer: EquipmentManufacturer;
}) {
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(equipManufacturerSchema),
    defaultValues: equipManufacturer,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/equipment-manufacturers/${equipManufacturer.id}/`,
    successMessage: "Equip. Manufacturer updated successfully.",
    queryKeysToInvalidate: "equipmentManufacturers",
    closeModal: true,
    reset,
    errorMessage: "Failed to create update equip. manufacturer.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  return (
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <EquipManuForm control={control} />
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

export function EquipMenuEditDialog({ onOpenChange, open }: TableSheetProps) {
  const [equipManufacturer] = useTableStore.use(
    "currentRecord",
  ) as EquipmentManufacturer[];

  if (!equipManufacturer) return null;

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle className="flex">
            <span>{equipManufacturer.name}</span>
            <Badge className="ml-5" variant="purple">
              {equipManufacturer.id}
            </Badge>
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on {formatToUserTimezone(equipManufacturer.updatedAt)}
        </CredenzaDescription>
        <EquipManuEditForm equipManufacturer={equipManufacturer} />
      </CredenzaContent>
    </Credenza>
  );
}
