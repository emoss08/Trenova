import { FormCreateModal } from "@/components/ui/form-create-modal";
import {
  FleetCodeSchema,
  fleetCodeSchema,
} from "@/lib/schemas/fleet-code-schema";
import { Status } from "@/types/common";
import { type TableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { FleetCodeForm } from "./fleet-code-form";

export function CreateFleetCodeModal({ open, onOpenChange }: TableSheetProps) {
  const form = useForm<FleetCodeSchema>({
    resolver: yupResolver(fleetCodeSchema),
    defaultValues: {
      name: "",
      status: Status.Active,
      description: "",
      managerId: undefined,
      revenueGoal: undefined,
      deadheadGoal: undefined,
      color: "",
    },
  });

  return (
    <FormCreateModal
      open={open}
      onOpenChange={onOpenChange}
      title="Fleet Code"
      formComponent={<FleetCodeForm />}
      form={form}
      schema={fleetCodeSchema}
      url="/fleet-codes/"
      queryKey={["fleet-code"]}
    />
  );
}
