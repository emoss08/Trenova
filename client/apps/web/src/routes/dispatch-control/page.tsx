import { useT } from "@trenova/shared/i18n/use-t";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { DispatchControlSkeleton } from "./skeleton";

const DispatchControlForm = lazy(() => import("./_components/dispatch-control-form"));

export function DispatchControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Dispatch control"),
        description: t("Configure and manage your dispatch control settings"),
      }}
    >
      <LazyComponent fallback={<DispatchControlSkeleton />}>
        <DispatchControlForm />
      </LazyComponent>
    </PageLayout>
  );
}
