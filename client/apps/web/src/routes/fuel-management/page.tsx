import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Tabs, TabsContent, TabsList, TabsTab } from "@/components/ui/tabs";
import { Fuel, Gauge, ListTree } from "lucide-react";
import { parseAsString, useQueryState } from "nuqs";
import { lazy } from "react";

const FuelDashboard = lazy(() => import("./_components/fuel-dashboard"));
const ProgramSection = lazy(() => import("./_components/program-section"));
const IndexSection = lazy(() => import("./_components/index-section"));

export function FuelManagementPage() {
  const [activeTab, setActiveTab] = useQueryState("tab", parseAsString.withDefault("dashboard"));

  return (
    <PageLayout
      pageHeaderProps={{
        title: "Fuel Management",
        description:
          "DOE diesel prices, fuel surcharge programs, and automatic surcharge application",
      }}
    >
      <Tabs
        value={activeTab}
        onValueChange={(value) => setActiveTab(value as string)}
        className="flex flex-1 flex-col"
      >
        <div className="border-b border-border">
          <TabsList variant="underline">
            <TabsTab value="dashboard">
              <Gauge className="size-4" />
              Price Dashboard
            </TabsTab>
            <TabsTab value="programs">
              <Fuel className="size-4" />
              Surcharge Programs
            </TabsTab>
            <TabsTab value="indices">
              <ListTree className="size-4" />
              Fuel Indices
            </TabsTab>
          </TabsList>
        </div>
        <TabsContent value="dashboard" className="pt-4">
          <DataTableLazyComponent>
            <FuelDashboard />
          </DataTableLazyComponent>
        </TabsContent>
        <TabsContent value="programs" className="pt-4">
          <DataTableLazyComponent>
            <ProgramSection />
          </DataTableLazyComponent>
        </TabsContent>
        <TabsContent value="indices" className="pt-4">
          <DataTableLazyComponent>
            <IndexSection />
          </DataTableLazyComponent>
        </TabsContent>
      </Tabs>
    </PageLayout>
  );
}
