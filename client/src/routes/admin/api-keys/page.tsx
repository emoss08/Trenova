import { DataTableLazyComponent } from "@/components/error-boundary";
import { MetricSkeleton } from "@/components/metric-skeleton";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy, Suspense } from "react";

const ApiKeyAnalytics = lazy(() => import("./_components/analytics/api-key-analytics"));
const Table = lazy(() => import("./_components/api-key-table"));

export function APIKeysPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="API Keys"
        description="Provision bearer credentials for third-party systems with direct, tenant-scoped permissions."
        className="p-0 py-4"
      />
      <Suspense fallback={<MetricSkeleton cardClassName="h-[125px]" />}>
        <ApiKeyAnalytics />
      </Suspense>
      <DataTableLazyComponent>
        <Table />
      </DataTableLazyComponent>
    </AdminPageLayout>
  );
}
