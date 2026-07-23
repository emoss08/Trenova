import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy, Suspense } from "react";
import { PageSkeleton } from "./skeleton";

const DistanceControlForm = lazy(() => import("./_components/distance-control-form"));

export function DistanceControlsPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Distance Control"
        description="Configure mileage storage behavior and distance profile assignments"
      />
      <Suspense fallback={<PageSkeleton />}>
        <div className="p-4">
          <DistanceControlForm />
        </div>
      </Suspense>
    </AdminPageLayout>
  );
}
