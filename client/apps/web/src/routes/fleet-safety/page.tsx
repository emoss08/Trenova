import { PageLayout } from "@/components/navigation/sidebar-layout";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { FleetSafetySkeleton } from "./_components/fleet-safety-skeleton";
import { fleetSafetyQuery } from "./_components/queries";

const FleetSafetyConsole = lazy(() => import("./_components/fleet-safety-console"));

// The page opens on twelve months across every terminal; that is the roll-up
// it paints first, and the only one it can know before a filter is chosen.
export const prefetch: RoutePrefetch = () => [
  fleetSafetyQuery({ windowMonths: 12, fleetCodeId: null }),
];

export function FleetSafetyPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Fleet Safety",
        description:
          "The fleet read the way a safety director thinks: the seven CSA BASICs, the trend behind them, and which terminals and drivers are carrying the weight.",
      }}
    >
      <div className="flex flex-col gap-4">
        <DataTableLazyComponent fallback={<FleetSafetySkeleton />}>
          <FleetSafetyConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
