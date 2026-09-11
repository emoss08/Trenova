import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { AccessorialChargeRow } from "@/lib/graphql/accessorial-charge-table";
import { accessorialChargeSchema } from "@trenova/shared/types/accessorial-charge";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { AccessorialChargeForm } from "./accessorial-charge-form";

export function AccessorialChargePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<AccessorialChargeRow>) {
  const t = useT();

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
        title={t("Accessorial Charge")}
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
      title={t("Accessorial Charge")}
      formComponent={<AccessorialChargeForm />}
    />
  );
}
