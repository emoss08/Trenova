import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  type AccessorialChargeSchema,
  accessorialChargeSchema,
} from "@/lib/schemas/accessorial-charge-schema";

import { AccessorialChargeMethod } from "@/types/billing";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { AccessorialChargeForm } from "./accessorial-charge-form";

export function CreateAccessorialChargeModal({
  open,
  onOpenChange,
}: TableSheetProps) {
  const form = useForm<AccessorialChargeSchema>({
    resolver: yupResolver(accessorialChargeSchema),
    defaultValues: {
      status: Status.Active,
      code: "",
      description: "",
      unit: 0,
      method: AccessorialChargeMethod.Flat,
      amount: 0,
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Accessorial Charge"
      formComponent={<AccessorialChargeForm />}
      form={form}
      schema={accessorialChargeSchema}
      url="/accessorial-charges/"
      queryKey="accessorial-charge-list"
      className="max-w-[550px]"
    />
  );
}
