import { LazyComponent } from "@/components/error-boundary";
import { MetaTags } from "@/components/meta-tags";
import { lazy } from "react";

// Lazy Loaded Components
const CommodityTable = lazy(() => import("./_components/commodity-table"));

export function Commodities() {
  return (
    <>
      <MetaTags title="Commodities" description="Commodities" />
      <LazyComponent>
        <CommodityTable />
      </LazyComponent>
    </>
  );
}
