import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { Suspense, lazy } from "react";

const OrganizationSettingsForm = lazy(() => import("./_components/organization-settings-form"));

export function OrganizationSettingsPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Organization Settings"
        description="Manage your organization profile, compliance, and security settings"
      />
      <Suspense
        fallback={<div className="px-1 py-8 text-sm text-muted-foreground">Loading...</div>}
      >
        <OrganizationSettingsForm />
      </Suspense>
    </AdminPageLayout>
  );
}
