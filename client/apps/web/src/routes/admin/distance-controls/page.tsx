import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { DistanceControlSkeleton } from "./skeleton";

const DistanceControlForm = lazy(() => import("./_components/distance-control-form"));

export function DistanceControlsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Distance control"),
        description: t("Configure mileage storage behavior and distance profile assignments"),
      }}
    >
      <LazyComponent fallback={<DistanceControlSkeleton />}>
        <DistanceControlForm />
      </LazyComponent>
    </PageLayout>
  );
}
