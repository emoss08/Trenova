import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { lazy, Suspense } from "react";
import { PageSkeleton } from "./skeleton";

const AccountingControlForm = lazy(() => import("./_components/accounting-control-form"));

export function AccountingControlPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Accounting Control"
        description="Configure and manage your accounting control settings"
        className="p-0 py-4"
      />
      <Suspense fallback={<PageSkeleton />}>
        <AccountingControlForm />
      </Suspense>
    </AdminPageLayout>
  );
}
