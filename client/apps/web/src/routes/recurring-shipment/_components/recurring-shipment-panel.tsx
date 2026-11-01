import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { RecurringShipmentRow } from "@/lib/graphql/recurring-shipment-table";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { recurringShipmentSchema, type RecurringShipment } from "@/types/recurring-shipment";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm, type DefaultValues, type Resolver } from "react-hook-form";
import { apiService } from "@/services/api";
import { RecurringShipmentForm } from "./recurring-shipment-form";

const defaultValues: Partial<RecurringShipment> = {
  name: "",
  description: "",
  sourceShipmentId: "",
  status: "Active",
  cronExpression: "0 8 * * 1",
  timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
  startDate: null,
  endDate: null,
  maxOccurrences: null,
  leadTimeDays: 1,
  skipWeekends: false,
  exceptionPolicy: "Skip",
  blackoutDates: [],
  autoGenerate: true,
};

export function RecurringShipmentPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<RecurringShipmentRow>) {
  const t = useT();

  const form = useForm<RecurringShipment>({
    resolver: zodResolver(recurringShipmentSchema) as Resolver<RecurringShipment>,
    defaultValues: defaultValues as DefaultValues<RecurringShipment>,
    mode: "onChange",
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        queryKey="recurring-shipment-list"
        title={t("Recurring shipment")}
        fieldKey="name"
        mutationFn={(values, currentRow) =>
          apiService.recurringShipmentService.update(currentRow.id, values)
        }
        formComponent={<RecurringShipmentForm mode="edit" />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey="recurring-shipment-list"
      title={t("Recurring shipment")}
      mutationFn={(values) => apiService.recurringShipmentService.create(values)}
      formComponent={<RecurringShipmentForm mode="create" />}
    />
  );
}
