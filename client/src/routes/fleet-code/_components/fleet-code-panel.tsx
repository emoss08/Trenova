import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import { fleetCodeSchema, type FleetCode } from "@/types/fleet-code";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { FleetCodeForm } from "./fleet-code-form";

export function FleetCodePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<FleetCode>) {
  const form = useForm({
    resolver: zodResolver(fleetCodeSchema),
    defaultValues: {
      code: "",
      status: "Active",
      description: "",
      managerId: "",
      revenueGoal: undefined,
      deadheadGoal: undefined,
      color: "",
    },
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/fleet-codes/"
        queryKey="fleet-code-list"
        title="Fleet Code"
        fieldKey="code"
        formComponent={<FleetCodeForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/fleet-codes/"
      queryKey="fleet-code-list"
      title="Fleet Code"
      formComponent={<FleetCodeForm />}
    />
  );
}
