import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { LeaveControlSkeleton } from "./skeleton";

const LeaveControlForm = lazy(() => import("./_components/leave-control-form"));

export function LeaveControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Leave settings"),
        description: t(
          "How family and medical leave is measured, and what an employee must do to qualify",
        ),
      }}
    >
      <LazyComponent fallback={<LeaveControlSkeleton />}>
        <LeaveControlForm />
      </LazyComponent>
    </PageLayout>
  );
}
