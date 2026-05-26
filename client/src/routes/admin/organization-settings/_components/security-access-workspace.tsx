import { RoleSelectAutocompleteField } from "@/components/autocomplete-fields";
import { FieldWrapper } from "@/components/fields/field-components";
import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { EntraLogo } from "@/components/logos/entra";
import { OktaLogo } from "@/components/logos/okta";
import { Badge, type BadgeVariant } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import { Switch } from "@/components/ui/switch";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTab } from "@/components/ui/tabs";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { formatUnixDateTime } from "@/lib/date";
import { queries } from "@/lib/queries";
import {
  getAvailableOperations,
  getAvailableResources,
  type OperationDefinition,
  type ResourceDefinition,
} from "@/lib/role-api";
import { cn, toTitleCase } from "@/lib/utils";
import { apiService } from "@/services/api";
import type {
  AccessPolicy,
  AuthEvent,
  ExternalIdentity,
  IdentityProvider,
  IdentityProviderFormValues,
  MFAAuthenticator,
  ProvisioningAuditRecord,
  RiskDecision,
  SCIMDirectory,
  SCIMGroupRoleMapping,
  SCIMToken,
} from "@/types/iam";
import { identityProviderCreateFormSchema, identityProviderFormSchema } from "@/types/iam";
import type { API_ENDPOINTS } from "@/types/server";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ActivityIcon,
  AlertTriangleIcon,
  CheckIcon,
  ClipboardIcon,
  ExternalLinkIcon,
  KeyRoundIcon,
  LockKeyholeIcon,
  PlusIcon,
  SaveIcon,
  SearchIcon,
  ShieldAlertIcon,
  ShieldCheckIcon,
  Trash2Icon,
  UsersRoundIcon,
  XIcon,
} from "lucide-react";
import { type ReactNode, useEffect, useMemo, useState } from "react";
import { Controller, type Resolver, useForm, useFormContext } from "react-hook-form";
import { toast } from "sonner";

type SecurityTab = "sign-in" | "provisioning" | "policies" | "activity";
type ActivityView = "auth" | "risk" | "identities" | "mfa";
type IdentityProviderPanelMode = "create" | "edit";
type IdentityProviderRecord = IdentityProvider & Record<string, unknown>;
type ConditionRow = { id: string; key: string; value: string };
type MappingFormValues = {
  externalGroupId: string;
  displayName: string;
  roleId: string;
};

const emptyProvider: IdentityProvider = {
  id: "",
  organizationId: "",
  businessUnitId: "",
  name: "",
  slug: "",
  protocol: "OIDC",
  enabled: true,
  enforceSso: false,
  autoProvision: false,
  allowFederatedMfa: true,
  allowedDomains: [],
  attributeMap: { email: "email" },
  oidcIssuerUrl: "",
  oidcClientId: "",
  oidcClientSecret: "",
  oidcRedirectUrl: "",
  oidcScopes: ["openid", "email", "profile"],
  version: 0,
  createdAt: 0,
  updatedAt: 0,
};

const emptyDirectory: SCIMDirectory = {
  id: "",
  organizationId: "",
  businessUnitId: "",
  tenantSlug: "",
  enabled: true,
  createdAt: 0,
  updatedAt: 0,
};

const emptyMapping: SCIMGroupRoleMapping = {
  id: "",
  organizationId: "",
  businessUnitId: "",
  directoryId: "",
  externalGroupId: "",
  displayName: "",
  roleId: "",
  createdAt: 0,
  updatedAt: 0,
};

const emptyPolicy: AccessPolicy = {
  id: "",
  organizationId: "",
  businessUnitId: "",
  name: "",
  resource: "",
  operation: "read",
  effect: "deny",
  priority: 100,
  conditions: {},
  enabled: true,
  createdAt: 0,
  updatedAt: 0,
};

const providerPresets = [
  {
    label: "Entra ID",
    name: "Microsoft Entra ID",
    slug: "entra-id",
    issuer: "https://login.microsoftonline.com/{tenant-id}/v2.0",
    scopes: ["openid", "email", "profile"],
  },
  {
    label: "Okta",
    name: "Okta",
    slug: "okta",
    issuer: "https://{yourOktaDomain}/oauth2/default",
    scopes: ["openid", "email", "profile"],
  },
];

function identityProviderQueryKey(organizationId: string) {
  return `identity-provider-list:${organizationId}`;
}

function identityProviderEndpoint(organizationId: string) {
  return `/organizations/${organizationId}/iam/identity-providers/` as API_ENDPOINTS;
}

export function SecurityAccessWorkspace({ organizationId }: { organizationId: string }) {
  const [tab, setTab] = useState<SecurityTab>("sign-in");
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
          label: `${formatProviderName(event.provider)} sign-in ${event.outcome}`,
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
      <SecurityHeader />
      <SecurityOverview
        isLoading={overviewLoading}
        providerCount={providers.filter((provider) => provider.enabled).length}
        enforcedProviderName={enforcedProvider ? formatProviderName(enforcedProvider.name) : ""}
        directoryStatus={activeDirectory ? activeDirectory.tenantSlug : ""}
        activePolicyCount={activePolicyCount}
        recentActivity={recentActivity}
      />

      <Tabs value={tab} onValueChange={(value) => setTab(value as SecurityTab)} className="gap-4">
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
        <TabsContent value="sign-in">
          <SignInTab organizationId={organizationId} />
        </TabsContent>
        <TabsContent value="provisioning">
          <ProvisioningTab organizationId={organizationId} />
        </TabsContent>
        <TabsContent value="policies">
          <PoliciesTab organizationId={organizationId} />
        </TabsContent>
        <TabsContent value="activity">
          <ActivityTab organizationId={organizationId} />
        </TabsContent>
      </Tabs>
    </section>
  );
}

function SecurityHeader() {
  return (
    <div className="flex flex-col gap-3 border-b pb-4 lg:flex-row lg:items-end lg:justify-between">
      <div className="space-y-1">
        <div className="flex items-center gap-2">
          <span className="flex size-8 items-center justify-center rounded-md border bg-muted/60">
            <ShieldCheckIcon className="size-4 text-primary" />
          </span>
          <h2 className="text-lg font-semibold tracking-tight">Security & Access</h2>
        </div>
        <p className="max-w-3xl text-sm text-muted-foreground">
          Manage federated sign-in, SCIM provisioning, access policies, and security activity.
        </p>
      </div>
    </div>
  );
}

function SecurityOverview({
  isLoading,
  providerCount,
  enforcedProviderName,
  directoryStatus,
  activePolicyCount,
  recentActivity,
}: {
  isLoading: boolean;
  providerCount: number;
  enforcedProviderName: string;
  directoryStatus: string;
  activePolicyCount: number;
  recentActivity: Array<{
    id: string;
    label: string;
    detail: string;
    status: string;
    occurredAt: number;
  }>;
}) {
  if (isLoading) {
    return <OverviewSkeleton />;
  }

  return (
    <div className="grid gap-3 xl:grid-cols-[minmax(0,1fr)_360px]">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <StatusTile
          icon={<KeyRoundIcon />}
          label="Providers"
          value={String(providerCount)}
          detail={providerCount === 1 ? "Enabled provider" : "Enabled providers"}
          tone={providerCount > 0 ? "active" : "muted"}
        />
        <StatusTile
          icon={<LockKeyholeIcon />}
          label="SSO enforcement"
          value={enforcedProviderName || "Optional"}
          detail={
            enforcedProviderName ? "Password fallback restricted" : "Password sign-in allowed"
          }
          tone={enforcedProviderName ? "warning" : "muted"}
        />
        <StatusTile
          icon={<UsersRoundIcon />}
          label="SCIM directory"
          value={directoryStatus || "Not connected"}
          detail={directoryStatus ? "Provisioning enabled" : "Directory sync inactive"}
          tone={directoryStatus ? "active" : "muted"}
        />
        <StatusTile
          icon={<ShieldCheckIcon />}
          label="Active policies"
          value={String(activePolicyCount)}
          detail={activePolicyCount === 1 ? "Policy evaluating" : "Policies evaluating"}
          tone={activePolicyCount > 0 ? "info" : "muted"}
        />
      </div>
      <div className="rounded-lg border bg-muted/20">
        <div className="flex items-center justify-between border-b px-3 py-2">
          <div>
            <div className="text-sm font-medium">Recent security activity</div>
            <div className="text-xs text-muted-foreground">
              Latest authentication and risk signals
            </div>
          </div>
          <ActivityIcon className="size-4 text-muted-foreground" />
        </div>
        <div className="divide-y">
          {recentActivity.length > 0 ? (
            recentActivity.map((activity) => (
              <ActivityItem
                key={activity.id}
                title={activity.label}
                detail={activity.detail}
                badge={activity.status}
                when={formatTimestamp(activity.occurredAt)}
              />
            ))
          ) : (
            <EmptyState
              icon={<ActivityIcon />}
              label="No activity yet"
              description="Sign-in and risk events will appear here after users authenticate."
              compact
            />
          )}
        </div>
      </div>
    </div>
  );
}

function StatusTile({
  icon,
  label,
  value,
  detail,
  tone,
}: {
  icon: ReactNode;
  label: string;
  value: string;
  detail: string;
  tone: "active" | "warning" | "info" | "muted";
}) {
  return (
    <div className="rounded-lg border bg-background p-3 shadow-xs">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-1">
          <div className="text-xs font-medium text-muted-foreground uppercase">{label}</div>
          <div className="truncate text-lg font-semibold tracking-tight">{value}</div>
          <div className="truncate text-xs text-muted-foreground">{detail}</div>
        </div>
        <span
          className={cn(
            "flex size-8 shrink-0 items-center justify-center rounded-md border [&_svg]:size-4",
            tone === "active" && "border-green-600/30 bg-green-600/10 text-green-700",
            tone === "warning" && "border-yellow-600/30 bg-yellow-600/10 text-yellow-700",
            tone === "info" && "border-blue-600/30 bg-blue-600/10 text-blue-700",
            tone === "muted" && "bg-muted text-muted-foreground",
          )}
        >
          {icon}
        </span>
      </div>
    </div>
  );
}

function SignInTab({ organizationId }: { organizationId: string }) {
  const queryClient = useQueryClient();
  const providersQuery = useQuery({
    queryKey: [identityProviderQueryKey(organizationId)],
    queryFn: async () => apiService.organizationService.listIdentityProviders(organizationId),
  });
  const createForm = useForm<IdentityProviderFormValues>({
    resolver: zodResolver(identityProviderCreateFormSchema) as Resolver<IdentityProviderFormValues>,
    defaultValues: emptyProvider,
    mode: "onChange",
  });
  const editForm = useForm<IdentityProviderFormValues>({
    resolver: zodResolver(identityProviderFormSchema) as Resolver<IdentityProviderFormValues>,
    defaultValues: emptyProvider,
    mode: "onChange",
  });
  const [panelMode, setPanelMode] = useState<IdentityProviderPanelMode>("create");
  const [panelOpen, setPanelOpen] = useState(false);
  const [editingProvider, setEditingProvider] = useState<IdentityProvider | null>(null);
  const [search, setSearch] = useState("");

  const providers = useMemo(() => providersQuery.data ?? [], [providersQuery.data]);
  const filteredProviders = useMemo(() => {
    const query = search.trim().toLowerCase();
    if (!query) return providers;
    return providers.filter((provider) =>
      [
        formatProviderName(provider.name),
        provider.slug,
        provider.oidcIssuerUrl,
        provider.oidcRedirectUrl,
        ...provider.allowedDomains,
      ]
        .join(" ")
        .toLowerCase()
        .includes(query),
    );
  }, [providers, search]);

  const deleteMutation = useMutation({
    mutationFn: async (providerId: string) =>
      apiService.organizationService.deleteIdentityProvider(organizationId, providerId),
    onSuccess: async () => {
      toast.success("Identity provider removed");
      await queryClient.invalidateQueries({
        queryKey: [identityProviderQueryKey(organizationId)],
      });
    },
  });

  const openCreatePanel = () => {
    setPanelMode("create");
    setEditingProvider(null);
    setPanelOpen(true);
  };

  const openEditPanel = (provider: IdentityProvider) => {
    setPanelMode("edit");
    setEditingProvider(provider);
    setPanelOpen(true);
  };

  return (
    <div className="space-y-3">
      <ConsoleToolbar
        title="Identity providers"
        description="OIDC sign-in providers available to this organization."
        search={search}
        onSearchChange={setSearch}
        searchPlaceholder="Search providers, domains, or issuer"
        action={
          <Button onClick={openCreatePanel}>
            <PlusIcon />
            Add provider
          </Button>
        }
      />
      {providersQuery.isLoading ? (
        <RowSkeleton rows={3} />
      ) : providersQuery.isError ? (
        <ErrorState label="Identity providers could not be loaded." />
      ) : filteredProviders.length > 0 ? (
        <div className="overflow-hidden rounded-lg border bg-background">
          {filteredProviders.map((provider) => (
            <ProviderRow
              key={provider.id}
              provider={provider}
              onEdit={() => openEditPanel(provider)}
              onDelete={() => deleteMutation.mutate(provider.id)}
              isDeleting={deleteMutation.isPending}
            />
          ))}
        </div>
      ) : (
        <EmptyState
          icon={<KeyRoundIcon />}
          label={providers.length === 0 ? "No identity providers configured" : "No providers found"}
          description={
            providers.length === 0
              ? "Add Entra ID or Okta to enable federated sign-in for users."
              : "Adjust the search filter to find a configured provider."
          }
        />
      )}

      {panelMode === "edit" ? (
        <FormEditPanel<IdentityProviderFormValues, IdentityProviderRecord>
          open={panelOpen}
          onOpenChange={setPanelOpen}
          row={(editingProvider as IdentityProviderRecord | null) ?? null}
          form={editForm}
          url={identityProviderEndpoint(organizationId)}
          queryKey={identityProviderQueryKey(organizationId)}
          title="Identity Provider"
          fieldKey="name"
          size="lg"
          formComponent={<IdentityProviderForm mode="edit" />}
          mutationFn={async (values) =>
            apiService.organizationService.updateIdentityProvider(
              organizationId,
              toIdentityProvider(values),
            )
          }
        />
      ) : (
        <FormCreatePanel<IdentityProviderFormValues, IdentityProviderRecord>
          open={panelOpen}
          onOpenChange={setPanelOpen}
          form={createForm}
          url={identityProviderEndpoint(organizationId)}
          queryKey={identityProviderQueryKey(organizationId)}
          title="Identity Provider"
          description="Configure OIDC sign-in details, allowed domains, scopes, and enforcement settings."
          size="lg"
          formComponent={<IdentityProviderForm mode="create" />}
          mutationFn={async (values) =>
            apiService.organizationService.createIdentityProvider(
              organizationId,
              toIdentityProvider(values),
            )
          }
        />
      )}
    </div>
  );
}

function ProviderRow({
  provider,
  onEdit,
  onDelete,
  isDeleting,
}: {
  provider: IdentityProvider;
  onEdit: () => void;
  onDelete: () => void;
  isDeleting: boolean;
}) {
  return (
    <div className="grid gap-3 border-b p-3 last:border-b-0 lg:grid-cols-[minmax(0,1fr)_180px] lg:items-center">
      <div className="min-w-0 space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <span className="flex size-9 items-center justify-center rounded-md border bg-muted/30">
            <ProviderLogo name={provider.name} />
          </span>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-medium">
                {formatProviderName(provider.name) || "OIDC provider"}
              </span>
              <Badge variant={provider.enabled ? "active" : "inactive"}>
                {provider.enabled ? "Enabled" : "Disabled"}
              </Badge>
              {provider.enforceSso && <Badge variant="warning">SSO enforced</Badge>}
              {provider.autoProvision && <Badge variant="info">Auto-provision</Badge>}
            </div>
            <div className="truncate text-xs text-muted-foreground">
              {provider.slug || "No slug"}
            </div>
          </div>
        </div>
        <div className="grid gap-2 text-xs text-muted-foreground md:grid-cols-2">
          <MetaLine label="Issuer" value={provider.oidcIssuerUrl || "-"} />
          <MetaLine label="Redirect URI" value={provider.oidcRedirectUrl || "-"} />
          <MetaLine label="Domains" value={provider.allowedDomains.join(", ") || "Any domain"} />
          <MetaLine label="Scopes" value={provider.oidcScopes.join(" ") || "Default OIDC scopes"} />
        </div>
      </div>
      <div className="flex justify-end gap-2">
        <Button variant="outline" onClick={onEdit}>
          Edit
        </Button>
        <Button variant="destructive" onClick={onDelete} disabled={isDeleting}>
          <Trash2Icon />
          Delete
        </Button>
      </div>
    </div>
  );
}

function IdentityProviderForm({ mode }: { mode: IdentityProviderPanelMode }) {
  const { control, getValues, setValue } = useFormContext<IdentityProviderFormValues>();

  const applyPreset = (presetSlug: string) => {
    const preset = providerPresets.find((item) => item.slug === presetSlug);
    if (!preset) return;

    const values = getValues();
    if (!values.name) setValue("name", preset.name, { shouldDirty: true, shouldValidate: true });
    if (!values.slug) setValue("slug", preset.slug, { shouldDirty: true, shouldValidate: true });
    if (!values.oidcIssuerUrl) {
      setValue("oidcIssuerUrl", preset.issuer, { shouldDirty: true, shouldValidate: true });
    }
    if (values.oidcScopes.length === 0) {
      setValue("oidcScopes", preset.scopes, { shouldDirty: true, shouldValidate: true });
    }
  };

  return (
    <div className="space-y-5">
      <FormSection
        title="Quick start"
        description="Start with common provider defaults, then adjust tenant-specific values."
      >
        <div className="grid gap-2 sm:grid-cols-2">
          {providerPresets.map((preset) => (
            <button
              key={preset.slug}
              type="button"
              className="flex items-center gap-2 rounded-md border bg-background px-3 py-2 text-left text-sm transition-colors hover:bg-muted/50"
              onClick={() => applyPreset(preset.slug)}
            >
              <ProviderLogo name={preset.name} />
              <span className="font-medium">{preset.label}</span>
            </button>
          ))}
        </div>
      </FormSection>

      <FormSection
        title="Provider"
        description="Name the sign-in provider and configure the identifier used by hosted login flows."
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label="Name"
              placeholder="Microsoft Entra ID"
              description="Displayed to administrators and users in sign-in flows."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="slug"
              label="Slug"
              placeholder="entra-id"
              description="Stable URL-safe provider key used internally for routing."
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="OIDC application"
        description="Copy these values from the provider application registration."
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="oidcIssuerUrl"
              label="Issuer URL"
              placeholder="https://login.microsoftonline.com/{tenant-id}/v2.0"
              description="OIDC issuer metadata URL for token validation."
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="oidcRedirectUrl"
              label="Redirect URI"
              placeholder="https://app.example.com/auth/callback"
              description="Callback URI registered in the provider application."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="oidcClientId"
              label="Client ID"
              placeholder="Application client ID"
              description="Public OIDC client identifier."
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={mode === "create" ? { required: true } : undefined}
              name="oidcClientSecret"
              label="Client secret"
              type="password"
              placeholder={mode === "edit" ? "Leave blank to keep current secret" : "Client secret"}
              description={
                mode === "edit"
                  ? "Only enter a value when rotating the provider secret."
                  : "Required secret from the provider application registration."
              }
            />
          </FormControl>
          <FormControl cols="full">
            <ChipArrayField
              control={control}
              name="oidcScopes"
              label="OIDC scopes"
              placeholder="openid email profile"
              description="Space-separated scopes requested during authentication."
              parseValue={parseWords}
              formatValue={(value) => value.join(" ")}
              required
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title="Access boundaries"
        description="Limit domains and tune how federated users are handled after sign-in."
      >
        <FormGroup cols={1}>
          <FormControl>
            <ChipArrayField
              control={control}
              name="allowedDomains"
              label="Allowed domains"
              placeholder="example.com, subsidiary.com"
              description="Comma-separated domains allowed to authenticate. Leave blank to allow any domain."
              parseValue={parseCSV}
              formatValue={(value) => value.join(", ")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="enabled"
              label="Enabled"
              description="Allow this provider to appear in sign-in flows."
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="enforceSso"
              label="Enforce SSO"
              description="Require users to authenticate with a federated provider."
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="autoProvision"
              label="Auto-provision users"
              description="Create user records after successful provider authentication."
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowFederatedMfa"
              label="Trust federated MFA"
              description="Accept MFA claims from the provider when risk policy allows it."
              outlined
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}

function ProvisioningTab({ organizationId }: { organizationId: string }) {
  const queryClient = useQueryClient();
  const directoriesQuery = useQuery(queries.organization.scimDirectories(organizationId));
  const auditQuery = useQuery(queries.organization.provisioningAudit(organizationId));
  const [directory, setDirectory] = useState<SCIMDirectory>(emptyDirectory);
  const [directorySheetOpen, setDirectorySheetOpen] = useState(false);
  const [selectedDirectoryId, setSelectedDirectoryId] = useState("");
  const [tokenName, setTokenName] = useState("");
  const [createdToken, setCreatedToken] = useState("");
  const [mapping, setMapping] = useState<SCIMGroupRoleMapping>(emptyMapping);
  const [mappingSheetOpen, setMappingSheetOpen] = useState(false);

  const directories = useMemo(() => directoriesQuery.data ?? [], [directoriesQuery.data]);
  const directoryId = selectedDirectoryId || directories[0]?.id || "";
  const selectedDirectory = directories.find((item) => item.id === directoryId);
  const tokensQuery = useQuery({
    queryKey: ["scim-tokens", organizationId, directoryId],
    queryFn: async () => apiService.organizationService.listSCIMTokens(organizationId, directoryId),
    enabled: Boolean(directoryId),
  });
  const mappingsQuery = useQuery({
    queryKey: ["scim-mappings", organizationId, directoryId],
    queryFn: async () =>
      apiService.organizationService.listSCIMGroupRoleMappings(organizationId, directoryId),
    enabled: Boolean(directoryId),
  });

  useEffect(() => {
    if (!selectedDirectoryId && directories[0]?.id) {
      setSelectedDirectoryId(directories[0].id);
    }
  }, [directories, selectedDirectoryId]);

  const invalidateProvisioning = async () => {
    await queryClient.invalidateQueries({
      queryKey: queries.organization.scimDirectories(organizationId).queryKey,
    });
    await queryClient.invalidateQueries({ queryKey: ["scim-tokens", organizationId, directoryId] });
    await queryClient.invalidateQueries({
      queryKey: ["scim-mappings", organizationId, directoryId],
    });
    await queryClient.invalidateQueries({
      queryKey: queries.organization.provisioningAudit(organizationId).queryKey,
    });
  };

  const saveDirectoryMutation = useMutation({
    mutationFn: async (value: SCIMDirectory) =>
      value.id
        ? apiService.organizationService.updateSCIMDirectory(organizationId, value)
        : apiService.organizationService.createSCIMDirectory(organizationId, value),
    onSuccess: async (saved) => {
      toast.success("SCIM directory saved");
      setDirectory(emptyDirectory);
      setDirectorySheetOpen(false);
      setSelectedDirectoryId(saved.id);
      await invalidateProvisioning();
    },
  });

  const createTokenMutation = useMutation({
    mutationFn: async () =>
      apiService.organizationService.createSCIMToken(organizationId, directoryId, tokenName),
    onSuccess: async (response) => {
      setCreatedToken(response.token);
      setTokenName("");
      toast.success("SCIM token created");
      await invalidateProvisioning();
    },
  });

  const revokeTokenMutation = useMutation({
    mutationFn: async (tokenId: string) =>
      apiService.organizationService.revokeSCIMToken(organizationId, tokenId),
    onSuccess: async () => {
      toast.success("SCIM token revoked");
      await invalidateProvisioning();
    },
  });

  const saveMappingMutation = useMutation({
    mutationFn: async (value: SCIMGroupRoleMapping) =>
      value.id
        ? apiService.organizationService.updateSCIMGroupRoleMapping(
            organizationId,
            directoryId,
            value,
          )
        : apiService.organizationService.createSCIMGroupRoleMapping(
            organizationId,
            directoryId,
            value,
          ),
    onSuccess: async () => {
      toast.success("Group mapping saved");
      setMapping(emptyMapping);
      setMappingSheetOpen(false);
      await invalidateProvisioning();
    },
  });

  const deleteMappingMutation = useMutation({
    mutationFn: async (mappingId: string) =>
      apiService.organizationService.deleteSCIMGroupRoleMapping(
        organizationId,
        directoryId,
        mappingId,
      ),
    onSuccess: async () => {
      toast.success("Group mapping removed");
      await invalidateProvisioning();
    },
  });

  const openDirectorySheet = (value: SCIMDirectory) => {
    setDirectory(value);
    setDirectorySheetOpen(true);
  };

  const openMappingSheet = (value: SCIMGroupRoleMapping) => {
    setMapping(value);
    setMappingSheetOpen(true);
  };

  return (
    <div className="grid gap-3 xl:grid-cols-[320px_minmax(0,1fr)]">
      <div className="space-y-3">
        <div className="rounded-lg border bg-background">
          <div className="flex items-center justify-between border-b p-3">
            <div>
              <div className="text-sm font-medium">SCIM directories</div>
              <div className="text-xs text-muted-foreground">Directory sync tenants</div>
            </div>
            <Button size="sm" onClick={() => openDirectorySheet(emptyDirectory)}>
              <PlusIcon />
              Add
            </Button>
          </div>
          {directoriesQuery.isLoading ? (
            <div className="space-y-2 p-3">
              <Skeleton className="h-14 w-full" />
              <Skeleton className="h-14 w-full" />
            </div>
          ) : directoriesQuery.isError ? (
            <ErrorState label="SCIM directories could not be loaded." compact />
          ) : directories.length > 0 ? (
            <div className="divide-y">
              {directories.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  className={cn(
                    "flex w-full items-center justify-between gap-3 px-3 py-3 text-left transition-colors hover:bg-muted/40",
                    item.id === directoryId && "bg-muted/60",
                  )}
                  onClick={() => setSelectedDirectoryId(item.id)}
                >
                  <div className="min-w-0">
                    <div className="truncate text-sm font-medium">{item.tenantSlug}</div>
                    <div className="text-xs text-muted-foreground">
                      Updated {formatTimestamp(item.updatedAt || item.createdAt)}
                    </div>
                  </div>
                  <Badge variant={item.enabled ? "active" : "inactive"}>
                    {item.enabled ? "Enabled" : "Disabled"}
                  </Badge>
                </button>
              ))}
            </div>
          ) : (
            <EmptyState
              icon={<UsersRoundIcon />}
              label="No directories"
              description="Create a SCIM directory before issuing tokens or mapping groups."
              compact
            />
          )}
        </div>
      </div>

      <div className="min-w-0 space-y-3">
        <div className="rounded-lg border bg-background p-3">
          <div className="flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
            <div className="min-w-0">
              <div className="flex flex-wrap items-center gap-2">
                <h3 className="truncate text-base font-semibold tracking-tight">
                  {selectedDirectory?.tenantSlug || "Select a directory"}
                </h3>
                {selectedDirectory && (
                  <Badge variant={selectedDirectory.enabled ? "active" : "inactive"}>
                    {selectedDirectory.enabled ? "Enabled" : "Disabled"}
                  </Badge>
                )}
              </div>
              <p className="text-sm text-muted-foreground">
                Manage SCIM tokens, group-to-role mappings, and provisioning audit events.
              </p>
            </div>
            <Button
              variant="outline"
              disabled={!selectedDirectory}
              onClick={() => selectedDirectory && openDirectorySheet(selectedDirectory)}
            >
              Edit directory
            </Button>
          </div>
        </div>

        <SCIMTokenPanel
          tokens={tokensQuery.data ?? []}
          isLoading={tokensQuery.isLoading}
          tokenName={tokenName}
          createdToken={createdToken}
          setTokenName={setTokenName}
          onCreate={() => createTokenMutation.mutate()}
          onRevoke={(tokenId) => revokeTokenMutation.mutate(tokenId)}
          disabled={!directoryId || createTokenMutation.isPending}
        />
        <MappingsPanel
          mappings={mappingsQuery.data ?? []}
          isLoading={mappingsQuery.isLoading}
          onAdd={() => openMappingSheet(emptyMapping)}
          onEdit={openMappingSheet}
          onDelete={(mappingId) => deleteMappingMutation.mutate(mappingId)}
          disabled={!directoryId}
        />
        <AuditTimeline records={auditQuery.data ?? []} isLoading={auditQuery.isLoading} />
      </div>

      <Sheet open={directorySheetOpen} onOpenChange={setDirectorySheetOpen}>
        <SheetContent className="w-[calc(100vw-1rem)] sm:max-w-md">
          <SheetHeader className="border-b">
            <SheetTitle>{directory.id ? "Edit SCIM directory" : "Add SCIM directory"}</SheetTitle>
            <SheetDescription>
              Configure the tenant slug and provisioning availability.
            </SheetDescription>
          </SheetHeader>
          <div className="space-y-3 px-4">
            <Field label="Tenant slug">
              <Input
                value={directory.tenantSlug}
                onChange={(event) => setDirectory({ ...directory, tenantSlug: event.target.value })}
              />
            </Field>
            <ToggleRow
              label="Enabled"
              description="Allow SCIM API calls for this directory."
              checked={directory.enabled}
              onCheckedChange={(enabled) => setDirectory({ ...directory, enabled })}
            />
          </div>
          <SheetFooter className="border-t">
            <Button
              onClick={() => saveDirectoryMutation.mutate(directory)}
              isLoading={saveDirectoryMutation.isPending}
              loadingText="Saving..."
            >
              <SaveIcon />
              Save directory
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>

      <Sheet open={mappingSheetOpen} onOpenChange={setMappingSheetOpen}>
        <SheetContent className="w-[calc(100vw-1rem)] sm:max-w-md">
          <SheetHeader className="border-b">
            <SheetTitle>{mapping.id ? "Edit group mapping" : "Add group mapping"}</SheetTitle>
            <SheetDescription>Map an external SCIM group to a Trenova role.</SheetDescription>
          </SheetHeader>
          <MappingEditor
            mapping={mapping}
            disabled={!directoryId || saveMappingMutation.isPending}
            onSave={(value) => saveMappingMutation.mutate(value)}
          />
        </SheetContent>
      </Sheet>
    </div>
  );
}

function SCIMTokenPanel({
  tokens,
  isLoading,
  tokenName,
  createdToken,
  setTokenName,
  onCreate,
  onRevoke,
  disabled,
}: {
  tokens: SCIMToken[];
  isLoading: boolean;
  tokenName: string;
  createdToken: string;
  setTokenName: (value: string) => void;
  onCreate: () => void;
  onRevoke: (tokenId: string) => void;
  disabled: boolean;
}) {
  return (
    <div className="rounded-lg border bg-background">
      <PanelHeader
        icon={<KeyRoundIcon />}
        title="SCIM tokens"
        description="Issue bearer tokens for directory synchronization."
      />
      <div className="space-y-3 p-3">
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input
            value={tokenName}
            placeholder="Token name"
            onChange={(event) => setTokenName(event.target.value)}
          />
          <Button onClick={onCreate} disabled={disabled || tokenName.trim() === ""}>
            <PlusIcon />
            Create token
          </Button>
        </div>
        {createdToken && <CopyableSecretBlock value={createdToken} />}
        {isLoading ? (
          <RowSkeleton rows={2} />
        ) : tokens.length > 0 ? (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Prefix</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Last used</TableHead>
                  <TableHead className="w-28">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {tokens.map((token) => (
                  <TableRow key={token.id}>
                    <TableCell className="font-medium">{token.name}</TableCell>
                    <TableCell>
                      <code className="rounded bg-muted px-1.5 py-0.5 text-xs">{token.prefix}</code>
                    </TableCell>
                    <TableCell>
                      <Badge variant={token.status === "active" ? "active" : "inactive"}>
                        {toTitleCase(token.status)}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-muted-foreground">
                      {token.lastUsedAt ? formatTimestamp(token.lastUsedAt) : "Never"}
                    </TableCell>
                    <TableCell>
                      <Button
                        size="sm"
                        variant="destructive"
                        disabled={token.status !== "active"}
                        onClick={() => onRevoke(token.id)}
                      >
                        Revoke
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <EmptyState
            icon={<KeyRoundIcon />}
            label="No SCIM tokens"
            description="Create a token and copy it into your directory sync application."
            compact
          />
        )}
      </div>
    </div>
  );
}

function CopyableSecretBlock({ value }: { value: string }) {
  const { copy, isCopied } = useCopyToClipboard();

  return (
    <div className="rounded-lg border border-amber-600/30 bg-amber-600/10 p-3">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <div>
          <div className="text-sm font-medium text-amber-800 dark:text-amber-300">
            Copy this token now
          </div>
          <div className="text-xs text-amber-700/80 dark:text-amber-300/80">
            The plaintext token is only shown once.
          </div>
        </div>
        <Button size="sm" variant="outline" onClick={() => void copy(value, { withToast: true })}>
          {isCopied ? <CheckIcon /> : <ClipboardIcon />}
          {isCopied ? "Copied" : "Copy"}
        </Button>
      </div>
      <code className="block rounded-md border bg-background/80 p-2 font-mono text-xs break-all">
        {value}
      </code>
    </div>
  );
}

function MappingsPanel({
  mappings,
  isLoading,
  onAdd,
  onEdit,
  onDelete,
  disabled,
}: {
  mappings: SCIMGroupRoleMapping[];
  isLoading: boolean;
  onAdd: () => void;
  onEdit: (mapping: SCIMGroupRoleMapping) => void;
  onDelete: (mappingId: string) => void;
  disabled: boolean;
}) {
  return (
    <div className="rounded-lg border bg-background">
      <PanelHeader
        icon={<UsersRoundIcon />}
        title="Group role mappings"
        description="Resolve external directory groups into application roles."
        action={
          <Button size="sm" onClick={onAdd} disabled={disabled}>
            <PlusIcon />
            Add mapping
          </Button>
        }
      />
      <div className="p-3">
        {isLoading ? (
          <RowSkeleton rows={2} />
        ) : mappings.length > 0 ? (
          <div className="overflow-x-auto">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>External group</TableHead>
                  <TableHead>Display name</TableHead>
                  <TableHead>Role</TableHead>
                  <TableHead className="w-36">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {mappings.map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>
                      <code className="rounded bg-muted px-1.5 py-0.5 text-xs">
                        {item.externalGroupId}
                      </code>
                    </TableCell>
                    <TableCell>{item.displayName || "-"}</TableCell>
                    <TableCell className="text-muted-foreground">{item.roleId}</TableCell>
                    <TableCell>
                      <div className="flex gap-2">
                        <Button size="sm" variant="outline" onClick={() => onEdit(item)}>
                          Edit
                        </Button>
                        <Button size="sm" variant="destructive" onClick={() => onDelete(item.id)}>
                          <Trash2Icon />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <EmptyState
            icon={<UsersRoundIcon />}
            label="No group mappings"
            description="Map directory groups to roles before enabling automated access assignment."
            compact
          />
        )}
      </div>
    </div>
  );
}

function MappingEditor({
  mapping,
  disabled,
  onSave,
}: {
  mapping: SCIMGroupRoleMapping;
  disabled: boolean;
  onSave: (mapping: SCIMGroupRoleMapping) => void;
}) {
  const { control, handleSubmit, register, reset } = useForm<MappingFormValues>({
    defaultValues: {
      externalGroupId: mapping.externalGroupId,
      displayName: mapping.displayName,
      roleId: mapping.roleId,
    },
  });

  useEffect(() => {
    reset({
      externalGroupId: mapping.externalGroupId,
      displayName: mapping.displayName,
      roleId: mapping.roleId,
    });
  }, [mapping, reset]);

  return (
    <form
      className="flex min-h-0 flex-1 flex-col"
      onSubmit={(event) => {
        event.stopPropagation();
        void handleSubmit((values) => onSave({ ...mapping, ...values }))(event);
      }}
    >
      <div className="space-y-3 px-4">
        <Field label="External group ID">
          <Input {...register("externalGroupId", { required: true })} disabled={disabled} />
        </Field>
        <Field label="Display name">
          <Input {...register("displayName")} disabled={disabled} />
        </Field>
        <RoleSelectAutocompleteField<MappingFormValues>
          control={control}
          name="roleId"
          label="Role"
          placeholder="Select role"
          clearable
          disabled={disabled}
          rules={{ required: true }}
        />
      </div>
      <SheetFooter className="border-t">
        <Button type="submit" disabled={disabled}>
          <SaveIcon />
          Save mapping
        </Button>
      </SheetFooter>
    </form>
  );
}

function AuditTimeline({
  records,
  isLoading,
}: {
  records: ProvisioningAuditRecord[];
  isLoading: boolean;
}) {
  return (
    <div className="rounded-lg border bg-background">
      <PanelHeader
        icon={<ActivityIcon />}
        title="Provisioning audit"
        description="Recent user and group synchronization events."
      />
      <div className="divide-y">
        {isLoading ? (
          <div className="space-y-2 p-3">
            <Skeleton className="h-12 w-full" />
            <Skeleton className="h-12 w-full" />
          </div>
        ) : records.length > 0 ? (
          records
            .slice(0, 8)
            .map((item) => (
              <ActivityItem
                key={item.id}
                title={`${toTitleCase(item.action)} ${item.resourceType}`}
                detail={
                  item.errorMessage || item.externalId || item.resourceId || "Provisioning event"
                }
                badge={item.status}
                when={formatTimestamp(item.createdAt)}
              />
            ))
        ) : (
          <EmptyState
            icon={<ActivityIcon />}
            label="No provisioning events"
            description="SCIM activity will appear after your directory starts syncing."
            compact
          />
        )}
      </div>
    </div>
  );
}

function PoliciesTab({ organizationId }: { organizationId: string }) {
  const queryClient = useQueryClient();
  const policiesQuery = useQuery(queries.organization.accessPolicies(organizationId));
  const resourcesQuery = useQuery({
    queryKey: ["permissions", "resources"],
    queryFn: getAvailableResources,
  });
  const operationsQuery = useQuery({
    queryKey: ["permissions", "operations"],
    queryFn: getAvailableOperations,
  });
  const [policy, setPolicy] = useState<AccessPolicy>(emptyPolicy);
  const [conditions, setConditions] = useState<ConditionRow[]>([]);
  const [sheetOpen, setSheetOpen] = useState(false);
  const [search, setSearch] = useState("");
  const [effectFilter, setEffectFilter] = useState("all");

  const resources = useMemo(
    () => (resourcesQuery.data ?? []).flatMap((category) => category.resources),
    [resourcesQuery.data],
  );
  const selectedResource = resources.find((resource) => resource.resource === policy.resource);
  const availableOperations = selectedResource?.operations.length
    ? selectedResource.operations
    : (operationsQuery.data ?? []);
  const policies = useMemo(
    () => [...(policiesQuery.data ?? [])].sort((left, right) => left.priority - right.priority),
    [policiesQuery.data],
  );
  const filteredPolicies = useMemo(() => {
    const query = search.trim().toLowerCase();
    return policies.filter((item) => {
      const matchesEffect = effectFilter === "all" || item.effect === effectFilter;
      const matchesSearch =
        !query ||
        [item.name, item.resource, item.operation, item.effect]
          .join(" ")
          .toLowerCase()
          .includes(query);
      return matchesEffect && matchesSearch;
    });
  }, [effectFilter, policies, search]);

  const saveMutation = useMutation({
    mutationFn: async (value: AccessPolicy) =>
      value.id
        ? apiService.organizationService.updateAccessPolicy(organizationId, value)
        : apiService.organizationService.createAccessPolicy(organizationId, value),
    onSuccess: async () => {
      toast.success("Access policy saved");
      setPolicy(emptyPolicy);
      setConditions([]);
      setSheetOpen(false);
      await queryClient.invalidateQueries({
        queryKey: queries.organization.accessPolicies(organizationId).queryKey,
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (policyId: string) =>
      apiService.organizationService.deleteAccessPolicy(organizationId, policyId),
    onSuccess: async () => {
      toast.success("Access policy removed");
      await queryClient.invalidateQueries({
        queryKey: queries.organization.accessPolicies(organizationId).queryKey,
      });
    },
  });

  const editPolicy = (item: AccessPolicy) => {
    setPolicy(item);
    setConditions(recordToConditionRows(item.conditions));
    setSheetOpen(true);
  };

  const createPolicy = () => {
    setPolicy(emptyPolicy);
    setConditions([]);
    setSheetOpen(true);
  };

  const savePolicy = () => {
    saveMutation.mutate({ ...policy, conditions: conditionRowsToRecord(conditions) });
  };

  return (
    <div className="space-y-3">
      <ConsoleToolbar
        title="Access policies"
        description="Priority-ordered authorization decisions for protected resources."
        search={search}
        onSearchChange={setSearch}
        searchPlaceholder="Search policies or resources"
        action={
          <div className="flex flex-wrap gap-2">
            <Select value={effectFilter} onValueChange={(value) => setEffectFilter(value ?? "all")}>
              <SelectTrigger className="h-8 w-32 bg-background text-xs">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All effects</SelectItem>
                <SelectItem value="allow">Allow</SelectItem>
                <SelectItem value="deny">Deny</SelectItem>
              </SelectContent>
            </Select>
            <Button onClick={createPolicy}>
              <PlusIcon />
              Add policy
            </Button>
          </div>
        }
      />

      {policiesQuery.isLoading ? (
        <RowSkeleton rows={4} />
      ) : policiesQuery.isError ? (
        <ErrorState label="Access policies could not be loaded." />
      ) : filteredPolicies.length > 0 ? (
        <div className="overflow-hidden rounded-lg border bg-background">
          {filteredPolicies.map((item) => (
            <PolicyRow
              key={item.id}
              policy={item}
              resource={resources.find((resource) => resource.resource === item.resource)}
              onEdit={() => editPolicy(item)}
              onDelete={() => deleteMutation.mutate(item.id)}
            />
          ))}
        </div>
      ) : (
        <EmptyState
          icon={<ShieldCheckIcon />}
          label={policies.length === 0 ? "No access policies configured" : "No policies found"}
          description={
            policies.length === 0
              ? "Create allow and deny policies to control sensitive operations."
              : "Adjust filters to find a policy."
          }
        />
      )}

      <Sheet open={sheetOpen} onOpenChange={setSheetOpen}>
        <SheetContent className="w-[calc(100vw-1rem)] overflow-hidden sm:max-w-xl">
          <SheetHeader className="border-b">
            <SheetTitle>{policy.id ? "Edit access policy" : "Add access policy"}</SheetTitle>
            <SheetDescription>
              Select a resource and operation, then define effect, priority, and optional
              conditions.
            </SheetDescription>
          </SheetHeader>
          <div className="min-h-0 flex-1 space-y-4 overflow-y-auto px-4">
            <PolicyForm
              policy={policy}
              resources={resources}
              operations={availableOperations}
              conditions={conditions}
              resourcesLoading={resourcesQuery.isLoading}
              operationsLoading={operationsQuery.isLoading}
              onPolicyChange={setPolicy}
              onConditionsChange={setConditions}
            />
          </div>
          <SheetFooter className="border-t">
            <Button
              onClick={savePolicy}
              isLoading={saveMutation.isPending}
              loadingText="Saving..."
              disabled={!policy.name.trim() || !policy.resource || !policy.operation}
            >
              <SaveIcon />
              Save policy
            </Button>
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </div>
  );
}

function PolicyRow({
  policy,
  resource,
  onEdit,
  onDelete,
}: {
  policy: AccessPolicy;
  resource?: ResourceDefinition;
  onEdit: () => void;
  onDelete: () => void;
}) {
  const conditionCount = Object.keys(policy.conditions).length;

  return (
    <div className="grid gap-3 border-b p-3 last:border-b-0 lg:grid-cols-[minmax(0,1fr)_160px] lg:items-center">
      <div className="min-w-0 space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <Badge variant={policy.effect === "allow" ? "active" : "inactive"}>
            {policy.effect === "allow" ? "Allow" : "Deny"}
          </Badge>
          <span className="font-medium">{policy.name}</span>
          <Badge variant={policy.enabled ? "info" : "outline"}>Priority {policy.priority}</Badge>
          {!policy.enabled && <Badge variant="outline">Disabled</Badge>}
        </div>
        <div className="flex flex-wrap gap-2 text-xs text-muted-foreground">
          <span className="rounded bg-muted px-1.5 py-0.5">
            {resource?.displayName || policy.resource}
          </span>
          <span className="rounded bg-muted px-1.5 py-0.5">{policy.operation}</span>
          <span>
            {conditionCount} condition{conditionCount === 1 ? "" : "s"}
          </span>
        </div>
      </div>
      <div className="flex justify-end gap-2">
        <Button variant="outline" onClick={onEdit}>
          Edit
        </Button>
        <Button variant="destructive" onClick={onDelete}>
          <Trash2Icon />
          Delete
        </Button>
      </div>
    </div>
  );
}

function PolicyForm({
  policy,
  resources,
  operations,
  conditions,
  resourcesLoading,
  operationsLoading,
  onPolicyChange,
  onConditionsChange,
}: {
  policy: AccessPolicy;
  resources: ResourceDefinition[];
  operations: OperationDefinition[];
  conditions: ConditionRow[];
  resourcesLoading: boolean;
  operationsLoading: boolean;
  onPolicyChange: (policy: AccessPolicy) => void;
  onConditionsChange: (conditions: ConditionRow[]) => void;
}) {
  const [resourceSearch, setResourceSearch] = useState("");
  const [operationSearch, setOperationSearch] = useState("");
  const filteredResources = useMemo(() => {
    const query = resourceSearch.trim().toLowerCase();
    if (!query) return resources;
    return resources.filter((resource) =>
      [resource.displayName, resource.resource, resource.category, resource.description]
        .join(" ")
        .toLowerCase()
        .includes(query),
    );
  }, [resourceSearch, resources]);
  const filteredOperations = useMemo(() => {
    const query = operationSearch.trim().toLowerCase();
    if (!query) return operations;
    return operations.filter((operation) =>
      [operation.displayName, operation.operation, operation.description]
        .join(" ")
        .toLowerCase()
        .includes(query),
    );
  }, [operationSearch, operations]);

  return (
    <div className="space-y-4 py-4">
      <Field label="Name">
        <Input
          value={policy.name}
          placeholder="Require managed devices for billing exports"
          onChange={(event) => onPolicyChange({ ...policy, name: event.target.value })}
        />
      </Field>
      <div className="grid gap-3 sm:grid-cols-2">
        <SearchablePolicySelect
          label="Resource"
          value={policy.resource}
          search={resourceSearch}
          searchPlaceholder="Search resources"
          selectPlaceholder="Select resource"
          disabled={resourcesLoading}
          options={filteredResources.map((resource) => ({
            value: resource.resource,
            label: resource.displayName,
          }))}
          onSearchChange={setResourceSearch}
          onValueChange={(value) => onPolicyChange({ ...policy, resource: value, operation: "" })}
        />
        <SearchablePolicySelect
          label="Operation"
          value={policy.operation}
          search={operationSearch}
          searchPlaceholder="Search operations"
          selectPlaceholder="Select operation"
          disabled={operationsLoading || operations.length === 0}
          options={filteredOperations.map((operation) => ({
            value: operation.operation,
            label: operation.displayName || operation.operation,
          }))}
          onSearchChange={setOperationSearch}
          onValueChange={(value) => onPolicyChange({ ...policy, operation: value })}
        />
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label="Effect">
          <Select
            value={policy.effect}
            onValueChange={(value) =>
              onPolicyChange({ ...policy, effect: (value ?? "deny") as AccessPolicy["effect"] })
            }
          >
            <SelectTrigger className="w-full bg-background">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="deny">Deny</SelectItem>
              <SelectItem value="allow">Allow</SelectItem>
            </SelectContent>
          </Select>
        </Field>
        <Field label="Priority">
          <Input
            type="number"
            value={policy.priority}
            onChange={(event) =>
              onPolicyChange({ ...policy, priority: Number(event.target.value || 0) })
            }
          />
        </Field>
      </div>
      <ToggleRow
        label="Enabled"
        description="Evaluate this policy during access decisions."
        checked={policy.enabled}
        onCheckedChange={(enabled) => onPolicyChange({ ...policy, enabled })}
      />
      <ConditionsEditor conditions={conditions} onChange={onConditionsChange} />
    </div>
  );
}

function ConditionsEditor({
  conditions,
  onChange,
}: {
  conditions: ConditionRow[];
  onChange: (conditions: ConditionRow[]) => void;
}) {
  const addCondition = () => {
    onChange([...conditions, { id: `${Date.now()}-${conditions.length}`, key: "", value: "" }]);
  };

  return (
    <div className="space-y-2 rounded-lg border bg-muted/20 p-3">
      <div className="flex items-center justify-between gap-3">
        <div>
          <div className="text-sm font-medium">Conditions</div>
          <div className="text-xs text-muted-foreground">
            Optional key/value checks persisted with the policy.
          </div>
        </div>
        <Button size="sm" variant="outline" onClick={addCondition}>
          <PlusIcon />
          Add
        </Button>
      </div>
      {conditions.length > 0 ? (
        <div className="space-y-2">
          {conditions.map((condition, index) => (
            <div key={condition.id} className="grid gap-2 sm:grid-cols-[1fr_1fr_32px]">
              <Input
                value={condition.key}
                placeholder="claim"
                onChange={(event) => {
                  const next = [...conditions];
                  next[index] = { ...condition, key: event.target.value };
                  onChange(next);
                }}
              />
              <Input
                value={condition.value}
                placeholder="expected value"
                onChange={(event) => {
                  const next = [...conditions];
                  next[index] = { ...condition, value: event.target.value };
                  onChange(next);
                }}
              />
              <Button
                size="icon"
                variant="ghost"
                onClick={() => onChange(conditions.filter((item) => item.id !== condition.id))}
              >
                <XIcon />
              </Button>
            </div>
          ))}
        </div>
      ) : (
        <div className="rounded-md border border-dashed bg-background p-3 text-xs text-muted-foreground">
          No conditions. The policy applies whenever the resource and operation match.
        </div>
      )}
    </div>
  );
}

function SearchablePolicySelect({
  label,
  value,
  search,
  searchPlaceholder,
  selectPlaceholder,
  disabled,
  options,
  onSearchChange,
  onValueChange,
}: {
  label: string;
  value: string;
  search: string;
  searchPlaceholder: string;
  selectPlaceholder: string;
  disabled: boolean;
  options: Array<{ value: string; label: string }>;
  onSearchChange: (value: string) => void;
  onValueChange: (value: string) => void;
}) {
  return (
    <div className="grid gap-1 text-sm">
      <span className="font-medium">{label}</span>
      <div className="relative">
        <SearchIcon className="pointer-events-none absolute top-1/2 left-2 size-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          value={search}
          placeholder={searchPlaceholder}
          className="pl-8"
          disabled={disabled}
          onChange={(event) => onSearchChange(event.target.value)}
        />
      </div>
      <Select
        value={value}
        onValueChange={(nextValue) => onValueChange(nextValue ?? "")}
        disabled={disabled || options.length === 0}
      >
        <SelectTrigger className="w-full bg-background">
          <SelectValue placeholder={selectPlaceholder} />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            {options.map((option) => (
              <SelectItem key={option.value} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectGroup>
        </SelectContent>
      </Select>
    </div>
  );
}

function ActivityTab({ organizationId }: { organizationId: string }) {
  const authEventsQuery = useQuery(queries.organization.authEvents(organizationId));
  const riskQuery = useQuery(queries.organization.riskDecisions(organizationId));
  const externalIdentityQuery = useQuery(queries.organization.externalIdentities(organizationId));
  const mfaQuery = useQuery(queries.organization.mfaAuthenticators(organizationId));
  const [view, setView] = useState<ActivityView>("auth");
  const [search, setSearch] = useState("");

  const query = search.trim().toLowerCase();
  const authEvents = (authEventsQuery.data ?? []).filter((item) =>
    [item.provider, item.outcome, item.riskOutcome, item.ipAddress, item.errorCode]
      .join(" ")
      .toLowerCase()
      .includes(query),
  );
  const riskDecisions = (riskQuery.data ?? []).filter((item) =>
    [item.outcome, item.reason, item.signals.join(" ")].join(" ").toLowerCase().includes(query),
  );
  const identities = (externalIdentityQuery.data ?? []).filter((item) =>
    [item.externalEmail, item.externalUsername, item.externalSubject]
      .join(" ")
      .toLowerCase()
      .includes(query),
  );
  const authenticators = (mfaQuery.data ?? []).filter((item) =>
    [item.name, item.type, item.enabled ? "enabled" : "disabled"]
      .join(" ")
      .toLowerCase()
      .includes(query),
  );

  return (
    <div className="space-y-3">
      <ConsoleToolbar
        title="Activity console"
        description="Authentication outcomes, risk decisions, linked identities, and MFA devices."
        search={search}
        onSearchChange={setSearch}
        searchPlaceholder="Search activity"
        action={
          <Select
            value={view}
            onValueChange={(value) => setView((value ?? "auth") as ActivityView)}
          >
            <SelectTrigger className="h-8 w-44 bg-background text-xs">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="auth">Auth events</SelectItem>
              <SelectItem value="risk">Risk decisions</SelectItem>
              <SelectItem value="identities">External identities</SelectItem>
              <SelectItem value="mfa">MFA authenticators</SelectItem>
            </SelectContent>
          </Select>
        }
      />
      <div className="grid gap-3 md:grid-cols-4">
        <ActivityViewButton
          active={view === "auth"}
          icon={<KeyRoundIcon />}
          label="Auth events"
          count={authEventsQuery.data?.length ?? 0}
          onClick={() => setView("auth")}
        />
        <ActivityViewButton
          active={view === "risk"}
          icon={<ShieldAlertIcon />}
          label="Risk decisions"
          count={riskQuery.data?.length ?? 0}
          onClick={() => setView("risk")}
        />
        <ActivityViewButton
          active={view === "identities"}
          icon={<ExternalLinkIcon />}
          label="External identities"
          count={externalIdentityQuery.data?.length ?? 0}
          onClick={() => setView("identities")}
        />
        <ActivityViewButton
          active={view === "mfa"}
          icon={<LockKeyholeIcon />}
          label="MFA authenticators"
          count={mfaQuery.data?.length ?? 0}
          onClick={() => setView("mfa")}
        />
      </div>
      {view === "auth" &&
        (authEventsQuery.isError ? (
          <ErrorState label="Authentication events could not be loaded." />
        ) : (
          <AuthEventsTable records={authEvents} isLoading={authEventsQuery.isLoading} />
        ))}
      {view === "risk" &&
        (riskQuery.isError ? (
          <ErrorState label="Risk decisions could not be loaded." />
        ) : (
          <RiskDecisionsTable records={riskDecisions} isLoading={riskQuery.isLoading} />
        ))}
      {view === "identities" &&
        (externalIdentityQuery.isError ? (
          <ErrorState label="External identities could not be loaded." />
        ) : (
          <ExternalIdentitiesTable
            records={identities}
            isLoading={externalIdentityQuery.isLoading}
          />
        ))}
      {view === "mfa" &&
        (mfaQuery.isError ? (
          <ErrorState label="MFA authenticators could not be loaded." />
        ) : (
          <MFAAuthenticatorsTable records={authenticators} isLoading={mfaQuery.isLoading} />
        ))}
    </div>
  );
}

function ActivityViewButton({
  active,
  icon,
  label,
  count,
  onClick,
}: {
  active: boolean;
  icon: ReactNode;
  label: string;
  count: number;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      className={cn(
        "flex items-center justify-between gap-2 rounded-lg border bg-background p-3 text-left transition-colors hover:bg-muted/40",
        active && "border-primary/40 bg-primary/5",
      )}
      onClick={onClick}
    >
      <span className="flex items-center gap-2 text-sm font-medium">
        <span className="text-muted-foreground [&_svg]:size-4">{icon}</span>
        {label}
      </span>
      <Badge variant={active ? "info" : "outline"}>{count}</Badge>
    </button>
  );
}

function AuthEventsTable({ records, isLoading }: { records: AuthEvent[]; isLoading: boolean }) {
  return (
    <ActivityTableShell
      isLoading={isLoading}
      rowCount={records.length}
      emptyLabel="No authentication events found."
      headers={["Provider", "Outcome", "Risk", "When", "Detail"]}
    >
      {records.map((item) => (
        <TableRow key={item.id}>
          <TableCell>
            <div className="font-medium">{formatProviderName(item.provider)}</div>
            <div className="text-xs text-muted-foreground">
              {item.ipAddress || "No IP captured"}
            </div>
          </TableCell>
          <TableCell>
            <Badge variant={outcomeVariant(item.outcome)}>{toTitleCase(item.outcome)}</Badge>
          </TableCell>
          <TableCell>
            <Badge variant={riskVariant(item.riskOutcome)}>{toTitleCase(item.riskOutcome)}</Badge>
          </TableCell>
          <TableCell className="text-muted-foreground">
            {formatTimestamp(item.occurredAt)}
          </TableCell>
          <TableCell className="max-w-72 truncate text-muted-foreground">
            {item.errorCode || item.riskSignals.join(", ") || "-"}
          </TableCell>
        </TableRow>
      ))}
    </ActivityTableShell>
  );
}

function RiskDecisionsTable({
  records,
  isLoading,
}: {
  records: RiskDecision[];
  isLoading: boolean;
}) {
  return (
    <ActivityTableShell
      isLoading={isLoading}
      rowCount={records.length}
      emptyLabel="No risk decisions found."
      headers={["Outcome", "Reason", "Signals", "When"]}
    >
      {records.map((item) => (
        <TableRow key={item.id}>
          <TableCell>
            <Badge variant={riskVariant(item.outcome)}>{toTitleCase(item.outcome)}</Badge>
          </TableCell>
          <TableCell className="max-w-80 truncate">{item.reason || "-"}</TableCell>
          <TableCell className="max-w-80 truncate text-muted-foreground">
            {item.signals.join(", ") || "-"}
          </TableCell>
          <TableCell className="text-muted-foreground">{formatTimestamp(item.createdAt)}</TableCell>
        </TableRow>
      ))}
    </ActivityTableShell>
  );
}

function ExternalIdentitiesTable({
  records,
  isLoading,
}: {
  records: ExternalIdentity[];
  isLoading: boolean;
}) {
  return (
    <ActivityTableShell
      isLoading={isLoading}
      rowCount={records.length}
      emptyLabel="No external identities found."
      headers={["Identity", "Subject", "Last login", "Created"]}
    >
      {records.map((item) => (
        <TableRow key={item.id}>
          <TableCell>
            <div className="font-medium">{item.externalEmail || "-"}</div>
            <div className="text-xs text-muted-foreground">{item.externalUsername || "-"}</div>
          </TableCell>
          <TableCell className="max-w-80 truncate">{item.externalSubject}</TableCell>
          <TableCell className="text-muted-foreground">
            {item.lastLoginAt ? formatTimestamp(item.lastLoginAt) : "Never"}
          </TableCell>
          <TableCell className="text-muted-foreground">{formatTimestamp(item.createdAt)}</TableCell>
        </TableRow>
      ))}
    </ActivityTableShell>
  );
}

function MFAAuthenticatorsTable({
  records,
  isLoading,
}: {
  records: MFAAuthenticator[];
  isLoading: boolean;
}) {
  return (
    <ActivityTableShell
      isLoading={isLoading}
      rowCount={records.length}
      emptyLabel="No MFA authenticators found."
      headers={["Authenticator", "Status", "Verified", "Last used"]}
    >
      {records.map((item) => (
        <TableRow key={item.id}>
          <TableCell>
            <div className="font-medium">{item.name}</div>
            <div className="text-xs text-muted-foreground">{item.type.toUpperCase()}</div>
          </TableCell>
          <TableCell>
            <Badge variant={item.enabled ? "active" : "inactive"}>
              {item.enabled ? "Enabled" : "Disabled"}
            </Badge>
          </TableCell>
          <TableCell className="text-muted-foreground">
            {item.verifiedAt ? formatTimestamp(item.verifiedAt) : "Not verified"}
          </TableCell>
          <TableCell className="text-muted-foreground">
            {item.lastUsedAt ? formatTimestamp(item.lastUsedAt) : "Never"}
          </TableCell>
        </TableRow>
      ))}
    </ActivityTableShell>
  );
}

function ActivityTableShell({
  isLoading,
  rowCount,
  emptyLabel,
  headers,
  children,
}: {
  isLoading: boolean;
  rowCount: number;
  emptyLabel: string;
  headers: string[];
  children: ReactNode;
}) {
  if (isLoading) {
    return <RowSkeleton rows={5} />;
  }

  if (rowCount === 0) {
    return (
      <EmptyState
        icon={<ActivityIcon />}
        label={emptyLabel}
        description="Try a different filter."
      />
    );
  }

  return (
    <div className="overflow-hidden rounded-lg border bg-background">
      <div className="overflow-x-auto">
        <Table>
          <TableHeader>
            <TableRow>
              {headers.map((header) => (
                <TableHead key={header}>{header}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>{children}</TableBody>
        </Table>
      </div>
    </div>
  );
}

function ConsoleToolbar({
  title,
  description,
  search,
  onSearchChange,
  searchPlaceholder,
  action,
}: {
  title: string;
  description: string;
  search: string;
  onSearchChange: (value: string) => void;
  searchPlaceholder: string;
  action?: ReactNode;
}) {
  return (
    <div className="rounded-lg border bg-muted/20 p-3">
      <div className="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
        <div>
          <h3 className="text-base font-semibold tracking-tight">{title}</h3>
          <p className="text-sm text-muted-foreground">{description}</p>
        </div>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <div className="relative min-w-0 sm:w-80">
            <SearchIcon className="pointer-events-none absolute top-1/2 left-2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={search}
              placeholder={searchPlaceholder}
              className="pl-8"
              onChange={(event) => onSearchChange(event.target.value)}
            />
          </div>
          {action}
        </div>
      </div>
    </div>
  );
}

function PanelHeader({
  icon,
  title,
  description,
  action,
}: {
  icon: ReactNode;
  title: string;
  description: string;
  action?: ReactNode;
}) {
  return (
    <div className="flex flex-col gap-3 border-b p-3 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex items-center gap-2">
        <span className="flex size-8 items-center justify-center rounded-md border bg-muted/40 text-muted-foreground [&_svg]:size-4">
          {icon}
        </span>
        <div>
          <div className="text-sm font-medium">{title}</div>
          <div className="text-xs text-muted-foreground">{description}</div>
        </div>
      </div>
      {action}
    </div>
  );
}

function ActivityItem({
  title,
  detail,
  badge,
  when,
}: {
  title: string;
  detail: string;
  badge: string;
  when: string;
}) {
  return (
    <div className="flex items-start justify-between gap-3 p-3">
      <div className="min-w-0">
        <div className="truncate text-sm font-medium">{title}</div>
        <div className="truncate text-xs text-muted-foreground">{detail}</div>
      </div>
      <div className="flex shrink-0 flex-col items-end gap-1">
        <Badge variant={riskVariant(badge)}>{toTitleCase(badge)}</Badge>
        <span className="text-xs text-muted-foreground">{when}</span>
      </div>
    </div>
  );
}

function ProviderLogo({ name }: { name: string }) {
  const lowerName = name.toLowerCase();
  if (
    lowerName.includes("entra") ||
    lowerName.includes("microsoft") ||
    lowerName.includes("azure")
  ) {
    return <EntraLogo className="size-5" />;
  }
  if (lowerName.includes("okta")) {
    return <OktaLogo className="h-5 w-auto" />;
  }
  return <KeyRoundIcon className="size-5 text-primary" />;
}

function ToggleRow({
  label,
  description,
  checked,
  onCheckedChange,
}: {
  label: string;
  description?: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <label className="flex items-center justify-between gap-3 rounded-lg border bg-background px-3 py-2 text-sm">
      <span className="min-w-0">
        <span className="block font-medium">{label}</span>
        {description && <span className="block text-xs text-muted-foreground">{description}</span>}
      </span>
      <Switch checked={checked} onCheckedChange={onCheckedChange} />
    </label>
  );
}

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <label className="grid gap-1 text-sm">
      <span className="font-medium">{label}</span>
      {children}
    </label>
  );
}

function ChipArrayField({
  control,
  name,
  label,
  placeholder,
  description,
  parseValue,
  formatValue,
  required,
}: {
  control: ReturnType<typeof useForm<IdentityProviderFormValues>>["control"];
  name: "allowedDomains" | "oidcScopes";
  label: string;
  placeholder: string;
  description: string;
  parseValue: (value: string) => string[];
  formatValue: (value: string[]) => string;
  required?: boolean;
}) {
  return (
    <Controller
      name={name}
      control={control}
      rules={required ? { required: true } : undefined}
      render={({ field, fieldState }) => {
        const chips = Array.isArray(field.value) ? field.value : [];

        return (
          <FieldWrapper
            label={label}
            description={description}
            required={required}
            error={fieldState.error?.message}
          >
            <div className="space-y-2">
              <Input
                value={formatValue(chips)}
                placeholder={placeholder}
                aria-invalid={fieldState.invalid}
                onChange={(event) => field.onChange(parseValue(event.target.value))}
              />
              {chips.length > 0 && (
                <div className="flex flex-wrap gap-1.5">
                  {chips.map((chip) => (
                    <span key={chip} className="rounded-full border bg-muted px-2 py-0.5 text-xs">
                      {chip}
                    </span>
                  ))}
                </div>
              )}
            </div>
          </FieldWrapper>
        );
      }}
    />
  );
}

function MetaLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0 truncate">
      <span className="font-medium text-foreground">{label}:</span> {value}
    </div>
  );
}

function EmptyState({
  icon,
  label,
  description,
  compact,
}: {
  icon?: ReactNode;
  label: string;
  description?: string;
  compact?: boolean;
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center rounded-lg border border-dashed text-center",
        compact ? "m-3 p-4" : "p-8",
      )}
    >
      {icon && (
        <span className="mb-2 flex size-9 items-center justify-center rounded-md bg-muted text-muted-foreground [&_svg]:size-4">
          {icon}
        </span>
      )}
      <div className="text-sm font-medium">{label}</div>
      {description && (
        <div className="mt-1 max-w-md text-sm text-muted-foreground">{description}</div>
      )}
    </div>
  );
}

function ErrorState({ label, compact }: { label: string; compact?: boolean }) {
  return (
    <div
      className={cn(
        "flex items-center gap-2 rounded-lg border border-red-600/30 bg-red-600/10 text-sm text-red-700 dark:text-red-400",
        compact ? "m-3 p-3" : "p-4",
      )}
    >
      <AlertTriangleIcon className="size-4" />
      {label}
    </div>
  );
}

function RowSkeleton({ rows }: { rows: number }) {
  return (
    <div className="space-y-2 rounded-lg border bg-background p-3">
      {Array.from({ length: rows }).map((_, index) => (
        <Skeleton key={index} className="h-16 w-full rounded-md" />
      ))}
    </div>
  );
}

function OverviewSkeleton() {
  return (
    <div className="grid gap-3 xl:grid-cols-[minmax(0,1fr)_360px]">
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        {Array.from({ length: 4 }).map((_, index) => (
          <Skeleton key={index} className="h-24 rounded-lg" />
        ))}
      </div>
      <Skeleton className="h-32 rounded-lg" />
    </div>
  );
}

function parseCSV(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function parseWords(value: string) {
  return value
    .split(/\s+/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function recordToConditionRows(value: Record<string, string>) {
  return Object.entries(value).map(([key, mapValue], index) => ({
    id: `${key}-${index}`,
    key,
    value: mapValue,
  }));
}

function conditionRowsToRecord(rows: ConditionRow[]) {
  return Object.fromEntries(
    rows.map((row) => [row.key.trim(), row.value.trim()] as const).filter(([key]) => key),
  );
}

function toIdentityProvider(values: IdentityProviderFormValues): IdentityProvider {
  return {
    ...emptyProvider,
    ...values,
    protocol: "OIDC",
    allowedDomains: values.allowedDomains ?? [],
    attributeMap: values.attributeMap ?? { email: "email" },
    oidcScopes: values.oidcScopes ?? [],
  };
}

function formatProviderName(name: string) {
  return name
    .replace(/azure\s*ad/gi, "Entra ID")
    .replace(/microsoft entra id/gi, "Microsoft Entra ID");
}

function formatTimestamp(value: number) {
  return value ? formatUnixDateTime(value) : "-";
}

function riskVariant(value: string): BadgeVariant {
  switch (value) {
    case "allow":
    case "success":
    case "active":
    case "created":
    case "updated":
    case "completed":
      return "active";
    case "challenge":
    case "pending":
      return "warning";
    case "deny":
    case "denied":
    case "failed":
    case "error":
    case "revoked":
      return "inactive";
    default:
      return "outline";
  }
}

function outcomeVariant(value: string): BadgeVariant {
  switch (value) {
    case "success":
      return "active";
    case "challenge":
      return "warning";
    case "denied":
    case "failed":
      return "inactive";
    default:
      return "outline";
  }
}
