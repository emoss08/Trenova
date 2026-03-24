import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import { shipmentTypeSchema, type ShipmentType } from "@/types/shipment-type";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { ShipmentTypeForm } from "./shipment-type-form";

export function ShipmentTypePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<ShipmentType>) {
  const form = useForm({
    resolver: zodResolver(shipmentTypeSchema),
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
        url="/shipment-types/"
        queryKey="shipment-type-list"
        title="Shipment Type"
        fieldKey="code"
        formComponent={<ShipmentTypeForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/shipment-types/"
      queryKey="shipment-type-list"
      title="Shipment Type"
      formComponent={<ShipmentTypeForm />}
    />
  );
}
