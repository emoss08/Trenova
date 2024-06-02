import { RateTable } from "@/components/rate-management/rate-table";
import { ComponentLoader } from "@/components/ui/component-loader";

import { Suspense, lazy } from "react";

const TotalActiveRateCard = lazy(
  () =>
    import("../../components/rate-management/cards/total-active-rates-card"),
);

export default function RateManagement() {
  return (
    <>
      <Suspense fallback={<ComponentLoader />}>
        <div className="mb-10 grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-3">
          <TotalActiveRateCard />
          <TotalActiveRateCard />
          <TotalActiveRateCard />
        </div>
      </Suspense>
      <RateTable />
    </>
  );
}
