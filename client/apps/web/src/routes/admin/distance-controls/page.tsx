import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy, Suspense } from "react";
import { PageSkeleton } from "./skeleton";

const DistanceControlForm = lazy(() => import("./_components/distance-control-form"));

export function DistanceControlsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Distance Control"),
        description: t("Configure mileage storage behavior and distance profile assignments"),
      }}
    >
      <Suspense fallback={<PageSkeleton />}>
        <DistanceControlForm />
      </Suspense>
    </PageLayout>
  );
}
