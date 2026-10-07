import { useT, type TranslateFn } from "@trenova/shared/i18n/use-t";
import type { AuthEvent, RiskDecision } from "@trenova/shared/types/iam";
import { Tabs, TabsContent, TabsList, TabsTab } from "@trenova/shared/components/ui/tabs";
import { securityTabParser, type SecurityTabValue } from "@/hooks/use-organization-setting-state";
import { queries } from "@/lib/queries";
import { formatIdentityProviderName } from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import {
  ActivityIcon,
  type IconComponent,
  Key01Icon,
  ShieldTickIcon,
  Users01Icon,
} from "@trenova/shared/components/icons";
import { useQueryState } from "nuqs";
import { Activity, useCallback, useMemo } from "react";
import { ProvisioningTab } from "./provisioning-tab";
import { ActivityTab } from "./security-access/activity-tab";
import { OuterSection } from "./security-access/layout";
import { SecurityOverview } from "./security-access/overview";
import { PoliciesTab } from "./security-access/policies-tab";
import { SignInTab } from "./security-access/sign-in-tab";
import { identityProviderQueryKey } from "./security-access/utils";

const securityTabs: Array<{
  value: SecurityTabValue;
  label: string;
  Icon: IconComponent;
}> = [
  { value: "sign-in", label: "Sign-in", Icon: Key01Icon },
  { value: "provisioning", label: "Provisioning", Icon: Users01Icon },
  { value: "policies", label: "Policies", Icon: ShieldTickIcon },
  { value: "activity", label: "Activity", Icon: ActivityIcon },
];

export function SecurityAccessWorkspace({ organizationId }: { organizationId: string }) {
  return (
    <OuterSection>
      <SecurityOverviewSection organizationId={organizationId} />
      <SecurityAccessTabs organizationId={organizationId} />
    </OuterSection>
  );
}

function signInActivityLabel(t: TranslateFn, event: AuthEvent): string {
  const provider = formatIdentityProviderName(event.provider);
  switch (event.outcome) {
    case "success":
      return t("{0} sign-in success", provider);
    case "challenge":
      return t("{0} sign-in challenge", provider);
    case "denied":
      return t("{0} sign-in denied", provider);
    case "failed":
      return t("{0} sign-in failed", provider);
  }
}

function riskDecisionLabel(t: TranslateFn, outcome: RiskDecision["outcome"]): string {
  switch (outcome) {
    case "allow":
      return t("Risk decision: allow");
    case "challenge":
      return t("Risk decision: challenge");
    case "deny":
      return t("Risk decision: deny");
  }
}

function SecurityOverviewSection({ organizationId }: { organizationId: string }) {
  const t = useT();
  const providersQuery = useQuery({
    queryKey: [identityProviderQueryKey(organizationId)],
    queryFn: async () => apiService.organizationService.listIdentityProviders(organizationId),
  });
  const directoriesQuery = useQuery(queries.organization.scimDirectories(organizationId));
  const policiesQuery = useQuery(queries.organization.accessPolicies(organizationId));
  const authEventsQuery = useQuery(queries.organization.authEvents(organizationId));
  const riskQuery = useQuery(queries.organization.riskDecisions(organizationId));

  const providers = useMemo(() => providersQuery.data ?? [], [providersQuery.data]);
  const directories = useMemo(
    () => directoriesQuery.data?.results ?? [],
    [directoriesQuery.data?.results],
  );
  const policies = useMemo(() => policiesQuery.data ?? [], [policiesQuery.data]);
  const authEvents = useMemo(() => authEventsQuery.data ?? [], [authEventsQuery.data]);
  const riskDecisions = useMemo(() => riskQuery.data ?? [], [riskQuery.data]);

  const recentActivity = useMemo(
    () =>
      [
        ...authEvents.map((event) => ({
          id: event.id,
          label: signInActivityLabel(t, event),
          detail: event.ipAddress || event.errorCode || t("Authentication event"),
          status: event.riskOutcome,
          occurredAt: event.occurredAt,
        })),
        ...riskDecisions.map((decision) => ({
          id: decision.id,
          label: riskDecisionLabel(t, decision.outcome),
          detail: decision.reason || decision.signals.join(", ") || t("No additional signals"),
          status: decision.outcome,
          occurredAt: decision.createdAt,
        })),
      ]
        .sort((left, right) => right.occurredAt - left.occurredAt)
        .slice(0, 4),
    [authEvents, riskDecisions, t],
  );

  const overviewLoading =
    providersQuery.isLoading ||
    directoriesQuery.isLoading ||
    policiesQuery.isLoading ||
    authEventsQuery.isLoading ||
    riskQuery.isLoading;
  const enforcedProvider = providers.find((provider) => provider.enabled && provider.enforceSso);
  const activeDirectory = directories.find((directory) => directory.enabled);
  const activePolicyCount = policies.filter((policy) => policy.enabled).length;

  return (
    <SecurityOverview
      isLoading={overviewLoading}
      providerCount={providers.filter((provider) => provider.enabled).length}
      enforcedProviderName={
        enforcedProvider ? formatIdentityProviderName(enforcedProvider.name) : ""
      }
      directoryStatus={activeDirectory ? activeDirectory.tenantSlug : ""}
      activePolicyCount={activePolicyCount}
      recentActivity={recentActivity}
    />
  );
}

function SecurityAccessTabs({ organizationId }: { organizationId: string }) {
  const [securityTab, setSecurityTab] = useQueryState("securityTab", securityTabParser);
  const handleTabChange = useCallback(
    (value: string) => {
      void setSecurityTab(value as SecurityTabValue);
    },
    [setSecurityTab],
  );

  return (
    <Tabs value={securityTab} onValueChange={handleTabChange}>
      <TabsList variant="underline">
        {securityTabs.map(({ value, label, Icon }) => (
          <TabsTab key={value} value={value}>
            <Icon size={16} />
            {label}
          </TabsTab>
        ))}
      </TabsList>
      <TabsContent value="sign-in" keepMounted>
        <Activity mode={securityTab === "sign-in" ? "visible" : "hidden"}>
          <SignInTab organizationId={organizationId} />
        </Activity>
      </TabsContent>
      <TabsContent value="provisioning" keepMounted>
        <Activity mode={securityTab === "provisioning" ? "visible" : "hidden"}>
          <ProvisioningTab
            organizationId={organizationId}
            isActive={securityTab === "provisioning"}
          />
        </Activity>
      </TabsContent>
      <TabsContent value="policies" keepMounted>
        <Activity mode={securityTab === "policies" ? "visible" : "hidden"}>
          <PoliciesTab organizationId={organizationId} />
        </Activity>
      </TabsContent>
      <TabsContent value="activity" keepMounted>
        <Activity mode={securityTab === "activity" ? "visible" : "hidden"}>
          <ActivityTab organizationId={organizationId} />
        </Activity>
      </TabsContent>
    </Tabs>
  );
}
