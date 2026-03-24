import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { IntegrationCatalogCard } from "./_components/integration-catalog";

export function IntegrationsPage() {
  return (
    <AdminPageLayout>
      <IntegrationCatalogCard />
    </AdminPageLayout>
  );
}
