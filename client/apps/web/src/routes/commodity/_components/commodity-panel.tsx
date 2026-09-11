import { useT } from "@trenova/shared/i18n/use-t";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import type { DataTablePanelProps } from "@trenova/shared/types/data-table";
import type { CommodityRow } from "@/lib/graphql/commodity-table";
import { commoditySchema } from "@trenova/shared/types/commodity";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { CommodityForm } from "./commodity-form";

export function CommodityPanel({
  open,
  onOpenChange,
  mode,
  row,
}: DataTablePanelProps<CommodityRow>) {
  const t = useT();

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
        title={t("Commodity")}
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
      title={t("Commodity")}
      formComponent={<CommodityForm />}
    />
  );
}
