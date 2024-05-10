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
  const { control, handleSubmit } = useForm<FormValues>({
    resolver: yupResolver(equipManufacturerSchema),
    defaultValues: equipManufacturer,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/equipment-manufacturers/${equipManufacturer.id}/`,
    successMessage: "Equip. Manufacturer updated successfully.",
    queryKeysToInvalidate: ["equipment-manufacturer-table-data"],
    closeModal: true,
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
