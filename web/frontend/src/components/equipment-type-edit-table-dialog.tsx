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
import { formatToUserTimezone } from "@/lib/date";
import { cn } from "@/lib/utils";
import { equipmentTypeSchema } from "@/lib/validations/EquipmentSchema";
import { useTableStore } from "@/stores/TableStore";
import type {
  EquipmentType,
  EquipmentTypeFormValues as FormValues,
} from "@/types/equipment";
import { type TableSheetProps } from "@/types/tables";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { EquipTypeForm } from "./equipment-type-table-dialog";
import { Badge } from "./ui/badge";

function EquipTypeEditForm({
  equipType,
  onOpenChange,
}: {
  equipType: EquipmentType;
  onOpenChange: (open: boolean) => void;
}) {
  const { handleSubmit, reset, control } = useForm<FormValues>({
    resolver: yupResolver(equipmentTypeSchema),
    defaultValues: equipType,
  });

  const mutation = useCustomMutation<FormValues>(control, {
    method: "PUT",
    path: `/equipment-types/${equipType.id}/`,
    successMessage: "Equipment Type updated successfully.",
    queryKeysToInvalidate: "equipmentTypes",
    closeModal: true,
    reset,
    errorMessage: "Failed to update equip. type.",
  });

  const onSubmit = (values: FormValues) => {
    mutation.mutate(values);
  };

  return (
    <form
      onSubmit={handleSubmit(onSubmit)}
      className="flex h-full flex-col overflow-y-auto"
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
        <Button type="submit" isLoading={mutation.isPending} className="w-full">
          Save Changes
        </Button>
      </SheetFooter>
    </form>
  );
}

export function EquipTypeEditSheet({ onOpenChange, open }: TableSheetProps) {
  const [equipType] = useTableStore.use("currentRecord") as EquipmentType[];

  if (!equipType) return null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className={cn("w-full xl:w-1/2")}>
        <SheetHeader>
          <SheetTitle className="flex">
            <span>{equipType.code}</span>
            <Badge className="ml-5" variant="purple">
              {equipType.id}
            </Badge>
          </SheetTitle>
          <SheetDescription>
            Last updated on {formatToUserTimezone(equipType.updatedAt)}
          </SheetDescription>
        </SheetHeader>
        <EquipTypeEditForm equipType={equipType} onOpenChange={onOpenChange} />
      </SheetContent>
    </Sheet>
  );
}
