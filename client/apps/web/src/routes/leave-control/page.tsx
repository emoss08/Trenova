import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { lazy } from "react";

const LeaveControlForm = lazy(() => import("./_components/leave-control-form"));

export function LeaveControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Leave Settings"),
        description: t(
          "How family and medical leave is measured, and what an employee must do to qualify",
        ),
      }}
    >
      <SuspenseLoader>
        <LeaveControlForm />
      </SuspenseLoader>
    </PageLayout>
  );
}
