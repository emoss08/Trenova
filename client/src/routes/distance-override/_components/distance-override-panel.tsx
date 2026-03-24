import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import {
  distanceOverrideSchema,
  type DistanceOverride,
} from "@/types/distance-override";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { DistanceOverrideForm } from "./distance-override-form";

export function DistanceOverridePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<DistanceOverride>) {
  const form = useForm({
    resolver: zodResolver(distanceOverrideSchema),
    defaultValues: {
      originLocationId: "",
      destinationLocationId: "",
      intermediateStops: [],
      customerId: null,
      distance: 0,
    },
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/distance-overrides/"
        queryKey="distance-override-list"
        title="Distance Override"
        fieldKey="originLocationId"
        formComponent={<DistanceOverrideForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/distance-overrides/"
      queryKey="distance-override-list"
      title="Distance Override"
      formComponent={<DistanceOverrideForm />}
    />
  );
}
