import { PageLayout } from "@/components/navigation/sidebar-layout";
import { useT } from "@trenova/shared/i18n/use-t";
import { IntegrationCatalogCard } from "./_components/integration-catalog";

export function IntegrationsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Integrations"),
        description: t(
          "Connect the outside services Trenova works with: mileage, telematics, email, fuel prices and carrier data.",
        ),
      }}
      className="p-0"
    >
      <IntegrationCatalogCard />
    </PageLayout>
  );
}
