import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import { serviceTypeSchema, type ServiceType } from "@/types/service-type";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { ServiceTypeForm } from "./service-type-form";

export function ServiceTypePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<ServiceType>) {
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
        url="/service-types/"
        queryKey="service-type-list"
        title="Service Type"
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
      url="/service-types/"
      queryKey="service-type-list"
      title="Service Type"
      formComponent={<ServiceTypeForm />}
    />
  );
}
