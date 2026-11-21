import { SettingsForm } from "@/components/settings/settings-form";
import { updateOrganizationSettingsGraphQL } from "@/lib/graphql/organization";
import { organization } from "@/lib/queries/organization";
import { userOrganization } from "@/lib/queries/user";
import { useSuspenseQuery } from "@tanstack/react-query";
import { useT } from "@trenova/shared/i18n/use-t";
import { useAuthStore } from "@trenova/shared/stores/auth-store";
import {
  organizationSettingsSchema,
  type OrganizationSettings,
} from "@trenova/shared/types/organization";
import {
  useOrganizationSettingsTabs,
  type OrganizationSettingsTab,
} from "../organization-settings-tabs";
import { BrandingTab } from "./branding-tab";
import { OperationsTab } from "./operations-tab";
import { ProfileTab } from "./profile-tab";

/**
 * Organization settings: the organization's own record, one form saved once, split into who
 * it is and where it is registered, how it looks on what it sends, and how it operates. Its
 * security and access live on a page of their own.
 */
export default function OrganizationSettingsForm() {
  const t = useT();
  const organizationId = useAuthStore((state) => state.user?.currentOrganizationId);

  if (!organizationId) {
    return (
      <div className="py-8 text-sm text-muted-foreground">
        {t("No active organization found for this session.")}
      </div>
    );
  }

  return <OrganizationSettingsTabs organizationId={organizationId} />;
}

/** The sidebar's name and mark and the organization switcher read the same record. */
const ORGANIZATION_READERS = [organization.branding._def, userOrganization.all._def] as const;

function OrganizationSettingsTabs({ organizationId }: { organizationId: string }) {
  const { data } = useSuspenseQuery(organization.detail(organizationId));
  const tabs = useOrganizationSettingsTabs();

  return (
    <SettingsForm<OrganizationSettings, OrganizationSettingsTab>
      values={data}
      schema={organizationSettingsSchema}
      queryKey={organization.detail(organizationId).queryKey}
      save={(values) => updateOrganizationSettingsGraphQL(organizationId, values)}
      refresh={ORGANIZATION_READERS}
      resourceName="Organization Settings"
      tabs={tabs}
      content={(tab) => {
        switch (tab) {
          case "profile":
            return <ProfileTab />;
          case "branding":
            return <BrandingTab organizationId={organizationId} />;
          case "operations":
            return <OperationsTab />;
        }
      }}
    />
  );
}
