import { AdminPageLayout } from "@/components/navigation/sidebar-layout";
import { PageHeader } from "@/components/page-header";
import { queries } from "@/lib/queries";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { Suspense, lazy } from "react";

const OrganizationSettingsForm = lazy(() => import("./_components/organization-settings-form"));

// The form reads the current organization from the auth store the moment it mounts; the
// logo URL is resolved only once the form holds a logo value, so it is left to the form.
export const prefetch: RoutePrefetch = () => {
  const organizationId = useAuthStore.getState().user?.currentOrganizationId;
  return organizationId ? [queries.organization.detail(organizationId)] : [];
};

export function OrganizationSettingsPage() {
  return (
    <AdminPageLayout>
      <PageHeader
        title="Organization Settings"
        description="Manage your organization profile, compliance, and security settings"
      />
      <Suspense
        fallback={<div className="text-muted-foreground px-1 py-8 text-sm">Loading...</div>}
      >
        <OrganizationSettingsForm />
      </Suspense>
    </AdminPageLayout>
  );
}
