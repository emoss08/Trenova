import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useT } from "@trenova/shared/i18n/use-t";
import { MonitoringWorkspace } from "./_components/monitoring-workspace";

export function CarrierMonitoringPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Carrier monitoring"),
        description: t(
          "Changes detected on the carriers you watch: authority, insurance and safety events to review, and the watchlist that produces them.",
        ),
      }}
    >
      <MonitoringWorkspace />
    </PageLayout>
  );
}
