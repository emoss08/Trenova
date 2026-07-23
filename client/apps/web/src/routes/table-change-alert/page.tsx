import { DataTableLazyComponent } from "@/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@/components/ui/tabs";
import { lazy, Suspense } from "react";

const SubscriptionTable = lazy(() => import("./_components/subscription-table"));
const NotificationList = lazy(() => import("./_components/notification-list"));

export function TableChangeAlertPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Table Change Alert"
        description="Monitor and review system activity across your organization"
      />
      <div className="px-4">
        <Tabs defaultValue="subscriptions">
          <TabsList variant="underline">
            <TabsTab value="subscriptions">Subscriptions</TabsTab>
            <TabsTab value="notifications">Notifications</TabsTab>
          </TabsList>
          <TabsPanel value="subscriptions">
            <DataTableLazyComponent>
              <SubscriptionTable />
            </DataTableLazyComponent>
          </TabsPanel>
          <TabsPanel value="notifications">
            <Suspense>
              <NotificationList />
            </Suspense>
          </TabsPanel>
        </Tabs>
      </div>
    </AdminPageLayout>
  );
}
