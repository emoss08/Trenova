import { useT } from "@trenova/shared/i18n/use-t";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";

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
      <SuspenseLoader>
        <DashControlForm />
      </SuspenseLoader>
    </PageLayout>
  );
}
