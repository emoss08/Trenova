import { useResetTableSearchParams } from "@/hooks/data-table/use-reset-table-search-params";
import { useT } from "@trenova/shared/i18n/use-t";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { useSecurityTab, securityTabValues, type SecurityTabValue } from "../security-access-state";
import {
  ActivityIcon,
  type IconComponent,
  Key01Icon,
  ShieldTickIcon,
  Shield01Icon,
  Users01Icon,
} from "@trenova/shared/components/icons";
import { Activity, useCallback, useMemo } from "react";
import { ProvisioningTab } from "./provisioning-tab";
import { ActivityTab } from "./security-access/activity-tab";
import { SecurityBriefing } from "./security-access/briefing/security-briefing";
import { PoliciesTab } from "./security-access/policies-tab";
import { SignInTab } from "./security-access/sign-in-tab";
import { firstAllowed, useSecurityAccess } from "./security-access/use-security-access";
import { defineLabels } from "@trenova/shared/i18n/labels";

const SECURITY_TAB_LABELS = defineLabels({
  overview: "Overview",
  "sign-in": "Sign-in",
  provisioning: "Provisioning",
  policies: "Policies",
  activity: "Activity",
} as const);

const SECURITY_TAB_ICONS: Record<SecurityTabValue, IconComponent> = {
  overview: Shield01Icon,
  "sign-in": Key01Icon,
  provisioning: Users01Icon,
  policies: ShieldTickIcon,
  activity: ActivityIcon,
};

/**
 * The security and access console: a briefing on the organization's posture, then how
 * people sign in, how they are provisioned, the policies they meet and what happened. Each
 * tab is shown to whoever may read it, and the open one lives in the address.
 */
export function SecurityAccessWorkspace({ organizationId }: { organizationId: string }) {
  const t = useT();
  const access = useSecurityAccess();
  const [requestedTab, setSecurityTab] = useSecurityTab();
  const securityTab = firstAllowed(requestedTab, securityTabValues, access.tabs);
  const visibleTabs = useMemo(
    () => securityTabValues.filter((value) => access.tabs[value]),
    [access.tabs],
  );
  const resetTable = useResetTableSearchParams();
  const openTab = useCallback(
    (value: SecurityTabValue) => {
      if (value === securityTab) {
        return;
      }
      resetTable();
      void setSecurityTab(value);
    },
    [resetTable, securityTab, setSecurityTab],
  );

  if (!securityTab) {
    return (
      <p className="py-8 text-sm text-muted-foreground">
        {t("You don't have access to any of this organization's security settings.")}
      </p>
    );
  }

  return (
    <Tabs
      value={securityTab}
      onValueChange={(value) => openTab(value as SecurityTabValue)}
      className="gap-4 pb-10"
    >
      <TabsList
        variant="underline"
        aria-label={t("Security sections")}
        className="max-w-full overflow-x-auto"
      >
        {visibleTabs.map((value) => {
          const Icon = SECURITY_TAB_ICONS[value];
          return (
            <TabsTab key={value} value={value} className="grow-0">
              <Icon size={16} />
              {SECURITY_TAB_LABELS[value]}
            </TabsTab>
          );
        })}
      </TabsList>
      {access.tabs.overview ? (
        <TabsContent value="overview">
          <Activity mode={securityTab === "overview" ? "visible" : "hidden"}>
            <SecurityBriefing organizationId={organizationId} />
          </Activity>
        </TabsContent>
      ) : null}
      {access.tabs["sign-in"] ? (
        <TabsContent value="sign-in" keepMounted>
          <Activity mode={securityTab === "sign-in" ? "visible" : "hidden"}>
            <SignInTab organizationId={organizationId} />
          </Activity>
        </TabsContent>
      ) : null}
      {access.tabs.provisioning ? (
        <TabsContent value="provisioning" keepMounted>
          <Activity mode={securityTab === "provisioning" ? "visible" : "hidden"}>
            <ProvisioningTab
              organizationId={organizationId}
              isActive={securityTab === "provisioning"}
            />
          </Activity>
        </TabsContent>
      ) : null}
      {access.tabs.policies ? (
        <TabsContent value="policies" keepMounted>
          <Activity mode={securityTab === "policies" ? "visible" : "hidden"}>
            <PoliciesTab organizationId={organizationId} />
          </Activity>
        </TabsContent>
      ) : null}
      {access.tabs.activity ? (
        <TabsContent value="activity" keepMounted>
          <Activity mode={securityTab === "activity" ? "visible" : "hidden"}>
            <ActivityTab />
          </Activity>
        </TabsContent>
      ) : null}
    </Tabs>
  );
}
