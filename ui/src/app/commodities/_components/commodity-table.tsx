import { DataTable } from "@/components/data-table/data-table";
import { type CommoditySchema } from "@/lib/schemas/commodity-schema";
import { Resource } from "@/types/audit-entry";
import { useMemo } from "react";
import { getColumns } from "./commodity-columns";
import { CreateCommodityModal } from "./commodity-create-modal";
import { EditCommodityModal } from "./commodity-edit-modal";

export default function CommodityTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<CommoditySchema>
      resource={Resource.Commodity}
      name="Commodity"
      link="/commodities/"
      queryKey="commodity-list"
      exportModelName="commodity"
      TableModal={CreateCommodityModal}
      TableEditModal={EditCommodityModal}
      columns={columns}
    />
  );
}
