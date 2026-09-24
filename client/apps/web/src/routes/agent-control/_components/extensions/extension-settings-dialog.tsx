import { ExternalLink } from "@/components/link";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import {
  IntegrationEnabledSwitch,
  SpecIntegrationFields,
} from "@/routes/admin/integrations/_components/shared/spec-integration-fields";
import { apiService } from "@/services/api";
import type {
  AgentExtensionAvailability,
  AgentExtensionCatalogItem,
} from "@/types/agent-extension";
import type { UpdateIntegrationConfigRequest } from "@/types/integration";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { BrandLogo } from "@trenova/shared/components/brand-logo";
import { Alert, AlertDescription } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Label } from "@trenova/shared/components/ui/label";
import { SegmentedControl } from "@trenova/shared/components/ui/segmented-control";
import { Skeleton } from "@trenova/shared/components/ui/skeleton";
import { useT } from "@trenova/shared/i18n/use-t";
import { CheckIcon, ShieldCheckIcon } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useForm } from "react-hook-form";
import { toast } from "sonner";

type ExtensionSettingsDialogProps = {
  extension: AgentExtensionCatalogItem | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  canUpdate: boolean;
};

const SEARCH_DEPTH_LABELS: Record<string, string> = {
  auto: "Auto",
  fast: "Fast",
  deep: "Deep",
};

/**
 * Everything an administrator decides about one extension: whether it is on,
 * which agents get its tools, and the credentials and limits it runs on. A
 * saved key is never shown again; leaving the field blank keeps it.
 */
export function ExtensionSettingsDialog({
  extension,
  open,
  onOpenChange,
  canUpdate,
}: ExtensionSettingsDialogProps) {
  const t = useT();
  const queryClient = useQueryClient();
  const type = extension?.type ?? "";

  const configQuery = useQuery({
    ...queries.agentExtension.config(type),
    enabled: open && type !== "",
  });
  const response = configQuery.data;
  const spec = useMemo(() => response?.spec ?? [], [response]);
  const storedByKey = useMemo(
    () => new Map(response?.fields.map((field) => [field.key, field.hasValue]) ?? []),
    [response],
  );

  const form = useForm<UpdateIntegrationConfigRequest>({
    defaultValues: { enabled: false, configuration: {} },
  });
  const { control, handleSubmit, reset } = form;
  const [chosenAvailability, setAvailability] = useState<AgentExtensionAvailability | null>(null);
  const availability = chosenAvailability ?? response?.availability ?? "SelectedAgents";

  useEffect(() => {
    if (!open || !response) {
      return;
    }

    const valueByKey = new Map(response.fields.map((field) => [field.key, field.value ?? ""]));
    const configuration: Record<string, string> = {};
    for (const field of response.spec) {
      configuration[field.key] = field.sensitive
        ? ""
        : (valueByKey.get(field.key) ?? field.default ?? "");
    }

    reset({ enabled: response.enabled, configuration });
  }, [open, response, reset]);

  const handleOpenChange = (next: boolean) => {
    if (!next) {
      setAvailability(null);
    }
    onOpenChange(next);
  };

  const invalidate = async () => {
    await queryClient.invalidateQueries({ queryKey: queries.agentExtension._def });
  };

  const saveMutation = useApiMutation({
    mutationFn: (values: UpdateIntegrationConfigRequest) =>
      apiService.agentExtensionService.updateConfig(type, {
        enabled: values.enabled,
        availability,
        configuration: values.configuration,
        version: response?.version ?? 0,
      }),
    form,
    resourceName: "Extension",
    onSuccess: async (saved) => {
      toast.success(
        saved.enabled
          ? t("{0} is on for your agents", extension?.name ?? "")
          : t("{0} settings saved", extension?.name ?? ""),
      );
      await invalidate();
      handleOpenChange(false);
    },
  });

  const testMutation = useApiMutation({
    mutationFn: () => apiService.agentExtensionService.test(type),
    resourceName: "Extension",
    onSuccess: async (result) => {
      toast.success(t("Connection works"), { description: result.message });
      await invalidate();
    },
  });

  const hasStoredKey = spec.some((field) => field.sensitive && storedByKey.get(field.key));
  const availabilityItems = [
    {
      value: "SelectedAgents" as const,
      label: t("Agents you choose"),
      caption: t("Add its tools to an agent in Agents"),
    },
    {
      value: "AllAgents" as const,
      label: t("Every agent"),
      caption: t("Including the assistant"),
    },
  ];

  if (!extension) {
    return null;
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent size="lg" className="max-h-[calc(100dvh-2rem)] overflow-y-auto">
        <DialogHeader>
          <div className="flex items-center gap-3">
            <BrandLogo domain={extension.brandDomain} name={extension.vendor} size={32} />
            <div className="flex min-w-0 flex-col">
              <DialogTitle>{extension.name}</DialogTitle>
              <span className="text-muted-foreground text-xs">
                {t("By {0} · {1}", extension.vendor, extension.categoryLabel)}
              </span>
            </div>
          </div>
          <DialogDescription>
            {extension.description}{" "}
            <ExternalLink href={extension.docsUrl}>{t("Documentation")}</ExternalLink>
          </DialogDescription>
        </DialogHeader>

        <ul className="grid gap-1.5 sm:grid-cols-2">
          {extension.capabilities.map((capability) => (
            <li key={capability} className="text-muted-foreground flex gap-2 text-xs">
              <CheckIcon className="text-success mt-0.5 size-3.5 shrink-0" aria-hidden />
              <span>{capability}</span>
            </li>
          ))}
        </ul>

        <Alert size="sm" variant="info">
          <ShieldCheckIcon aria-hidden />
          <AlertDescription>{extension.dataNotice}</AlertDescription>
        </Alert>

        <Form
          onSubmit={handleSubmit((values) => saveMutation.mutateAsync(values))}
          className="space-y-4"
        >
          <fieldset disabled={!canUpdate} className="contents">
            <FormGroup cols={1}>
              <IntegrationEnabledSwitch
                control={control}
                id={`extension-${extension.type}-enabled`}
                label={t("Turn on {0}", extension.name)}
                description={t(
                  "Agents can use its tools only while it is on and its API key is saved.",
                )}
              />
              <FormControl cols="full">
                <div className="flex flex-col gap-2">
                  <Label>{t("Available to")}</Label>
                  <SegmentedControl
                    items={availabilityItems}
                    value={availability}
                    onValueChange={setAvailability}
                    fullWidth
                    aria-label={t("Available to")}
                  />
                </div>
              </FormControl>
              {configQuery.isLoading ? (
                <div className="space-y-3">
                  <Skeleton className="h-12 w-full" />
                  <Skeleton className="h-12 w-full" />
                </div>
              ) : configQuery.isError ? (
                <p className="text-danger text-sm">
                  {t("The settings could not be loaded. Close the dialog and try again.")}
                </p>
              ) : (
                <SpecIntegrationFields
                  fields={spec}
                  control={control}
                  storedByKey={storedByKey}
                  renderOptionLabel={(field, option) =>
                    field.key === "searchType" ? t(SEARCH_DEPTH_LABELS[option] ?? option) : option
                  }
                />
              )}
            </FormGroup>
          </fieldset>

          <DialogFooter className="mx-0 mb-0 rounded-md sm:justify-between">
            {extension.supportsTestConnect && canUpdate ? (
              <Button
                type="button"
                variant="ghost"
                onClick={() => testMutation.mutate()}
                isLoading={testMutation.isPending}
                loadingText={t("Testing...")}
                disabled={!hasStoredKey || configQuery.isLoading || saveMutation.isPending}
                title={hasStoredKey ? undefined : t("Save an API key first")}
              >
                {t("Test connection")}
              </Button>
            ) : (
              <span />
            )}
            <div className="flex flex-col-reverse gap-2 sm:flex-row">
              <Button type="button" variant="outline" onClick={() => handleOpenChange(false)}>
                {canUpdate ? t("Cancel") : t("Close")}
              </Button>
              {canUpdate && (
                <Button
                  type="submit"
                  isLoading={saveMutation.isPending}
                  loadingText={t("Saving...")}
                  disabled={configQuery.isLoading || configQuery.isError}
                >
                  {t("Save changes")}
                </Button>
              )}
            </div>
          </DialogFooter>
        </Form>
      </DialogContent>
    </Dialog>
  );
}
