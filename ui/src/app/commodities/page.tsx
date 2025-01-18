import { MetaTags } from "@/components/meta-tags";
import CommodityTable from "./_components/commodity-table";

export function Commodities() {
  return (
    <>
      <MetaTags title="Commodities" description="Commodities" />
      <CommodityTable />
    </>
  );
}
