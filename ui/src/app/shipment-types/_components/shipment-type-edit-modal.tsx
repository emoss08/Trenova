import { FormEditModal } from "@/components/ui/form-edit-model";
import {
  shipmentTypeSchema,
  ShipmentTypeSchema,
} from "@/lib/schemas/shipment-type-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { ShipmentTypeForm } from "./shipment-type-form";

export function EditShipmentTypeModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<ShipmentTypeSchema>) {
  const form = useForm<ShipmentTypeSchema>({
    resolver: yupResolver(shipmentTypeSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      url="/shipment-types/"
      title="Shipment Type"
      queryKey="shipment-type-list"
      formComponent={<ShipmentTypeForm />}
      fieldKey="code"
      form={form}
      schema={shipmentTypeSchema}
    />
  );
}
