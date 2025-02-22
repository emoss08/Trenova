import { MetaTags } from "@/components/meta-tags";
import { SuspenseLoader } from "@/components/ui/component-loader";
import { lazy } from "react";

// Lazy Loaded Components
const CommodityTable = lazy(() => import("./_components/commodity-table"));

export function Commodities() {
  return (
    <>
      <MetaTags title="Commodities" description="Commodities" />
      <SuspenseLoader>
        <CommodityTable />
      </SuspenseLoader>
    </>
  );
}
