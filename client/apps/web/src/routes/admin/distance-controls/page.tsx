import { useT } from "@trenova/shared/i18n/use-t";
import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy, Suspense } from "react";
import { PageSkeleton } from "./skeleton";

const DistanceControlForm = lazy(() => import("./_components/distance-control-form"));

export function DistanceControlsPage() {
  const t = useT();

  return (
    <AdminPageLayout>
      <PageHeader
        title={t("Distance Control")}
        description={t("Configure mileage storage behavior and distance profile assignments")}
      />
      <Suspense fallback={<PageSkeleton />}>
        <div className="p-4">
          <DistanceControlForm />
        </div>
      </Suspense>
    </AdminPageLayout>
  );
}
