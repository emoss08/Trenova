import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { MonitoringWorkspace } from "./_components/monitoring-workspace";

export function CarrierMonitoringPage() {
  return (
    <AdminPageLayout>
      <MonitoringWorkspace />
    </AdminPageLayout>
  );
}
