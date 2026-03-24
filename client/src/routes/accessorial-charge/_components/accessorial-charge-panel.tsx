import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import {
  accessorialChargeSchema,
  type AccessorialCharge,
} from "@/types/accessorial-charge";
import type { DataTablePanelProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { AccessorialChargeForm } from "./accessorial-charge-form";

export function AccessorialChargePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<AccessorialCharge>) {
  const form = useForm({
    resolver: zodResolver(accessorialChargeSchema),
    defaultValues: {
      status: "Active",
      code: "",
      description: "",
      method: "Flat",
      amount: undefined,
    },
    mode: "onChange",
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/accessorial-charges/"
        queryKey="accessorial-charge-list"
        title="Accessorial Charge"
        fieldKey="code"
        formComponent={<AccessorialChargeForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/accessorial-charges/"
      queryKey="accessorial-charge-list"
      title="Accessorial Charge"
      formComponent={<AccessorialChargeForm />}
    />
  );
}
