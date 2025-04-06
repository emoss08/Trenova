import { FormEditModal } from "@/components/ui/form-edit-model";
import {
  accessorialChargeSchema,
  type AccessorialChargeSchema,
} from "@/lib/schemas/accessorial-charge-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { AccessorialChargeForm } from "./accessorial-charge-form";

export function EditAccessorialChargeModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<AccessorialChargeSchema>) {
  const form = useForm<AccessorialChargeSchema>({
    resolver: yupResolver(accessorialChargeSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      url="/accessorial-charges/"
      title="Accessorial Charge"
      queryKey="accessorial-charge-list"
      formComponent={<AccessorialChargeForm />}
      fieldKey="code"
      className="max-w-[500px]"
      form={form}
      schema={accessorialChargeSchema}
    />
  );
}
