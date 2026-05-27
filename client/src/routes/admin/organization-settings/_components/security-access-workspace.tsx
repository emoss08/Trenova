import { Tabs, TabsContent, TabsList, TabsTab } from "@/components/ui/tabs";
import { searchParamsParser, type SecurityTabValue } from "@/hooks/use-organization-setting-state";
import { queries } from "@/lib/queries";
import { formatIdentityProviderName } from "@/lib/utils";
import { apiService } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import {
  ActivityIcon,
  KeyRoundIcon,
  ShieldCheckIcon,
  type LucideIcon,
  UsersRoundIcon,
} from "lucide-react";
import { useQueryStates } from "nuqs";
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
  Icon: LucideIcon;
}> = [
  { value: "sign-in", label: "Sign-in", Icon: KeyRoundIcon },
  { value: "provisioning", label: "Provisioning", Icon: UsersRoundIcon },
  { value: "policies", label: "Policies", Icon: ShieldCheckIcon },
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

function SecurityOverviewSection({ organizationId }: { organizationId: string }) {
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
          label: `${formatIdentityProviderName(event.provider)} sign-in ${event.outcome}`,
          detail: event.ipAddress || event.errorCode || "Authentication event",
          status: event.riskOutcome,
          occurredAt: event.occurredAt,
        })),
        ...riskDecisions.map((decision) => ({
          id: decision.id,
          label: `Risk decision: ${decision.outcome}`,
          detail: decision.reason || decision.signals.join(", ") || "No additional signals",
          status: decision.outcome,
          occurredAt: decision.createdAt,
        })),
      ]
        .sort((left, right) => right.occurredAt - left.occurredAt)
        .slice(0, 4),
    [authEvents, riskDecisions],
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
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);
  const { securityTab } = searchParams;
  const handleTabChange = useCallback(
    (value: string) => {
      const nextSecurityTab = value as SecurityTabValue;

      void setSearchParams({
        securityTab: nextSecurityTab,
        directoryId: nextSecurityTab === "provisioning" ? searchParams.directoryId : null,
        activityView: nextSecurityTab === "activity" ? searchParams.activityView : null,
      });
    },
    [searchParams.activityView, searchParams.directoryId, setSearchParams],
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
          <ProvisioningTab organizationId={organizationId} />
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
