import { SettingsPage } from "@/components/settings/settings-page";
import { organization } from "@/lib/queries/organization";
import type { RoutePrefetch } from "@/lib/route-prefetch";
import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import { lazy } from "react";
import { OrganizationSettingsSkeleton } from "./skeleton";

const OrganizationSettingsForm = lazy(() => import("./_components/organization-settings-form"));

// The form reads the current organization from the auth store the moment it mounts; the
// logo URL is resolved only once the form holds a logo value, so it is left to the form.
export const prefetch: RoutePrefetch = () => {
  const organizationId = useAuthStore.getState().user?.currentOrganizationId;
  return organizationId ? [organization.detail(organizationId)] : [];
};

export function OrganizationSettingsPage() {
  const t = useT();

  return (
    <SettingsPage
      title={t("Organization settings")}
      description={t("Your organization's profile, branding and how it operates")}
      skeleton={<OrganizationSettingsSkeleton />}
    >
      <OrganizationSettingsForm />
    </SettingsPage>
  );
}
