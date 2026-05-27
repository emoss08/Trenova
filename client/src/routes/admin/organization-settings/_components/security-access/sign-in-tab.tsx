import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@/components/ui/form";
import {
  formatIdentityProviderName,
  parseCommaSeparatedList,
  parseWhitespaceSeparatedList,
} from "@/lib/utils";
import { apiService } from "@/services/api";
import type { IdentityProvider, IdentityProviderFormValues } from "@/types/iam";
import { identityProviderCreateFormSchema, identityProviderFormSchema } from "@/types/iam";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { KeyRoundIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { memo, useCallback, useMemo, useState } from "react";
import { type Resolver, useForm, useFormContext } from "react-hook-form";
import { toast } from "sonner";
import {
  ChipArrayField,
  ConsoleToolbar,
  EmptyState,
  ErrorState,
  MetaLine,
  ProviderLogo,
  RowSkeleton,
} from "./shared";
import {
  emptyProvider,
  identityProviderEndpoint,
  identityProviderQueryKey,
  providerPresets,
  toIdentityProvider,
} from "./utils";

type IdentityProviderPanelMode = "create" | "edit";
type IdentityProviderRecord = IdentityProvider & Record<string, unknown>;

export function SignInTab({ organizationId }: { organizationId: string }) {
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
        formatIdentityProviderName(provider.name),
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

  const { mutate: deleteProvider, isPending: isDeletingProvider } = useMutation({
    mutationFn: async (providerId: string) =>
      apiService.organizationService.deleteIdentityProvider(organizationId, providerId),
    onSuccess: async () => {
      toast.success("Identity provider removed");
      await queryClient.invalidateQueries({
        queryKey: [identityProviderQueryKey(organizationId)],
      });
    },
  });

  const openCreatePanel = useCallback(() => {
    setPanelMode("create");
    setEditingProvider(null);
    setPanelOpen(true);
  }, []);

  const openEditPanel = useCallback((provider: IdentityProvider) => {
    setPanelMode("edit");
    setEditingProvider(provider);
    setPanelOpen(true);
  }, []);

  const handleDeleteProvider = useCallback(
    (providerId: string) => deleteProvider(providerId),
    [deleteProvider],
  );

  return (
    <div className="space-y-3">
      <ConsoleToolbar
        title="Identity providers"
        description="OIDC sign-in providers available to this organization."
        search={search}
        onSearchChange={setSearch}
        searchPlaceholder="Search providers, domains, or issuer"
        action={
          <Button size="sm" onClick={openCreatePanel}>
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
              onEditProvider={openEditPanel}
              onDeleteProvider={handleDeleteProvider}
              isDeleting={isDeletingProvider}
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

const ProviderRow = memo(function ProviderRow({
  provider,
  onEditProvider,
  onDeleteProvider,
  isDeleting,
}: {
  provider: IdentityProvider;
  onEditProvider: (provider: IdentityProvider) => void;
  onDeleteProvider: (providerId: string) => void;
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
                {formatIdentityProviderName(provider.name) || "OIDC provider"}
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
        <Button size="sm" variant="outline" onClick={() => onEditProvider(provider)}>
          Edit
        </Button>
        <Button
          size="sm"
          variant="destructive"
          onClick={() => onDeleteProvider(provider.id)}
          disabled={isDeleting}
        >
          <Trash2Icon />
          Delete
        </Button>
      </div>
    </div>
  );
});

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
              parseValue={parseWhitespaceSeparatedList}
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
              parseValue={parseCommaSeparatedList}
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
