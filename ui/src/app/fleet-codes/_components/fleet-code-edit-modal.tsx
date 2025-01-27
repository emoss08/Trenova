import { FormEditModal } from "@/components/ui/form-edit-model";
import {
  fleetCodeSchema,
  type FleetCodeSchema,
} from "@/lib/schemas/fleet-code-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { FleetCodeForm } from "./fleet-code-form";

export function EditFleetCodeModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<FleetCodeSchema>) {
  const form = useForm<FleetCodeSchema>({
    resolver: yupResolver(fleetCodeSchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      url="/fleet-codes/"
      title="Fleet Code"
      queryKey="fleet-code-list"
      formComponent={<FleetCodeForm />}
      fieldKey="name"
      form={form}
      schema={fleetCodeSchema}
    />
  );
}
