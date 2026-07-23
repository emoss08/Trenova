import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import { distanceProfileSchema, type DistanceProfile } from "@/types/distance-profile";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { DistanceProfileForm } from "./distance-profile-form";

const defaultValues: DistanceProfile = {
  id: "",
  organizationId: "",
  businessUnitId: "",
  name: "",
  description: "",
  status: "Active",
  isDefault: false,
  provider: "PCMiler",
  dataVersion: "Current",
  region: "NA",
  routingType: "Practical",
  distanceUnits: "Miles",
  locationGranularity: "PostalCode",
  profileName: "",
  highwayOnly: false,
  tollRoads: true,
  bordersOpen: true,
  includeTollData: false,
};

export function DistanceProfilePanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<DistanceProfile>) {
  const form = useForm<DistanceProfile>({
    resolver: zodResolver(distanceProfileSchema),
    defaultValues,
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/distance-profiles/"
        queryKey="distance-profile-list"
        title="Distance Profile"
        fieldKey="name"
        formComponent={<DistanceProfileForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/distance-profiles/"
      queryKey="distance-profile-list"
      title="Distance Profile"
      formComponent={<DistanceProfileForm />}
    />
  );
}
