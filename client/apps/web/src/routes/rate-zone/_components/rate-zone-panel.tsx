import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { zodResolver } from "@hookform/resolvers/zod";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { rateZoneSchema, type RateZone } from "@trenova/shared/types/rate";
import { useForm, type Resolver } from "react-hook-form";
import { RateZoneForm } from "./rate-zone-form";

export function RateZonePanel({ open, onOpenChange, mode, row }: DataTablePanelProps<RateZone>) {
  const form = useForm<RateZone>({
    resolver: zodResolver(rateZoneSchema) as Resolver<RateZone>,
    defaultValues: {
      status: "Active",
      kind: "Custom",
      code: "",
      name: "",
      description: "",
      members: [],
    } as RateZone,
    mode: "onChange",
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        size="lg"
        url="/rate-zones/"
        queryKey="rate-zone-list"
        title="Rate Zone"
        fieldKey="name"
        formComponent={<RateZoneForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      size="lg"
      url="/rate-zones/"
      queryKey="rate-zone-list"
      title="Rate Zone"
      formComponent={<RateZoneForm />}
    />
  );
}
