import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { LazyComponent } from "@trenova/shared/components/error-boundary";
import { lazy } from "react";
import { AccountingControlSkeleton } from "./skeleton";

const AccountingControlForm = lazy(() => import("./_components/accounting-control-form"));

export function AccountingControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Accounting control"),
        description: t("Configure and manage your accounting control settings"),
      }}
    >
      <LazyComponent fallback={<AccountingControlSkeleton />}>
        <AccountingControlForm />
      </LazyComponent>
    </PageLayout>
  );
}
