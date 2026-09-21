import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { usePermission } from "@/hooks/use-permission";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { RadarIcon, ShieldCheckIcon } from "lucide-react";
import { parseAsStringLiteral, useQueryState } from "nuqs";
import { lazy } from "react";
import { FleetSafetySkeleton } from "./_components/fleet-safety-skeleton";
import { fleetSafetyQuery } from "./_components/queries";

const FleetSafetyConsole = lazy(() => import("./_components/fleet-safety-console"));
const MyDotIntelligence = lazy(() =>
  import("./_components/my-dot-intelligence").then((module) => ({
    default: module.MyDotIntelligence,
  })),
);

const FLEET_SAFETY_VIEWS = ["fleet", "my-dot"] as const;
type FleetSafetyView = (typeof FLEET_SAFETY_VIEWS)[number];

const viewParser = parseAsStringLiteral(FLEET_SAFETY_VIEWS).withDefault("fleet");

// The page opens on twelve months across every terminal; that is the roll-up
// it paints first, and the only one it can know before a filter is chosen.
export const prefetch: RoutePrefetch = () => [
  fleetSafetyQuery({ windowMonths: 12, fleetCodeId: null }),
];

export function FleetSafetyPage() {
  const t = useT();

  const [view, setView] = useQueryState("view", viewParser);
  const { allowed: canReadFleet } = usePermission(Resource.WorkerSafetyEvent, Operation.Read);
  const { allowed: canReadIntel } = usePermission(Resource.CarrierIntelligence, Operation.Read);

  const activeView: FleetSafetyView = !canReadFleet && canReadIntel ? "my-dot" : view;

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Fleet safety"),
        description: t(
          "The fleet read the way a safety director thinks: the seven CSA BASICs, the trend behind them, and which terminals and drivers are carrying the weight.",
        ),
      }}
    >
      <Tabs
        value={activeView}
        onValueChange={(value) => void setView(value as FleetSafetyView)}
        className="flex flex-1 flex-col"
      >
        <div className="border-border border-b">
          <TabsList variant="underline">
            {canReadFleet ? (
              <TabsTab value="fleet">
                <ShieldCheckIcon className="size-4" />
                {t("Fleet")}
              </TabsTab>
            ) : null}
            {canReadIntel ? (
              <TabsTab value="my-dot">
                <RadarIcon className="size-4" />
                {t("My DOT")}
              </TabsTab>
            ) : null}
          </TabsList>
        </div>
        <TabsContent value="fleet" className="pt-4">
          <div className="flex flex-col gap-4">
            <DataTableLazyComponent fallback={<FleetSafetySkeleton />}>
              <FleetSafetyConsole />
            </DataTableLazyComponent>
          </div>
        </TabsContent>
        <TabsContent value="my-dot" className="pt-4">
          <DataTableLazyComponent>
            <MyDotIntelligence />
          </DataTableLazyComponent>
        </TabsContent>
      </Tabs>
    </PageLayout>
  );
}
