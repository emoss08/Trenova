import { Suspense, lazy } from "react";

const DailyShipmentCounts = lazy(
  () => import("../components/dashboard/cards/daily-shipment-count-card"),
);

const NewShipmentCard = lazy(
  () => import("../components/dashboard/cards/new-shipment-card"),
);

export default function Index() {
  return (
    <div className="grid size-full grid-cols-1 gap-4 p-4 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4">
      <Suspense fallback={<div>Loading...</div>}>
        <NewShipmentCard />
        <DailyShipmentCounts />
      </Suspense>
    </div>
  );
}
