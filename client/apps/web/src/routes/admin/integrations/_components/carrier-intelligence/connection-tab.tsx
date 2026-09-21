import { useCarrierIntelLabels } from "@/components/carrier-intelligence/use-carrier-intel-labels";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { carrierIntelProviderLabel } from "@/lib/carrier-intelligence";
import type { CarrierIntelSettings } from "@/lib/graphql/carrier-intelligence";
import { switchCarrierIntelProvider } from "@/lib/graphql/carrier-intel-settings";
import { queries } from "@/lib/queries";
import type { ConfigFieldSpec } from "@/types/integration";
import { useQueryClient } from "@tanstack/react-query";
import { Badge } from "@trenova/shared/components/ui/badge";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormGroup } from "@trenova/shared/components/ui/form";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { useCallback } from "react";
import { useWatch } from "react-hook-form";
import { toast } from "sonner";
import {
  IntegrationEnabledSwitch,
  SpecIntegrationFields,
  SpecIntegrationFooter,
  SpecIntegrationPrerequisite,
} from "../shared/spec-integration-fields";
import { useSpecIntegrationConfig } from "../shared/use-spec-integration-config";
import { capabilityLabels } from "./carrier-intel-settings-schema";
import {
  CARRIER_INTEL_FALLBACK_ROLE,
  CARRIER_INTEL_ROLE_KEY,
  type CarrierIntelVendor,
} from "./carrier-intelligence-vendors";

const roleOptionLabels: Record<string, string> = {
  primary: "Primary provider",
  fallback: "Fallback only",
};

export type CarrierIntelProviderRole = "primary" | "fallback" | "unused";

export function resolveProviderRole(
  vendor: Pick<CarrierIntelVendor, "integrationType">,
  settings: CarrierIntelSettings | undefined,
): CarrierIntelProviderRole {
  if (settings?.carrierIntelControl.primaryProvider === vendor.integrationType) {
    return "primary";
  }
  if (settings?.carrierIntelControl.fallbackProvider === vendor.integrationType) {
    return "fallback";
  }
  return "unused";
}

export function CarrierIntelConnectionTab({
  vendor,
  open,
  onClose,
  settings,
  settingsLoading,
  canRead,
  canManage,
}: {
  vendor: CarrierIntelVendor;
  open: boolean;
  onClose: () => void;
  settings: CarrierIntelSettings | undefined;
  settingsLoading: boolean;
  canRead: boolean;
  canManage: boolean;
}) {
  const t = useT();
  const queryClient = useQueryClient();

  const invalidateIntel = useCallback(
    () => queryClient.invalidateQueries({ queryKey: queries.carrierIntelSettings._def }),
    [queryClient],
  );

  const { configQuery, response, spec, form, storedByKey, saveMutation, testConnectionMutation } =
    useSpecIntegrationConfig({
      integrationType: vendor.integrationType,
      name: vendor.name,
      open,
      onChanged: invalidateIntel,
    });
  const { control, handleSubmit } = form;

  const secretValue = useWatch({ control, name: `configuration.${vendor.secretKey}` as const });
  const roleValue = useWatch({ control, name: `configuration.${CARRIER_INTEL_ROLE_KEY}` as const });

  const switchMutation = useApiMutation({
    mutationFn: () => switchCarrierIntelProvider(vendor.integrationType),
    resourceName: "Carrier intelligence provider",
    onSuccess: async () => {
      toast.success(t("{0} is now the primary carrier intelligence provider", vendor.name));
      await Promise.all([
        invalidateIntel(),
        queryClient.invalidateQueries({ queryKey: queries.integration.catalog().queryKey }),
      ]);
    },
  });

  const role = resolveProviderRole(vendor, settings);
  const isSandboxKey = Boolean(
    vendor.sandboxKeyPrefix &&
    typeof secretValue === "string" &&
    secretValue.trim().startsWith(vendor.sandboxKeyPrefix),
  );
  const wantsFallback = vendor.supportsFallbackRole && roleValue === CARRIER_INTEL_FALLBACK_ROLE;
  const canBecomePrimary =
    canManage && Boolean(response?.enabled) && role !== "primary" && !wantsFallback;

  const renderOptionLabel = (field: ConfigFieldSpec, option: string) =>
    field.key === CARRIER_INTEL_ROLE_KEY ? t(roleOptionLabels[option] ?? option) : option;

  return (
    <div className="space-y-4">
      {vendor.prerequisite ? (
        <SpecIntegrationPrerequisite>{t(vendor.prerequisite)}</SpecIntegrationPrerequisite>
      ) : null}
      {canRead ? (
        <ProviderStatusCard
          vendor={vendor}
          role={role}
          settings={settings}
          loading={settingsLoading}
          canBecomePrimary={canBecomePrimary}
          onMakePrimary={() => switchMutation.mutate()}
          isSwitching={switchMutation.isPending}
        />
      ) : null}
      <Form onSubmit={handleSubmit((data) => saveMutation.mutateAsync(data))} className="space-y-4">
        <FormGroup cols={1}>
          <IntegrationEnabledSwitch
            control={control}
            id={`${vendor.integrationType}-enabled`}
            label={t("Enable {0}", vendor.name)}
            description={t("Use {0} to vet and monitor carriers and brokers.", vendor.name)}
          />
          {isSandboxKey ? (
            <div className="flex items-center gap-2">
              <Badge variant="warning">{t("Sandbox key")}</Badge>
              <span className="text-muted-foreground text-xs">
                {t("Sandbox keys only return the CarrierOk fixture carriers.")}
              </span>
            </div>
          ) : null}
          {configQuery.isLoading ? (
            <div className="space-y-3">
              <Skeleton className="h-12 w-full" />
              <Skeleton className="h-12 w-full" />
            </div>
          ) : configQuery.isError ? (
            <p className="text-destructive text-sm">
              {t("The connection settings could not be loaded. Close the dialog and try again.")}
            </p>
          ) : (
            <SpecIntegrationFields
              fields={spec}
              control={control}
              storedByKey={storedByKey}
              renderOptionLabel={renderOptionLabel}
            />
          )}
        </FormGroup>
        <SpecIntegrationFooter
          className="mx-0 mb-0 rounded-md"
          onCancel={onClose}
          onTestConnection={() => testConnectionMutation.mutate()}
          isTesting={testConnectionMutation.isPending}
          isSaving={saveMutation.isPending}
          isLoading={configQuery.isLoading}
        />
      </Form>
    </div>
  );
}

function ProviderStatusCard({
  vendor,
  role,
  settings,
  loading,
  canBecomePrimary,
  onMakePrimary,
  isSwitching,
}: {
  vendor: CarrierIntelVendor;
  role: CarrierIntelProviderRole;
  settings: CarrierIntelSettings | undefined;
  loading: boolean;
  canBecomePrimary: boolean;
  onMakePrimary: () => void;
  isSwitching: boolean;
}) {
  const t = useT();
  const labels = useCarrierIntelLabels();

  if (loading) {
    return <Skeleton className="h-24 w-full" />;
  }

  const provider = settings?.carrierIntelProvider;
  const primaryProvider = settings?.carrierIntelControl.primaryProvider;
  const primaryName = primaryProvider ? carrierIntelProviderLabel(primaryProvider) : "";
  const isActivePrimary = role === "primary" && provider?.provider === vendor.integrationType;

  return (
    <div className="border-border bg-muted/30 space-y-3 rounded-md border p-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium">{t("Provider role")}</span>
          {role === "primary" ? (
            <Badge variant="success">{t("Primary provider")}</Badge>
          ) : role === "fallback" ? (
            <Badge variant="info">{t("Fallback provider")}</Badge>
          ) : (
            <Badge variant="neutral" appearance="outline">{t("Not in use")}</Badge>
          )}
          {isActivePrimary && provider && !provider.configured ? (
            <Badge variant="warning">{t("Needs setup")}</Badge>
          ) : null}
        </div>
        {canBecomePrimary ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={onMakePrimary}
            isLoading={isSwitching}
            loadingText={t("Switching...")}
          >
            {t("Make primary")}
          </Button>
        ) : null}
      </div>
      {role !== "primary" && primaryName ? (
        <p className="text-muted-foreground text-xs">
          {t("{0} is the primary carrier intelligence provider.", primaryName)}
        </p>
      ) : null}
      {isActivePrimary && provider ? (
        <div className="space-y-2">
          <BadgeList
            label={t("Capabilities")}
            values={provider.capabilities.map((capability) =>
              t(capabilityLabels[capability] ?? capability),
            )}
            emptyText={t("No capabilities reported")}
          />
          <BadgeList
            label={t("Data sections")}
            values={provider.sections.map((section) => labels.section[section] ?? section)}
            emptyText={t("No data sections reported")}
          />
        </div>
      ) : (
        <p className="text-muted-foreground text-xs">
          {t("Capabilities are listed once {0} is the primary provider.", vendor.name)}
        </p>
      )}
    </div>
  );
}

function BadgeList({
  label,
  values,
  emptyText,
}: {
  label: string;
  values: string[];
  emptyText: string;
}) {
  return (
    <div className="space-y-1">
      <p className="text-muted-foreground text-xs font-medium">{label}</p>
      {values.length === 0 ? (
        <p className="text-muted-foreground text-xs">{emptyText}</p>
      ) : (
        <div className="flex flex-wrap gap-1">
          {values.map((value) => (
            <Badge key={value} variant="neutral">
              {value}
            </Badge>
          ))}
        </div>
      )}
    </div>
  );
}
