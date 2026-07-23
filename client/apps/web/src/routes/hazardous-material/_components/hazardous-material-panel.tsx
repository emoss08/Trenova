import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import {
  hazardousMaterialSchema,
  type HazardousMaterial,
} from "@/types/hazardous-material";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { HazardousMaterialForm } from "./hazardous-material-form";

export function HazardousMaterialPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<HazardousMaterial>) {
  const form = useForm({
    resolver: zodResolver(hazardousMaterialSchema),
    defaultValues: {
      status: "Active",
      name: "",
      description: "",
      class: "HazardClass1",
      packingGroup: "I",
      unNumber: "",
      subsidiaryHazardClass: "",
      ergGuideNumber: "",
      labelCodes: "",
      specialProvisions: "",
      properShippingName: "",
      handlingInstructions: "",
      emergencyContact: "",
      emergencyContactPhoneNumber: "",
      quantityThreshold: "",
      placardRequired: false,
      isReportableQuantity: false,
      marinePollutant: false,
      inhalationHazard: false,
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
        url="/hazardous-materials/"
        queryKey="hazardous-material-list"
        title="Hazardous Material"
        fieldKey="name"
        formComponent={<HazardousMaterialForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/hazardous-materials/"
      queryKey="hazardous-material-list"
      title="Hazardous Material"
      formComponent={<HazardousMaterialForm isEditing />}
    />
  );
}
