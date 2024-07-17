/**
 * Copyright (c) 2024 Trenova Technologies, LLC
 *
 * Licensed under the Business Source License 1.1 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://trenova.app/pricing/
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *
 * Key Terms:
 * - Non-production use only
 * - Change Date: 2026-11-16
 * - Change License: GNU General Public License v2 or later
 *
 * For full license text, see the LICENSE file in the root directory.
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
