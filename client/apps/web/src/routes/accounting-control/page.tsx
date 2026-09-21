import { useT } from "@trenova/shared/i18n/use-t";
import { PageLayout } from "@/components/navigation/sidebar-layout";
import { lazy, Suspense } from "react";
import { PageSkeleton } from "./skeleton";

const AccountingControlForm = lazy(() => import("./_components/accounting-control-form"));

export function AccountingControlPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Accounting Control"),
        description: t("Configure and manage your accounting control settings"),
      }}
    >
      <Suspense fallback={<PageSkeleton />}>
        <div className="p-4">
          <AccountingControlForm />
        </div>
      </Suspense>
    </PageLayout>
  );
}
