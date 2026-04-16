import { analytics } from "@/lib/queries/analytics";
import { useSuspenseQuery } from "@tanstack/react-query";
import { ActiveShipmentsCard } from "./cards/active-shipments-card";
import { DetentionAlertsCard } from "./cards/detention-alerts-card";
import { EmptyMilesCard } from "./cards/empty-miles-card";
import { OnTimeCard } from "./cards/on-time-card";
import { ReadyToDispatchCard } from "./cards/ready-to-dispatch-card";
import { RevenueTodayCard } from "./cards/revenue-today-card";

export default function ShipmentAnalytics() {
  const { data } = useSuspenseQuery(analytics.get("shipment-management"));

  return (
    <div className="grid grid-cols-2 gap-2.5 lg:grid-cols-3 xl:grid-cols-6">
      <ActiveShipmentsCard data={data.activeShipments} />
      <OnTimeCard data={data.onTimePercent} />
      <RevenueTodayCard data={data.revenueToday} />
      <EmptyMilesCard data={data.emptyMilePercent} />
      <ReadyToDispatchCard data={data.readyToDispatch} />
      <DetentionAlertsCard data={data.detentionAlerts} />
    </div>
  );
}
