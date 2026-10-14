import { useT } from "@trenova/shared/i18n/use-t";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy } from "react";
import { SettlementControlSkeleton } from "./skeleton";

const SettlementControlForm = lazy(() => import("./_components/settlement-control-form"));

export function SettlementControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Settlement control"),
        description: t(
          "Configure pay periods, accrual triggers, exception policies, and escrow interest",
        ),
      }}
    >
      <LazyComponent fallback={<SettlementControlSkeleton />}>
        <SettlementControlForm />
      </LazyComponent>
    </PageLayout>
  );
}
