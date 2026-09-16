import { useT } from "@trenova/shared/i18n/use-t";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { Button } from "@trenova/shared/components/ui/button";
import { BotIcon, BlocksIcon } from "lucide-react";
import { Link } from "react-router";
import { AIProviderList } from "./_components/ai-provider-list";

export function AIProvidersPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("AI Providers")}
        description={t(
          "Where AI work goes. Work is offered to providers in priority order, so a cheap model can take a task first and hand off when it cannot.",
        )}
        actions={
          <>
            <Button
              size="sm"
              variant="outline"
              nativeButton={false}
              render={<Link to="/admin/integrations?category=ArtificialIntelligence" />}
            >
              <BlocksIcon className="size-3.5" />
              {t("Marketplace")}
            </Button>
            <Button
              size="sm"
              variant="outline"
              nativeButton={false}
              render={<Link to="/admin/agent-control" />}
            >
              <BotIcon className="size-3.5" />
              {t("Agent Control")}
            </Button>
          </>
        }
      />
      <div className="p-4">
        <AIProviderList />
      </div>
    </AdminPageLayout>
  );
}
