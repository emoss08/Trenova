import { useT } from "@trenova/shared/i18n/use-t";
import { DataTableLazyComponent } from "@trenova/shared/components/error-boundary";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { Tabs, TabsList, TabsPanel, TabsTab } from "@trenova/shared/components/ui/tabs";
import { lazy, Suspense } from "react";

const SubscriptionTable = lazy(() => import("./_components/subscription-table"));
const NotificationList = lazy(() => import("./_components/notification-list"));

export function TableChangeAlertPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Table Change Alert")}
        description={t("Monitor and review system activity across your organization")}
      />
      <div className="px-4">
        <Tabs defaultValue="subscriptions">
          <TabsList variant="underline">
            <TabsTab value="subscriptions">{t("Subscriptions")}</TabsTab>
            <TabsTab value="notifications">{t("Notifications")}</TabsTab>
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
