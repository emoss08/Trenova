import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  fleetCodeSchema,
  type FleetCodeSchema,
} from "@/lib/schemas/fleet-code-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { FleetCodeForm } from "./fleet-code-form";

export function EditFleetCodeModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<FleetCodeSchema>) {
  const form = useForm({
    resolver: zodResolver(fleetCodeSchema),
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
    />
  );
}
