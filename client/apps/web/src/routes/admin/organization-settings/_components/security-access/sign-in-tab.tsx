import { useT } from "@trenova/shared/i18n/use-t";
import { InputField } from "@/components/fields/input-field";
import { SwitchField } from "@/components/fields/switch-field";
import { FormCreatePanel } from "@/components/form-create-panel";
import { FormEditPanel } from "@/components/form-edit-panel";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { FormControl, FormGroup, FormSection } from "@trenova/shared/components/ui/form";
import {
  identityProviderPanelSearchParamsParser,
  identityProviderSearchParser,
} from "@/hooks/use-organization-setting-state";
import {
  formatIdentityProviderName,
  parseCommaSeparatedList,
  parseWhitespaceSeparatedList,
} from "@trenova/shared/lib/utils";
import { apiService } from "@/services/api";
import type { IdentityProvider, IdentityProviderFormValues } from "@trenova/shared/types/iam";
import {
  identityProviderCreateFormSchema,
  identityProviderFormSchema,
} from "@trenova/shared/types/iam";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { KeyRoundIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { useQueryState, useQueryStates } from "nuqs";
import { memo, useCallback, useEffect, useMemo } from "react";
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
type IdentityProviderPanelParams = {
  editingProvider: string | null;
  panelMode: IdentityProviderPanelMode;
  panelOpen: boolean;
};
type SetIdentityProviderPanelParams = (values: {
  editingProvider?: string | null;
  panelMode?: IdentityProviderPanelMode | null;
  panelOpen?: boolean | null;
}) => Promise<URLSearchParams>;

export function SignInTab({ organizationId }: { organizationId: string }) {
  const [panelParams, setPanelParams] = useQueryStates(identityProviderPanelSearchParamsParser);
  const providersQuery = useQuery({
    queryKey: [identityProviderQueryKey(organizationId)],
    queryFn: async () => apiService.organizationService.listIdentityProviders(organizationId),
  });
  const providers = useMemo(() => providersQuery.data ?? [], [providersQuery.data]);

  const openCreatePanel = useCallback(() => {
    void setPanelParams({
      panelMode: "create",
      editingProvider: null,
      panelOpen: true,
    });
  }, [setPanelParams]);

  const openEditPanel = useCallback(
    (provider: IdentityProvider) => {
      void setPanelParams({
        panelMode: "edit",
        editingProvider: provider.id,
        panelOpen: true,
      });
    },
    [setPanelParams],
  );

  return (
    <div className="space-y-3">
      <IdentityProviderListSection
        organizationId={organizationId}
        providers={providers}
        isLoading={providersQuery.isLoading}
        isError={providersQuery.isError}
        onCreateProvider={openCreatePanel}
        onEditProvider={openEditPanel}
      />
      <IdentityProviderPanelController
        organizationId={organizationId}
        providers={providers}
        providersLoaded={providersQuery.isSuccess}
        panelParams={panelParams}
        setPanelParams={setPanelParams}
      />
    </div>
  );
}

const IdentityProviderListSection = memo(function IdentityProviderListSection({
  organizationId,
  providers,
  isLoading,
  isError,
  onCreateProvider,
  onEditProvider,
}: {
  organizationId: string;
  providers: IdentityProvider[];
  isLoading: boolean;
  isError: boolean;
  onCreateProvider: () => void;
  onEditProvider: (provider: IdentityProvider) => void;
}) {
  const t = useT();

  const queryClient = useQueryClient();
  const [search, setSearch] = useQueryState("search", identityProviderSearchParser);
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
      toast.success(t("Identity provider removed"));
      await queryClient.invalidateQueries({
        queryKey: [identityProviderQueryKey(organizationId)],
      });
    },
  });

  const handleSearchChange = useCallback(
    (value: string) => {
      void setSearch(value || null);
    },
    [setSearch],
  );

  const handleDeleteProvider = useCallback(
    (providerId: string) => deleteProvider(providerId),
    [deleteProvider],
  );

  return (
    <div className="space-y-3">
      <ConsoleToolbar
        title={t("Identity providers")}
        description={t("OIDC sign-in providers available to this organization.")}
        search={search}
        onSearchChange={handleSearchChange}
        searchPlaceholder={t("Search providers, domains, or issuer")}
        action={
          <Button size="sm" onClick={onCreateProvider}>
            <PlusIcon />
            {t("Add provider")}
          </Button>
        }
      />
      {isLoading ? (
        <RowSkeleton rows={3} />
      ) : isError ? (
        <ErrorState label={t("Identity providers could not be loaded.")} />
      ) : filteredProviders.length > 0 ? (
        <div className="bg-background overflow-hidden rounded-lg border">
          {filteredProviders.map((provider) => (
            <ProviderRow
              key={provider.id}
              provider={provider}
              onEditProvider={onEditProvider}
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
    </div>
  );
});

function IdentityProviderPanelController({
  organizationId,
  providers,
  providersLoaded,
  panelParams,
  setPanelParams,
}: {
  organizationId: string;
  providers: IdentityProvider[];
  providersLoaded: boolean;
  panelParams: IdentityProviderPanelParams;
  setPanelParams: SetIdentityProviderPanelParams;
}) {
  const t = useT();

  const { panelMode, panelOpen, editingProvider } = panelParams;
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
  const selectedProvider = useMemo(
    () => providers.find((provider) => provider.id === editingProvider) ?? null,
    [editingProvider, providers],
  );

  const handlePanelOpenChange = useCallback(
    (open: boolean) => {
      if (open) {
        void setPanelParams({ panelOpen: true });
        return;
      }

      void setPanelParams({
        panelOpen: null,
        panelMode: null,
        editingProvider: null,
      });
    },
    [setPanelParams],
  );

  useEffect(() => {
    if (
      providersLoaded &&
      panelOpen &&
      panelMode === "edit" &&
      editingProvider &&
      !selectedProvider
    ) {
      handlePanelOpenChange(false);
    }
  }, [
    editingProvider,
    handlePanelOpenChange,
    panelMode,
    panelOpen,
    providersLoaded,
    selectedProvider,
  ]);

  return panelMode === "edit" ? (
    <FormEditPanel<IdentityProviderFormValues, IdentityProviderRecord>
      open={panelOpen}
      onOpenChange={handlePanelOpenChange}
      row={(selectedProvider as IdentityProviderRecord | null) ?? null}
      form={editForm}
      url={identityProviderEndpoint(organizationId)}
      queryKey={identityProviderQueryKey(organizationId)}
      title={t("Identity Provider")}
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
      onOpenChange={handlePanelOpenChange}
      form={createForm}
      url={identityProviderEndpoint(organizationId)}
      queryKey={identityProviderQueryKey(organizationId)}
      title={t("Identity Provider")}
      description={t(
        "Configure OIDC sign-in details, allowed domains, scopes, and enforcement settings.",
      )}
      size="lg"
      formComponent={<IdentityProviderForm mode="create" />}
      mutationFn={async (values) =>
        apiService.organizationService.createIdentityProvider(
          organizationId,
          toIdentityProvider(values),
        )
      }
    />
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
  const t = useT();

  return (
    <div className="grid gap-3 border-b p-3 last:border-b-0 lg:grid-cols-[minmax(0,1fr)_180px] lg:items-center">
      <div className="min-w-0 space-y-2">
        <div className="flex flex-wrap items-center gap-2">
          <span className="bg-muted/30 flex size-9 items-center justify-center rounded-md border">
            <ProviderLogo name={provider.name} />
          </span>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <span className="font-medium">
                {formatIdentityProviderName(provider.name) || t("OIDC provider")}
              </span>
              <Badge variant={provider.enabled ? "active" : "inactive"}>
                {provider.enabled ? t("Enabled") : t("Disabled")}
              </Badge>
              {provider.enforceSso && <Badge variant="warning">{t("SSO enforced")}</Badge>}
              {provider.autoProvision && <Badge variant="info">{t("Auto-provision")}</Badge>}
            </div>
            <div className="text-muted-foreground truncate text-xs">
              {provider.slug || t("No slug")}
            </div>
          </div>
        </div>
        <div className="text-muted-foreground grid gap-2 text-xs md:grid-cols-2">
          <MetaLine label={t("Issuer")} value={provider.oidcIssuerUrl || "-"} />
          <MetaLine label={t("Redirect URI")} value={provider.oidcRedirectUrl || "-"} />
          <MetaLine
            label={t("Domains")}
            value={provider.allowedDomains.join(", ") || "Any domain"}
          />
          <MetaLine
            label={t("Scopes")}
            value={provider.oidcScopes.join(" ") || "Default OIDC scopes"}
          />
        </div>
      </div>
      <div className="flex justify-end gap-2">
        <Button size="sm" variant="outline" onClick={() => onEditProvider(provider)}>
          {t("Edit")}
        </Button>
        <Button
          size="sm"
          variant="destructive"
          onClick={() => onDeleteProvider(provider.id)}
          disabled={isDeleting}
        >
          <Trash2Icon />
          {t("Delete")}
        </Button>
      </div>
    </div>
  );
});

function IdentityProviderForm({ mode }: { mode: IdentityProviderPanelMode }) {
  const t = useT();

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
        title={t("Quick start")}
        description={t("Start with common provider defaults, then adjust tenant-specific values.")}
      >
        <div className="grid gap-2 sm:grid-cols-2">
          {providerPresets.map((preset) => (
            <button
              key={preset.slug}
              type="button"
              className="bg-background hover:bg-muted/50 flex items-center gap-2 rounded-md border px-3 py-2 text-left text-sm transition-colors"
              onClick={() => applyPreset(preset.slug)}
            >
              <ProviderLogo name={preset.name} />
              <span className="font-medium">{t(preset.label)}</span>
            </button>
          ))}
        </div>
      </FormSection>

      <FormSection
        title={t("Provider")}
        description={t(
          "Name the sign-in provider and configure the identifier used by hosted login flows.",
        )}
      >
        <FormGroup cols={2}>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="name"
              label={t("Name")}
              placeholder={t("Microsoft Entra ID")}
              description={t("Displayed to administrators and users in sign-in flows.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="slug"
              label={t("Slug")}
              placeholder="entra-id"
              description={t("Stable URL-safe provider key used internally for routing.")}
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("OIDC application")}
        description={t("Copy these values from the provider application registration.")}
      >
        <FormGroup cols={2}>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="oidcIssuerUrl"
              label={t("Issuer URL")}
              placeholder="https://login.microsoftonline.com/{tenant-id}/v2.0"
              description={t("OIDC issuer metadata URL for token validation.")}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              control={control}
              rules={{ required: true }}
              name="oidcRedirectUrl"
              label={t("Redirect URI")}
              placeholder="https://app.example.com/auth/callback"
              description={t("Callback URI registered in the provider application.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={{ required: true }}
              name="oidcClientId"
              label={t("Client ID")}
              placeholder={t("Application client ID")}
              description={t("Public OIDC client identifier.")}
            />
          </FormControl>
          <FormControl>
            <InputField
              control={control}
              rules={mode === "create" ? { required: true } : undefined}
              name="oidcClientSecret"
              label={t("Client secret")}
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
              label={t("OIDC scopes")}
              placeholder={t("openid email profile")}
              description={t("Space-separated scopes requested during authentication.")}
              parseValue={parseWhitespaceSeparatedList}
              formatValue={(value) => value.join(" ")}
              required
            />
          </FormControl>
        </FormGroup>
      </FormSection>

      <FormSection
        title={t("Access boundaries")}
        description={t("Limit domains and tune how federated users are handled after sign-in.")}
      >
        <FormGroup cols={1}>
          <FormControl>
            <ChipArrayField
              control={control}
              name="allowedDomains"
              label={t("Allowed domains")}
              placeholder={t("example.com, subsidiary.com")}
              description={t(
                "Comma-separated domains allowed to authenticate. Leave blank to allow any domain.",
              )}
              parseValue={parseCommaSeparatedList}
              formatValue={(value) => value.join(", ")}
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="enabled"
              label={t("Enabled")}
              description={t("Allow this provider to appear in sign-in flows.")}
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="enforceSso"
              label={t("Enforce SSO")}
              description={t("Require users to authenticate with a federated provider.")}
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="autoProvision"
              label={t("Auto-provision users")}
              description={t("Create user records after successful provider authentication.")}
              outlined
            />
          </FormControl>
          <FormControl>
            <SwitchField
              control={control}
              name="allowFederatedMfa"
              label={t("Trust federated MFA")}
              description={t("Accept MFA claims from the provider when risk policy allows it.")}
              outlined
            />
          </FormControl>
        </FormGroup>
      </FormSection>
    </div>
  );
}
