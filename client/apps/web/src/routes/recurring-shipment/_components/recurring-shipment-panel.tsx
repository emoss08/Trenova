import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { recurringShipmentSchema, type RecurringShipment } from "@/types/recurring-shipment";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
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
}: DataTablePanelProps<RecurringShipment>) {
  const form = useForm({
    resolver: zodResolver(recurringShipmentSchema),
    defaultValues,
    mode: "onChange",
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/recurring-shipments/"
        queryKey="recurring-shipment-list"
        title="Recurring Shipment"
        fieldKey="name"
        size="lg"
        formComponent={<RecurringShipmentForm mode="edit" />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/recurring-shipments/"
      queryKey="recurring-shipment-list"
      title="Recurring Shipment"
      size="lg"
      formComponent={<RecurringShipmentForm mode="create" />}
    />
  );
}
