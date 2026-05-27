import { Tabs, TabsContent, TabsList, TabsTab } from "@/components/ui/tabs";
import { searchParamsParser } from "@/hooks/use-organization-setting-state";
import { queries } from "@/lib/queries";
import { formatIdentityProviderName } from "@/lib/utils";
import { apiService } from "@/services/api";
import { useQuery } from "@tanstack/react-query";
import {
  ActivityIcon,
  KeyRoundIcon,
  ShieldCheckIcon,
  UsersRoundIcon,
} from "lucide-react";
import { useQueryStates } from "nuqs";
import { Activity, useMemo } from "react";
import { ProvisioningTab } from "./provisioning-tab";
import { ActivityTab } from "./security-access/activity-tab";
import { SecurityOverview } from "./security-access/overview";
import { PoliciesTab } from "./security-access/policies-tab";
import { SignInTab } from "./security-access/sign-in-tab";
import { identityProviderQueryKey } from "./security-access/utils";

export function SecurityAccessWorkspace({ organizationId }: { organizationId: string }) {
  const [searchParams, setSearchParams] = useQueryStates(searchParamsParser);
  const providersQuery = useQuery({
    queryKey: [identityProviderQueryKey(organizationId)],
    queryFn: async () => apiService.organizationService.listIdentityProviders(organizationId),
  });
  const directoriesQuery = useQuery(queries.organization.scimDirectories(organizationId));
  const policiesQuery = useQuery(queries.organization.accessPolicies(organizationId));
  const authEventsQuery = useQuery(queries.organization.authEvents(organizationId));
  const riskQuery = useQuery(queries.organization.riskDecisions(organizationId));

  const providers = useMemo(() => providersQuery.data ?? [], [providersQuery.data]);
  const directories = useMemo(() => directoriesQuery.data ?? [], [directoriesQuery.data]);
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
    <section className="space-y-4 pb-10">
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

      <Tabs
        value={searchParams.securityTab}
        onValueChange={(value) => setSearchParams({ securityTab: value })}
        className="gap-4"
      >
        <TabsList variant="underline">
          <TabsTab value="sign-in">
            <KeyRoundIcon size={16} />
            Sign-in
          </TabsTab>
          <TabsTab value="provisioning">
            <UsersRoundIcon size={16} />
            Provisioning
          </TabsTab>
          <TabsTab value="policies">
            <ShieldCheckIcon size={16} />
            Policies
          </TabsTab>
          <TabsTab value="activity">
            <ActivityIcon size={16} />
            Activity
          </TabsTab>
        </TabsList>
        <TabsContent value="sign-in" keepMounted>
          <Activity mode={searchParams.securityTab === "sign-in" ? "visible" : "hidden"}>
            <SignInTab organizationId={organizationId} />
          </Activity>
        </TabsContent>
        <TabsContent value="provisioning" keepMounted>
          <Activity mode={searchParams.securityTab === "provisioning" ? "visible" : "hidden"}>
            <ProvisioningTab organizationId={organizationId} />
          </Activity>
        </TabsContent>
        <TabsContent value="policies" keepMounted>
          <Activity mode={searchParams.securityTab === "policies" ? "visible" : "hidden"}>
            <PoliciesTab organizationId={organizationId} />
          </Activity>
        </TabsContent>
        <TabsContent value="activity" keepMounted>
          <Activity mode={searchParams.securityTab === "activity" ? "visible" : "hidden"}>
            <ActivityTab organizationId={organizationId} />
          </Activity>
        </TabsContent>
      </Tabs>
    </section>
  );
}
