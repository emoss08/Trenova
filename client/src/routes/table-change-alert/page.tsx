import { DataTableLazyComponent } from "@/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { Tabs, TabsPanel, TabsList, TabsTab } from "@/components/ui/tabs";
import { lazy, Suspense } from "react";

const SubscriptionTable = lazy(() => import("./_components/subscription-table"));
const NotificationList = lazy(() => import("./_components/notification-list"));

export function TableChangeAlertPage() {
  return (
    <PageLayout
      pageHeaderProps={{
        title: "Table Change Alerts",
        description:
          "Subscribe to changes on database tables and receive real-time notifications",
      }}
    >
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
    </PageLayout>
  );
}
