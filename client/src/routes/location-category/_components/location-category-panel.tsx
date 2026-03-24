import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import { locationCategorySchema, type LocationCategory } from "@/types/location-category";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { LocationCategoryForm } from "./location-category-form";

export function LocationCategoryPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<LocationCategory>) {
  const form = useForm({
    resolver: zodResolver(locationCategorySchema),
    defaultValues: {
      name: "",
      description: "",
      type: "Terminal",
      facilityType: null,
      color: null,
      hasSecureParking: false,
      requiresAppointment: false,
      allowsOvernight: false,
      hasRestroom: false,
    },
  });

  if (mode === "edit") {
    return (
      <FormEditPanel
        open={open}
        onOpenChange={onOpenChange}
        row={row}
        form={form}
        url="/location-categories/"
        queryKey="location-category-list"
        title="Location Category"
        fieldKey="name"
        formComponent={<LocationCategoryForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/location-categories/"
      queryKey="location-category-list"
      title="Location Category"
      formComponent={<LocationCategoryForm />}
    />
  );
}
