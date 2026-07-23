import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@/types/data-table";
import { commoditySchema, type Commodity } from "@/types/commodity";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { CommodityForm } from "./commodity-form";

export function CommodityPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<Commodity>) {
  const form = useForm({
    resolver: zodResolver(commoditySchema),
    defaultValues: {
      status: "Active",
      name: "",
      description: "",
      hazardousMaterialId: null,
      minTemperature: null,
      maxTemperature: null,
      weightPerUnit: null,
      linearFeetPerUnit: null,
      maxQuantityPerShipment: null,
      freightClass: null,
      loadingInstructions: "",
      stackable: false,
      fragile: false,
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
        url="/commodities/"
        queryKey="commodity-list"
        title="Commodity"
        fieldKey="name"
        formComponent={<CommodityForm />}
      />
    );
  }

  return (
    <FormCreatePanel
      open={open}
      onOpenChange={onOpenChange}
      form={form}
      url="/commodities/"
      queryKey="commodity-list"
      title="Commodity"
      formComponent={<CommodityForm />}
    />
  );
}
