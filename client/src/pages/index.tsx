import { Suspense, lazy } from "react";

const DailyShipmentCounts = lazy(
  () => import("../components/dashboard/daily-shipment-count"),
);

export default function Index() {
  return (
    <div className="grid w-full grid-cols-1 gap-4 p-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
      <Suspense fallback={<div>Loading...</div>}>
        <DailyShipmentCounts />
      </Suspense>
    </div>
  );
}
