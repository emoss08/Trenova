import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { usePermission } from "@/hooks/use-permission";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useT } from "@trenova/shared/i18n/use-t";
import { Operation, Resource } from "@trenova/shared/types/permission";
import { ActivityIcon, BotIcon, LayoutDashboardIcon, PlugZapIcon } from "lucide-react";
import { useQueryState } from "nuqs";
import { lazy, useCallback } from "react";
import { AI_CONTROL_TAB_PARAM, aiControlTabParser, type AIControlTab } from "./ai-control-tabs";

const OverviewTab = lazy(() => import("./_components/overview/overview-tab"));
const AgentsTab = lazy(() => import("./_components/agents/agents-tab"));
const ProvidersTab = lazy(() => import("./_components/providers/providers-tab"));
const ActivityTab = lazy(() => import("./_components/activity/activity-tab"));

/**
 * One place for everything AI in the organization: where work goes
 * (providers), what it may do (agents), and what it did (activity).
 */
export function AgentControlPage() {
  const t = useT();
  const [tab, setTab] = useQueryState(AI_CONTROL_TAB_PARAM, aiControlTabParser);

  const { allowed: canReadAgents } = usePermission(Resource.AgentDefinition, Operation.Read);
  const { allowed: canReadProviders } = usePermission(Resource.AIProvider, Operation.Read);
  const { allowed: canReadRuns } = usePermission(Resource.AgentRun, Operation.Read);

  const openProviders = useCallback(() => void setTab("providers"), [setTab]);

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("AI Control")}
        description={t(
          "Providers say where AI work goes, agents say what it may do, and activity shows what it did.",
        )}
      />
      <div className="flex flex-col gap-4 px-4">
        <Tabs
          value={tab}
          className="gap-4"
          onValueChange={(value) => void setTab(value as AIControlTab)}
        >
          <TabsList variant="underline">
            <TabsTab value="overview">
              <LayoutDashboardIcon size={16} aria-hidden="true" />
              {t("Overview")}
            </TabsTab>
            {canReadAgents && (
              <TabsTab value="agents">
                <BotIcon size={16} aria-hidden="true" />
                {t("Agents")}
              </TabsTab>
            )}
            {canReadProviders && (
              <TabsTab value="providers">
                <PlugZapIcon size={16} aria-hidden="true" />
                {t("Providers")}
              </TabsTab>
            )}
            {canReadRuns && (
              <TabsTab value="activity">
                <ActivityIcon size={16} aria-hidden="true" />
                {t("Activity")}
              </TabsTab>
            )}
          </TabsList>

          <TabsContent value="overview">
            <DataTableLazyComponent>
              <OverviewTab onOpenProviders={openProviders} />
            </DataTableLazyComponent>
          </TabsContent>
          {canReadAgents && (
            <TabsContent value="agents">
              <DataTableLazyComponent>
                <AgentsTab />
              </DataTableLazyComponent>
            </TabsContent>
          )}
          {canReadProviders && (
            <TabsContent value="providers">
              <DataTableLazyComponent>
                <ProvidersTab />
              </DataTableLazyComponent>
            </TabsContent>
          )}
          {canReadRuns && (
            <TabsContent value="activity">
              <DataTableLazyComponent>
                <ActivityTab />
              </DataTableLazyComponent>
            </TabsContent>
          )}
        </Tabs>
      </div>
    </AdminPageLayout>
  );
}
