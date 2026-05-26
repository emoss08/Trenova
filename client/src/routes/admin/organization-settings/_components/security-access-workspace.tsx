import { EntraLogo } from "@/components/logos/entra";
import { OktaLogo } from "@/components/logos/okta";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
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
import { Textarea } from "@/components/ui/textarea";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type {
  AccessPolicy,
  AuthEvent,
  ExternalIdentity,
  IdentityProvider,
  MFAAuthenticator,
  ProvisioningAuditRecord,
  RiskDecision,
  SCIMDirectory,
  SCIMGroupRoleMapping,
  SCIMToken,
} from "@/types/iam";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  ActivityIcon,
  KeyRoundIcon,
  PlusIcon,
  SaveIcon,
  ShieldCheckIcon,
  Trash2Icon,
  UsersRoundIcon,
} from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

type SecurityTab = "sign-in" | "provisioning" | "policies" | "activity";

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

export function SecurityAccessWorkspace({ organizationId }: { organizationId: string }) {
  const [tab, setTab] = useState<SecurityTab>("sign-in");

  return (
    <section className="space-y-4 pb-10">
      <div className="flex flex-col gap-1 border-b pb-3">
        <div className="flex items-center gap-2">
          <ShieldCheckIcon className="size-5 text-primary" />
          <h2 className="text-lg font-semibold tracking-tight">Security & Access</h2>
        </div>
        <p className="max-w-3xl text-sm text-muted-foreground">
          Manage federated sign-in, SCIM provisioning, access policies, and security activity.
        </p>
      </div>
      <Tabs value={tab} onValueChange={(value) => setTab(value as SecurityTab)} className="gap-4">
        <TabsList variant="underline" className="overflow-x-auto">
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

function SignInTab({ organizationId }: { organizationId: string }) {
  const queryClient = useQueryClient();
  const providersQuery = useQuery(queries.organization.identityProviders(organizationId));
  const [editingProvider, setEditingProvider] = useState<IdentityProvider>(emptyProvider);

  const saveMutation = useMutation({
    mutationFn: async (provider: IdentityProvider) =>
      provider.id
        ? apiService.organizationService.updateIdentityProvider(organizationId, provider)
        : apiService.organizationService.createIdentityProvider(organizationId, provider),
    onSuccess: async () => {
      toast.success("Identity provider saved");
      setEditingProvider(emptyProvider);
      await queryClient.invalidateQueries({
        queryKey: queries.organization.identityProviders(organizationId).queryKey,
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: async (providerId: string) =>
      apiService.organizationService.deleteIdentityProvider(organizationId, providerId),
    onSuccess: async () => {
      toast.success("Identity provider removed");
      await queryClient.invalidateQueries({
        queryKey: queries.organization.identityProviders(organizationId).queryKey,
      });
    },
  });

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div className="space-y-3">
        {(providersQuery.data ?? []).map((provider) => (
          <Card key={provider.id} className="rounded-lg">
            <CardContent className="flex flex-col gap-3 p-4 md:flex-row md:items-center md:justify-between">
              <div className="min-w-0 space-y-2">
                <div className="flex flex-wrap items-center gap-2">
                  <ProviderLogo name={provider.name} />
                  <span className="font-medium">{provider.name}</span>
                  <Badge variant={provider.enabled ? "active" : "inactive"}>
                    {provider.enabled ? "Enabled" : "Disabled"}
                  </Badge>
                  {provider.enforceSso && <Badge variant="warning">SSO enforced</Badge>}
                  {provider.autoProvision && <Badge variant="info">Auto-provisioning</Badge>}
                </div>
                <div className="grid gap-1 text-xs text-muted-foreground md:grid-cols-2">
                  <span className="truncate">Issuer: {provider.oidcIssuerUrl}</span>
                  <span className="truncate">Redirect: {provider.oidcRedirectUrl}</span>
                  <span>Domains: {provider.allowedDomains.join(", ") || "Any"}</span>
                  <span>Scopes: {provider.oidcScopes.join(" ")}</span>
                </div>
              </div>
              <div className="flex shrink-0 gap-2">
                <Button variant="outline" onClick={() => setEditingProvider(provider)}>
                  Edit
                </Button>
                <Button
                  variant="destructive"
                  onClick={() => deleteMutation.mutate(provider.id)}
                  disabled={deleteMutation.isPending}
                >
                  <Trash2Icon />
                  Delete
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
        {providersQuery.data?.length === 0 && <EmptyState label="No OIDC providers configured." />}
      </div>
      <Card className="rounded-lg">
        <CardHeader>
          <CardTitle>{editingProvider.id ? "Edit provider" : "Add OIDC provider"}</CardTitle>
        </CardHeader>
        <CardContent>
          <ProviderForm
            value={editingProvider}
            onChange={setEditingProvider}
            onSubmit={() => saveMutation.mutate(editingProvider)}
            isSaving={saveMutation.isPending}
          />
        </CardContent>
      </Card>
    </div>
  );
}

function ProviderForm({
  value,
  onChange,
  onSubmit,
  isSaving,
}: {
  value: IdentityProvider;
  onChange: (value: IdentityProvider) => void;
  onSubmit: () => void;
  isSaving: boolean;
}) {
  return (
    <div className="space-y-3">
      <Field label="Name">
        <Input
          value={value.name}
          onChange={(event) => onChange({ ...value, name: event.target.value })}
        />
      </Field>
      <Field label="Slug">
        <Input
          value={value.slug}
          onChange={(event) => onChange({ ...value, slug: event.target.value })}
        />
      </Field>
      <Field label="Issuer URL">
        <Input
          value={value.oidcIssuerUrl}
          onChange={(event) => onChange({ ...value, oidcIssuerUrl: event.target.value })}
        />
      </Field>
      <Field label="Client ID">
        <Input
          value={value.oidcClientId}
          onChange={(event) => onChange({ ...value, oidcClientId: event.target.value })}
        />
      </Field>
      <Field label="Client secret">
        <Input
          type="password"
          value={value.oidcClientSecret}
          placeholder={value.id ? "Leave blank to keep current secret" : ""}
          onChange={(event) => onChange({ ...value, oidcClientSecret: event.target.value })}
        />
      </Field>
      <Field label="Redirect URL">
        <Input
          value={value.oidcRedirectUrl}
          onChange={(event) => onChange({ ...value, oidcRedirectUrl: event.target.value })}
        />
      </Field>
      <Field label="Allowed domains">
        <Input
          value={value.allowedDomains.join(", ")}
          onChange={(event) => onChange({ ...value, allowedDomains: parseCSV(event.target.value) })}
        />
      </Field>
      <Field label="OIDC scopes">
        <Input
          value={value.oidcScopes.join(" ")}
          onChange={(event) => onChange({ ...value, oidcScopes: parseWords(event.target.value) })}
        />
      </Field>
      <div className="grid gap-2">
        <ToggleRow
          label="Enabled"
          checked={value.enabled}
          onCheckedChange={(enabled) => onChange({ ...value, enabled })}
        />
        <ToggleRow
          label="Enforce SSO"
          checked={value.enforceSso}
          onCheckedChange={(enforceSso) => onChange({ ...value, enforceSso })}
        />
        <ToggleRow
          label="Auto-provision users"
          checked={value.autoProvision}
          onCheckedChange={(autoProvision) => onChange({ ...value, autoProvision })}
        />
        <ToggleRow
          label="Trust federated MFA"
          checked={value.allowFederatedMfa}
          onCheckedChange={(allowFederatedMfa) => onChange({ ...value, allowFederatedMfa })}
        />
      </div>
      <Button className="w-full" onClick={onSubmit} disabled={isSaving}>
        <SaveIcon />
        {isSaving ? "Saving..." : "Save provider"}
      </Button>
    </div>
  );
}

function ProvisioningTab({ organizationId }: { organizationId: string }) {
  const queryClient = useQueryClient();
  const directoriesQuery = useQuery(queries.organization.scimDirectories(organizationId));
  const auditQuery = useQuery(queries.organization.provisioningAudit(organizationId));
  const [directory, setDirectory] = useState<SCIMDirectory>(emptyDirectory);
  const [selectedDirectoryId, setSelectedDirectoryId] = useState("");
  const [tokenName, setTokenName] = useState("");
  const [createdToken, setCreatedToken] = useState("");
  const [mapping, setMapping] = useState<SCIMGroupRoleMapping>(emptyMapping);

  const directoryId = selectedDirectoryId || directoriesQuery.data?.[0]?.id || "";
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

  const invalidateProvisioning = async () => {
    await queryClient.invalidateQueries({
      queryKey: queries.organization.scimDirectories(organizationId).queryKey,
    });
    await queryClient.invalidateQueries({ queryKey: ["scim-tokens", organizationId, directoryId] });
    await queryClient.invalidateQueries({
      queryKey: ["scim-mappings", organizationId, directoryId],
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
      await invalidateProvisioning();
    },
  });

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div className="space-y-4">
        <Card className="rounded-lg">
          <CardHeader>
            <CardTitle>SCIM directories</CardTitle>
          </CardHeader>
          <CardContent>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Tenant slug</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="w-28">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {(directoriesQuery.data ?? []).map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>{item.tenantSlug}</TableCell>
                    <TableCell>
                      <Badge variant={item.enabled ? "active" : "inactive"}>
                        {item.enabled ? "Enabled" : "Disabled"}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          setDirectory(item);
                          setSelectedDirectoryId(item.id);
                        }}
                      >
                        Edit
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
        <SCIMTokenPanel
          tokens={tokensQuery.data ?? []}
          tokenName={tokenName}
          createdToken={createdToken}
          setTokenName={setTokenName}
          onCreate={() => createTokenMutation.mutate()}
          onRevoke={(tokenId) => revokeTokenMutation.mutate(tokenId)}
          disabled={!directoryId || createTokenMutation.isPending}
        />
        <MappingsPanel
          mappings={mappingsQuery.data ?? []}
          mapping={mapping}
          setMapping={setMapping}
          onSave={() => saveMappingMutation.mutate(mapping)}
          disabled={!directoryId || saveMappingMutation.isPending}
        />
        <AuditTable records={auditQuery.data ?? []} />
      </div>
      <Card className="rounded-lg">
        <CardHeader>
          <CardTitle>{directory.id ? "Edit directory" : "Add SCIM directory"}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <Field label="Tenant slug">
            <Input
              value={directory.tenantSlug}
              onChange={(event) => setDirectory({ ...directory, tenantSlug: event.target.value })}
            />
          </Field>
          <ToggleRow
            label="Enabled"
            checked={directory.enabled}
            onCheckedChange={(enabled) => setDirectory({ ...directory, enabled })}
          />
          <Button
            className="w-full"
            onClick={() => saveDirectoryMutation.mutate(directory)}
            disabled={saveDirectoryMutation.isPending}
          >
            <SaveIcon />
            Save directory
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

function SCIMTokenPanel({
  tokens,
  tokenName,
  createdToken,
  setTokenName,
  onCreate,
  onRevoke,
  disabled,
}: {
  tokens: SCIMToken[];
  tokenName: string;
  createdToken: string;
  setTokenName: (value: string) => void;
  onCreate: () => void;
  onRevoke: (tokenId: string) => void;
  disabled: boolean;
}) {
  return (
    <Card className="rounded-lg">
      <CardHeader>
        <CardTitle>SCIM tokens</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="flex flex-col gap-2 sm:flex-row">
          <Input value={tokenName} onChange={(event) => setTokenName(event.target.value)} />
          <Button onClick={onCreate} disabled={disabled || tokenName.trim() === ""}>
            <PlusIcon />
            Create token
          </Button>
        </div>
        {createdToken && (
          <div className="rounded-md border border-amber-600/30 bg-amber-600/10 p-2 font-mono text-xs break-all text-amber-700 dark:text-amber-300">
            {createdToken}
          </div>
        )}
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Prefix</TableHead>
              <TableHead>Status</TableHead>
              <TableHead className="w-28">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {tokens.map((token) => (
              <TableRow key={token.id}>
                <TableCell>{token.name}</TableCell>
                <TableCell>{token.prefix}</TableCell>
                <TableCell>
                  <Badge variant={token.status === "active" ? "active" : "inactive"}>
                    {token.status}
                  </Badge>
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
      </CardContent>
    </Card>
  );
}

function MappingsPanel({
  mappings,
  mapping,
  setMapping,
  onSave,
  disabled,
}: {
  mappings: SCIMGroupRoleMapping[];
  mapping: SCIMGroupRoleMapping;
  setMapping: (mapping: SCIMGroupRoleMapping) => void;
  onSave: () => void;
  disabled: boolean;
}) {
  return (
    <Card className="rounded-lg">
      <CardHeader>
        <CardTitle>Group role mappings</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="grid gap-2 md:grid-cols-3">
          <Input
            placeholder="External group ID"
            value={mapping.externalGroupId}
            onChange={(event) => setMapping({ ...mapping, externalGroupId: event.target.value })}
          />
          <Input
            placeholder="Display name"
            value={mapping.displayName}
            onChange={(event) => setMapping({ ...mapping, displayName: event.target.value })}
          />
          <Input
            placeholder="Role ID"
            value={mapping.roleId}
            onChange={(event) => setMapping({ ...mapping, roleId: event.target.value })}
          />
        </div>
        <Button onClick={onSave} disabled={disabled}>
          <SaveIcon />
          Save mapping
        </Button>
        <SimpleTable
          headers={["Group", "Display name", "Role ID"]}
          rows={mappings.map((item) => [item.externalGroupId, item.displayName, item.roleId])}
        />
      </CardContent>
    </Card>
  );
}

function AuditTable({ records }: { records: ProvisioningAuditRecord[] }) {
  return (
    <Card className="rounded-lg">
      <CardHeader>
        <CardTitle>Provisioning audit</CardTitle>
      </CardHeader>
      <CardContent>
        <SimpleTable
          headers={["Action", "Resource", "External ID", "Status"]}
          rows={records.map((item) => [
            item.action,
            item.resourceType,
            item.externalId || "-",
            item.status,
          ])}
        />
      </CardContent>
    </Card>
  );
}

function PoliciesTab({ organizationId }: { organizationId: string }) {
  const queryClient = useQueryClient();
  const policiesQuery = useQuery(queries.organization.accessPolicies(organizationId));
  const [policy, setPolicy] = useState<AccessPolicy>(emptyPolicy);
  const [conditionsText, setConditionsText] = useState("");

  const saveMutation = useMutation({
    mutationFn: async (value: AccessPolicy) =>
      value.id
        ? apiService.organizationService.updateAccessPolicy(organizationId, value)
        : apiService.organizationService.createAccessPolicy(organizationId, value),
    onSuccess: async () => {
      toast.success("Access policy saved");
      setPolicy(emptyPolicy);
      setConditionsText("");
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
    setConditionsText(formatMap(item.conditions));
  };

  const savePolicy = () => {
    saveMutation.mutate({ ...policy, conditions: parseMap(conditionsText) });
  };

  return (
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div className="space-y-3">
        {(policiesQuery.data ?? []).map((item) => (
          <Card key={item.id} className="rounded-lg">
            <CardContent className="flex flex-col gap-3 p-4 md:flex-row md:items-center md:justify-between">
              <div className="space-y-2">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="font-medium">{item.name}</span>
                  <Badge variant={item.effect === "allow" ? "active" : "inactive"}>
                    {item.effect}
                  </Badge>
                  <Badge variant={item.enabled ? "info" : "outline"}>
                    Priority {item.priority}
                  </Badge>
                </div>
                <div className="text-xs text-muted-foreground">
                  {item.resource}:{item.operation}
                </div>
              </div>
              <div className="flex gap-2">
                <Button variant="outline" onClick={() => editPolicy(item)}>
                  Edit
                </Button>
                <Button variant="destructive" onClick={() => deleteMutation.mutate(item.id)}>
                  <Trash2Icon />
                  Delete
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
        {policiesQuery.data?.length === 0 && <EmptyState label="No access policies configured." />}
      </div>
      <Card className="rounded-lg">
        <CardHeader>
          <CardTitle>{policy.id ? "Edit policy" : "Add policy"}</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          <Field label="Name">
            <Input
              value={policy.name}
              onChange={(event) => setPolicy({ ...policy, name: event.target.value })}
            />
          </Field>
          <Field label="Resource">
            <Input
              value={policy.resource}
              onChange={(event) => setPolicy({ ...policy, resource: event.target.value })}
            />
          </Field>
          <Field label="Operation">
            <Input
              value={policy.operation}
              onChange={(event) => setPolicy({ ...policy, operation: event.target.value })}
            />
          </Field>
          <Field label="Effect">
            <select
              className="h-7 rounded-md border border-input bg-muted px-2 text-sm"
              value={policy.effect}
              onChange={(event) =>
                setPolicy({ ...policy, effect: event.target.value as AccessPolicy["effect"] })
              }
            >
              <option value="deny">Deny</option>
              <option value="allow">Allow</option>
            </select>
          </Field>
          <Field label="Priority">
            <Input
              type="number"
              value={policy.priority}
              onChange={(event) =>
                setPolicy({ ...policy, priority: Number(event.target.value || 0) })
              }
            />
          </Field>
          <Field label="Conditions">
            <Textarea
              value={conditionsText}
              minRows={4}
              onChange={(event) => setConditionsText(event.target.value)}
            />
          </Field>
          <ToggleRow
            label="Enabled"
            checked={policy.enabled}
            onCheckedChange={(enabled) => setPolicy({ ...policy, enabled })}
          />
          <Button className="w-full" onClick={savePolicy} disabled={saveMutation.isPending}>
            <SaveIcon />
            Save policy
          </Button>
        </CardContent>
      </Card>
    </div>
  );
}

function ActivityTab({ organizationId }: { organizationId: string }) {
  const authEventsQuery = useQuery(queries.organization.authEvents(organizationId));
  const riskQuery = useQuery(queries.organization.riskDecisions(organizationId));
  const externalIdentityQuery = useQuery(queries.organization.externalIdentities(organizationId));
  const mfaQuery = useQuery(queries.organization.mfaAuthenticators(organizationId));

  return (
    <div className="grid gap-4 xl:grid-cols-2">
      <ActivityCard
        title="Auth events"
        headers={["Provider", "Outcome", "Risk", "When"]}
        rows={(authEventsQuery.data ?? []).map((item: AuthEvent) => [
          item.provider,
          item.outcome,
          item.riskOutcome,
          formatUnix(item.occurredAt),
        ])}
      />
      <ActivityCard
        title="Risk decisions"
        headers={["Outcome", "Reason", "Signals", "When"]}
        rows={(riskQuery.data ?? []).map((item: RiskDecision) => [
          item.outcome,
          item.reason || "-",
          item.signals.join(", ") || "-",
          formatUnix(item.createdAt),
        ])}
      />
      <ActivityCard
        title="External identities"
        headers={["Email", "Username", "Subject", "Last login"]}
        rows={(externalIdentityQuery.data ?? []).map((item: ExternalIdentity) => [
          item.externalEmail || "-",
          item.externalUsername || "-",
          item.externalSubject,
          item.lastLoginAt ? formatUnix(item.lastLoginAt) : "-",
        ])}
      />
      <ActivityCard
        title="MFA authenticators"
        headers={["Name", "Type", "Status", "Last used"]}
        rows={(mfaQuery.data ?? []).map((item: MFAAuthenticator) => [
          item.name,
          item.type,
          item.enabled ? "enabled" : "disabled",
          item.lastUsedAt ? formatUnix(item.lastUsedAt) : "-",
        ])}
      />
    </div>
  );
}

function ActivityCard({
  title,
  headers,
  rows,
}: {
  title: string;
  headers: string[];
  rows: string[][];
}) {
  return (
    <Card className="rounded-lg">
      <CardHeader>
        <CardTitle>{title}</CardTitle>
      </CardHeader>
      <CardContent>
        <SimpleTable headers={headers} rows={rows} />
      </CardContent>
    </Card>
  );
}

function SimpleTable({ headers, rows }: { headers: string[]; rows: string[][] }) {
  if (rows.length === 0) {
    return <EmptyState label="No records found." />;
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          {headers.map((header) => (
            <TableHead key={header}>{header}</TableHead>
          ))}
        </TableRow>
      </TableHeader>
      <TableBody>
        {rows.map((row, rowIndex) => (
          <TableRow key={`${row.join(":")}-${rowIndex}`}>
            {row.map((cell, cellIndex) => (
              <TableCell key={`${cell}-${cellIndex}`} className="max-w-60 truncate">
                {cell}
              </TableCell>
            ))}
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

function ProviderLogo({ name }: { name: string }) {
  const lowerName = name.toLowerCase();
  if (lowerName.includes("entra") || lowerName.includes("microsoft")) {
    return <EntraLogo className="size-5" />;
  }
  if (lowerName.includes("okta")) {
    return <OktaLogo className="h-5 w-auto" />;
  }
  return <KeyRoundIcon className="size-5 text-primary" />;
}

function ToggleRow({
  label,
  checked,
  onCheckedChange,
}: {
  label: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <label className="flex items-center justify-between gap-3 rounded-md border bg-muted/30 px-3 py-2 text-sm">
      <span>{label}</span>
      <Switch checked={checked} onCheckedChange={onCheckedChange} />
    </label>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="grid gap-1 text-sm">
      <span className="font-medium">{label}</span>
      {children}
    </label>
  );
}

function EmptyState({ label }: { label: string }) {
  return (
    <div className="rounded-md border border-dashed p-4 text-sm text-muted-foreground">{label}</div>
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

function parseMap(value: string) {
  return Object.fromEntries(
    value
      .split("\n")
      .map((line) => line.trim())
      .filter(Boolean)
      .map((line) => {
        const [key, ...rest] = line.split("=");
        return [key.trim(), rest.join("=").trim()];
      })
      .filter(([key]) => key),
  );
}

function formatMap(value: Record<string, string>) {
  return Object.entries(value)
    .map(([key, mapValue]) => `${key}=${mapValue}`)
    .join("\n");
}

function formatUnix(value: number) {
  if (!value) {
    return "-";
  }
  return new Date(value * 1000).toLocaleString();
}
