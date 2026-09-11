import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { FleetCodeRow } from "@/lib/graphql/fleet-code-table";
import { fleetCodeSchema } from "@trenova/shared/types/fleet-code";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { FleetCodeForm } from "./fleet-code-form";

export function FleetCodePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<FleetCodeRow>) {
  const t = useT();

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
        title={t("Fleet Code")}
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
      title={t("Fleet Code")}
      formComponent={<FleetCodeForm />}
    />
  );
}
