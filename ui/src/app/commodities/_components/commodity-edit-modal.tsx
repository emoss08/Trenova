import { FormEditModal } from "@/components/ui/form-edit-modal";
import {
  commoditySchema,
  CommoditySchema,
} from "@/lib/schemas/commodity-schema";
import { type EditTableSheetProps } from "@/types/data-table";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { CommodityForm } from "./commodity-form";

export function EditCommodityModal({
  currentRecord,
}: EditTableSheetProps<CommoditySchema>) {
  const form = useForm({
    resolver: zodResolver(commoditySchema),
    defaultValues: currentRecord,
  });

  return (
    <FormEditModal
      currentRecord={currentRecord}
      url="/commodities/"
      title="Commodity"
      queryKey="commodity-list"
      formComponent={<CommodityForm />}
      fieldKey="name"
      className="max-w-[500px]"
      form={form}
    />
  );
}
