import { analytics } from "@/lib/queries/analytics";
import { useQuery } from "@tanstack/react-query";
import {
  type DeepPartial,
  type ShipmentAnalyticsData,
  mergeShipmentAnalyticsWithDefaults,
} from "../../analytics/mock-data";
import { ActivityFeed } from "./activity-feed";
import { CustomerMix } from "./customer-mix";
import { LaneHeatmap } from "./lane-heatmap";

export default function BottomModules({
  backgroundEnabled = true,
}: {
  backgroundEnabled?: boolean;
}) {
  return (
    <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
      <ActivityFeed enabled={backgroundEnabled} />
      <ShipmentAnalyticsModules enabled={backgroundEnabled} />
    </div>
  );
}

function ShipmentAnalyticsModules({ enabled }: { enabled: boolean }) {
  const { data } = useQuery({
    ...analytics.get("shipment-management"),
    enabled,
  });
  const shipmentAnalytics = mergeShipmentAnalyticsWithDefaults(
    (data ?? {}) as DeepPartial<ShipmentAnalyticsData>,
  );

  return (
    <>
      <LaneHeatmap data={shipmentAnalytics.laneHeatmap} />
      <CustomerMix
        customerMix={shipmentAnalytics.customerMix}
        tomorrowsPickups={shipmentAnalytics.tomorrowsPickups}
        enabled={enabled}
      />
    </>
  );
}

export { ActivityFeed, CustomerMix, LaneHeatmap };
