import { Card, CardContent } from "@/components/ui/card";
import { queries } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { AnalyticsPage } from "@/types/analytics";
import { useSuspenseQuery } from "@tanstack/react-query";

export function ShipmentAnalytics() {
  const { data: analytics } = useSuspenseQuery({
    ...queries.analytics.getAnalytics(AnalyticsPage.ShipmentManagement),
  });

  return (
    <dl className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
      <ShipmentCountCard
        count={analytics?.shipmentCountCard?.count}
        trendPercentage={analytics?.shipmentCountCard?.trendPercentage}
      />
      <ShipmentCountCard
        count={analytics?.shipmentCountCard?.count}
        trendPercentage={analytics?.shipmentCountCard?.trendPercentage}
      />
      <ShipmentCountCard
        count={analytics?.shipmentCountCard?.count}
        trendPercentage={analytics?.shipmentCountCard?.trendPercentage}
      />
      <ShipmentCountCard
        count={analytics?.shipmentCountCard?.count}
        trendPercentage={analytics?.shipmentCountCard?.trendPercentage}
      />
    </dl>
  );
}

export function ShipmentCountCard({
  count,
  trendPercentage,
}: {
  count: number;
  trendPercentage: number;
}) {
  return (
    <Card>
      <CardContent>
        <dt className="text-muted-foreground font-medium text-sm">
          Current Shipment Count
        </dt>
        <dd className="mt-2 flex items-baseline gap-x-2.5">
          <span className="text-primary font-semibold text-4xl">{count}</span>
          <span
            className={cn(
              trendPercentage > 0 ? "text-green-500" : "text-destructive",
              "text-primary font-medium text-sm",
            )}
          >
            {trendPercentage > 0 ? "+" : ""}
            {trendPercentage}%
          </span>
        </dd>
      </CardContent>
    </Card>
  );
}
