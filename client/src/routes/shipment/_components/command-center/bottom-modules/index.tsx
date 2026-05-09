import { analytics } from "@/lib/queries/analytics";
import { useSuspenseQuery } from "@tanstack/react-query";
import {
  type DeepPartial,
  type ShipmentAnalyticsData,
  mergeShipmentAnalyticsWithDefaults,
} from "../../analytics/mock-data";
import { ActivityFeed } from "./activity-feed";
import { CustomerMix } from "./customer-mix";
import { LaneHeatmap } from "./lane-heatmap";

export default function BottomModules() {
  return (
    <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
      <ActivityFeed />
      <ShipmentAnalyticsModules />
    </div>
  );
}

function ShipmentAnalyticsModules() {
  const { data } = useSuspenseQuery(analytics.get("shipment-management"));
  const shipmentAnalytics = mergeShipmentAnalyticsWithDefaults(
    data as DeepPartial<ShipmentAnalyticsData>,
  );

  return (
    <>
      <LaneHeatmap data={shipmentAnalytics.laneHeatmap} />
      <CustomerMix
        customerMix={shipmentAnalytics.customerMix}
        tomorrowsPickups={shipmentAnalytics.tomorrowsPickups}
      />
    </>
  );
}

export { ActivityFeed, CustomerMix, LaneHeatmap };
