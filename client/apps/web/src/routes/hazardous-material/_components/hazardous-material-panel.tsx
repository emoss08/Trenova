import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { apiService } from "@/services/api";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import { hazardousMaterialSchema, type HazardousMaterial } from "@/types/hazardous-material";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { HazardousMaterialForm } from "./hazardous-material-form";

export function HazardousMaterialPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<HazardousMaterial>) {
  const t = useT();

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
        queryKey="hazardous-material-list"
        title={t("Hazardous material")}
        fieldKey="name"
        formComponent={<HazardousMaterialForm />}
        mutationFn={(values, currentRow) => {
          if (!currentRow.id) {
            throw new Error("No Hazardous Material ID selected");
          }

          return apiService.hazardousMaterialService.update(currentRow.id, values);
        }}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      queryKey="hazardous-material-list"
      title={t("Hazardous material")}
      formComponent={<HazardousMaterialForm isEditing />}
      mutationFn={(values) => apiService.hazardousMaterialService.create(values)}
    />
  );
}
