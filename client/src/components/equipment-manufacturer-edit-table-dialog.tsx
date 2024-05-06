import { useCustomMutation } from "@/hooks/useCustomMutation";
import { yupResolver } from "@hookform/resolvers/yup";
import React from "react";
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
  const [isSubmitting, setIsSubmitting] = React.useState<boolean>(false);
  const { control, reset, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(equipManufacturerSchema),
    defaultValues: equipManufacturer,
  });

  const mutation = useCustomMutation<FormValues>(
    control,
    {
      method: "PUT",
      path: `/equipment-manufacturers/${equipManufacturer.id}/`,
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
    <CredenzaBody>
      <form onSubmit={handleSubmit(onSubmit)}>
        <EquipManuForm control={control} />
        <CredenzaFooter>
          <CredenzaClose asChild>
            <Button variant="outline" type="button">
              Cancel
            </Button>
          </CredenzaClose>
          <Button type="submit" isLoading={isSubmitting}>
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

  return (
    <Credenza open={open} onOpenChange={onOpenChange}>
      <CredenzaContent>
        <CredenzaHeader>
          <CredenzaTitle>
            {equipManufacturer && equipManufacturer.name}
          </CredenzaTitle>
        </CredenzaHeader>
        <CredenzaDescription>
          Last updated on{" "}
          {equipManufacturer &&
            formatToUserTimezone(equipManufacturer.updatedAt)}
        </CredenzaDescription>
        {equipManufacturer && (
          <EquipManuEditForm equipManufacturer={equipManufacturer} />
        )}
      </CredenzaContent>
    </Credenza>
  );
}
