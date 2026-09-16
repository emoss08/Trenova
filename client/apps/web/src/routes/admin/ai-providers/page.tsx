import { useT } from "@trenova/shared/i18n/use-t";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { AIProviderList } from "./_components/ai-provider-list";

export function AIProvidersPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("AI Providers")}
        description={t(
          "Connect model endpoints and choose which one handles each kind of work. Hosted APIs, gateways, and models you run on your own hardware are all configured here.",
        )}
      />
      <div className="p-4">
        <AIProviderList />
      </div>
    </AdminPageLayout>
  );
}
