import { FormEditModal } from "@/components/ui/form-edit-model";
import {
  commoditySchema,
  CommoditySchema,
} from "@/lib/schemas/commodity-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { yupResolver } from "@hookform/resolvers/yup";
import { useForm } from "react-hook-form";
import { CommodityForm } from "./commodity-form";

export function EditCommodityModal({
  open,
  onOpenChange,
  currentRecord,
}: EditTableSheetProps<CommoditySchema>) {
  const form = useForm<CommoditySchema>({
    resolver: yupResolver(commoditySchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      open={open}
      onOpenChange={onOpenChange}
      url="/commodities/"
      title="Commodity"
      queryKey="commodity-list"
      formComponent={<CommodityForm />}
      fieldKey="name"
      className="max-w-[500px]"
      form={form}
      schema={commoditySchema}
    />
  );
}
