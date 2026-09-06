import { PageLayout } from "@/components/navigation/sidebar-layout";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";

const FleetSafetyConsole = lazy(() => import("./_components/fleet-safety-console"));

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
        <DataTableLazyComponent>
          <FleetSafetyConsole />
        </DataTableLazyComponent>
      </div>
    </PageLayout>
  );
}
