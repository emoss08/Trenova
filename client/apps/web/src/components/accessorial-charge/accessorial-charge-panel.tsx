import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { AccessorialChargeRow } from "@/lib/graphql/accessorial-charge-table";
import { apiService } from "@/services/api";
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
  initial,
  onCreated,
}: DataTablePanelProps<AccessorialChargeRow> & {
  /** What a new charge starts with, such as the code a tariff bills that the catalog lacks. */
  initial?: { code: string; description: string };
  /** Runs once the charge is created. */
  onCreated?: () => void;
}) {
  const t = useT();

  const form = useForm({
    resolver: zodResolver(accessorialChargeSchema),
    defaultValues: {
      status: "Active",
      code: initial?.code ?? "",
      description: initial?.description ?? "",
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
        queryKey="accessorial-charge-list"
        title={t("Accessorial charge")}
        fieldKey="code"
        formComponent={<AccessorialChargeForm />}
        mutationFn={(values, currentRow) =>
          apiService.accessorialChargeService.update(currentRow.id, values)
        }
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey="accessorial-charge-list"
      title={t("Accessorial charge")}
      formComponent={<AccessorialChargeForm />}
      mutationFn={(values) => apiService.accessorialChargeService.create(values)}
      onCreated={onCreated}
    />
  );
}
