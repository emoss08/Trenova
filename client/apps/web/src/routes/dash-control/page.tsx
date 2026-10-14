import { useT } from "@trenova/shared/i18n/use-t";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { DashControlSkeleton } from "./skeleton";

const DashControlForm = lazy(() => import("./_components/dash-control-form"));

export function DashControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Dash control"),
        description: t("Choose what drivers can see and do in the Dash driver portal"),
      }}
    >
      <LazyComponent fallback={<DashControlSkeleton />}>
        <DashControlForm />
      </LazyComponent>
    </PageLayout>
  );
}
