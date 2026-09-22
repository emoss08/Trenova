import { PageLayout } from "@/components/navigation/sidebar-layout";
import { queries } from "@/lib/queries";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { SuspenseLoader } from "@trenova/shared/components/component-loader";
import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { lazy } from "react";

const OrganizationSettingsForm = lazy(() => import("./_components/organization-settings-form"));

// The form reads the current organization from the auth store the moment it mounts; the
// logo URL is resolved only once the form holds a logo value, so it is left to the form.
export const prefetch: RoutePrefetch = () => {
  const organizationId = useAuthStore.getState().user?.currentOrganizationId;
  return organizationId ? [queries.organization.detail(organizationId)] : [];
};

export function OrganizationSettingsPage() {
  const t = useT();

  return (
    <PageLayout
      pageHeaderProps={{
        title: t("Organization settings"),
        description: t("Manage your organization profile, compliance, and security settings"),
      }}
      className="p-0"
    >
      <SuspenseLoader>
        <OrganizationSettingsForm />
      </SuspenseLoader>
    </PageLayout>
  );
}
