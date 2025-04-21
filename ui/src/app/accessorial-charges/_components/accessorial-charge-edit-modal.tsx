import { FormEditModal } from "@/components/ui/form-edit-model";
import {
  accessorialChargeSchema,
  type AccessorialChargeSchema,
} from "@/lib/schemas/accessorial-charge-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { AccessorialChargeForm } from "./accessorial-charge-form";

export function EditAccessorialChargeModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<AccessorialChargeSchema>) {
  const form = useForm({
    resolver: zodResolver(accessorialChargeSchema),
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
      form={form}
    />
  );
}
