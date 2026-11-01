import { useT } from "@trenova/shared/i18n/use-t";
import { apiService } from "@/services/api";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { ServiceTypeRow } from "@/lib/graphql/service-type-table";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { serviceTypeSchema } from "@/types/service-type";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { ServiceTypeForm } from "./service-type-form";

export function ServiceTypePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<ServiceTypeRow>) {
  const t = useT();

  const form = useForm({
    resolver: zodResolver(serviceTypeSchema),
    defaultValues: {
      status: "Active",
      code: "",
      description: "",
      color: "",
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
        mutationFn={(values, currentRow) =>
          apiService.serviceTypeService.update(currentRow.id, values)
        }
        queryKey="service-type-list"
        title={t("Service type")}
        fieldKey="code"
        formComponent={<ServiceTypeForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      mutationFn={(values) => apiService.serviceTypeService.create(values)}
      queryKey="service-type-list"
      title={t("Service type")}
      formComponent={<ServiceTypeForm />}
    />
  );
}
