import { useT } from "@trenova/shared/i18n/use-t";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { Dataflow02Icon, FuelIcon, Speedometer03Icon } from "@trenova/shared/components/icons";
import { parseAsString, useQueryState } from "nuqs";
import { lazy } from "react";
import {
  FuelDashboardSkeleton,
  IndexSectionSkeleton,
  ProgramSectionSkeleton,
} from "./_components/fuel-management-skeleton";

const FuelDashboard = lazy(() => import("./_components/fuel-dashboard"));
const ProgramSection = lazy(() => import("./_components/program-section"));
const IndexSection = lazy(() => import("./_components/index-section"));

export function FuelManagementPage() {
  const t = useT();

  const [activeTab, setActiveTab] = useQueryState("tab", parseAsString.withDefault("dashboard"));

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Fuel management"),
        description: t(
          "DOE diesel prices, fuel surcharge programs, and automatic surcharge application",
        ),
      }}
    >
      <Tabs
        value={activeTab}
        onValueChange={(value) => setActiveTab(value as string)}
        className="flex flex-1 flex-col"
      >
        <div className="border-border border-b">
          <TabsList variant="underline">
            <TabsTab value="dashboard">
              <Speedometer03Icon className="size-4" />
              {t("Price dashboard")}
            </TabsTab>
            <TabsTab value="programs">
              <FuelIcon className="size-4" />
              {t("Surcharge programs")}
            </TabsTab>
            <TabsTab value="indices">
              <Dataflow02Icon className="size-4" />
              {t("Fuel indices")}
            </TabsTab>
          </TabsList>
        </div>
        <TabsContent value="dashboard" className="pt-4">
          <LazyComponent fallback={<FuelDashboardSkeleton />}>
            <FuelDashboard onOpenIndices={() => void setActiveTab("indices")} />
          </LazyComponent>
        </TabsContent>
        <TabsContent value="programs" className="pt-4">
          <LazyComponent fallback={<ProgramSectionSkeleton />}>
            <ProgramSection />
          </LazyComponent>
        </TabsContent>
        <TabsContent value="indices" className="pt-4">
          <LazyComponent fallback={<IndexSectionSkeleton />}>
            <IndexSection />
          </LazyComponent>
        </TabsContent>
      </Tabs>
    </PageLayout>
  );
}
