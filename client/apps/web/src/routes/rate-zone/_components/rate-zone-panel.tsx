import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { RateZoneRow } from "@/lib/graphql/rate-tables";
import { rateZone } from "@/lib/queries/rate-zone";
import { apiService } from "@/services/api";
import { zodResolver } from "@hookform/resolvers/zod";
import { useQuery } from "@tanstack/react-query";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { rateZoneSchema, type RateZone } from "@trenova/shared/types/rate";
import { useForm, type Resolver } from "react-hook-form";
import { RateZoneForm } from "./rate-zone-form";

export function RateZonePanel({ open, onOpenChange, mode, row }: DataTablePanelProps<RateZoneRow>) {
  const t = useT();

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

  // The table row carries no members. The form opens on the full zone instead, so
  // saving never sends an empty member list for a zone that covers places.
  const record = useQuery({
    ...rateZone.record(row?.id ?? "", row?.version),
    enabled: open && mode === "edit" && Boolean(row?.id),
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={record.data ?? null}
        form={form}
        size="lg"
        queryKey="rate-zone-list"
        title={t("Rate zone")}
        fieldKey="name"
        formComponent={<RateZoneForm />}
        mutationFn={(values, currentRow) => {
          if (!currentRow.id) {
            throw new Error("No rate zone selected");
          }

          return apiService.rateZoneService.update(currentRow.id, values);
        }}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      size="lg"
      queryKey="rate-zone-list"
      title={t("Rate zone")}
      formComponent={<RateZoneForm />}
      mutationFn={(values) => apiService.rateZoneService.create(values)}
    />
  );
}
