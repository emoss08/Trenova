import { FormCreateModal } from "@/components/ui/form-create-modal";
import { accessorialChargeSchema } from "@/lib/schemas/accessorial-charge-schema";

import { AccessorialChargeMethod } from "@/types/billing";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { AccessorialChargeForm } from "./accessorial-charge-form";

export function CreateAccessorialChargeModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm({
    resolver: zodResolver(accessorialChargeSchema),
    defaultValues: {
      status: Status.Active,
      code: "",
      description: "",
      unit: 1,
      method: AccessorialChargeMethod.Flat,
      amount: 1,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Accessorial Charge"
      formComponent={<AccessorialChargeForm />}
      form={form}
      url="/accessorial-charges/"
      queryKey="accessorial-charge-list"
    />
  );
}
