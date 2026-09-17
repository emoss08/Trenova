import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { CarrierMonitoringWorkspace } from "./_components/carrier-monitoring-workspace";

export function CarrierMonitoringPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Carrier Monitoring"),
        description: t(
          "Authority, insurance and safety changes on the carriers you watch, the reviews they call for, and what the provider is costing",
        ),
      }}
    >
      <CarrierMonitoringWorkspace />
    </PageLayout>
  );
}
