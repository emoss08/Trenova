/**
 * # Copyright 2023-2025 Eric Moss
 * # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md
 */

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
