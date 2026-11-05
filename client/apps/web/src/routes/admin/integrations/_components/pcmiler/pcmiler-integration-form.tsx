const pcMilerLogoLight = "/integrations/logos/pc-miler-logo-light.png";
const pcMilerLogoDark = "/integrations/logos/pc-miler-logo-dark.svg";

import { useT } from "@trenova/shared/i18n/use-t";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { InputField } from "@/components/fields/input-field";
import { ExternalLink } from "@/components/link";
import { Button } from "@trenova/shared/components/ui/button";
import { DialogFooter } from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Label } from "@trenova/shared/components/ui/label";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { integration } from "@/lib/queries/integration";
import { integrationService } from "@/services/integration";
import type { UpdateIntegrationConfigRequest } from "@/types/integration";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { toast } from "@trenova/shared/components/ui/toast";
import { IntegrationConnectHeader } from "../shared/integration-connect-header";

const INTEGRATION_TYPE = "PCMiler";

export function PCMilerIntegrationForm({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT();

  const queryClient = useQueryClient();
  const configQuery = useQuery({ ...integration.config(INTEGRATION_TYPE), enabled: open });
  const form = useForm<UpdateIntegrationConfigRequest>({
    defaultValues: defaultValues(),
  });
  const { control, reset, handleSubmit } = form;

  const response = configQuery.data;
  const hasApiKey =
    response?.fields?.some((field) => field.key === "apiKey" && field.hasValue) ?? false;

  useEffect(() => {
    if (!open || !response) {
      return;
    }
    const valueByKey = new Map(response.fields.map((field) => [field.key, field.value ?? ""]));
    const next = defaultValues();
    for (const key of Object.keys(next.configuration)) {
      next.configuration[key] = valueByKey.get(key) || next.configuration[key] || "";
    }
    next.configuration.apiKey = "";
    next.enabled = response.enabled;
    reset(next);
  }, [open, response, reset]);

  const saveMutation = useApiMutation({
    mutationFn: (payload: UpdateIntegrationConfigRequest) =>
      integrationService.updateConfig(INTEGRATION_TYPE, payload),
    form,
    resourceName: "PC*Miler configuration",
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: integration.config(INTEGRATION_TYPE).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: integration.catalog().queryKey }),
      ]);
    },
  });

  const testConnectionMutation = useMutation({
    mutationFn: () => integrationService.testConnection(INTEGRATION_TYPE),
    onSuccess: async () => {
      toast.success(t("PC*Miler connection successful"));
      await queryClient.invalidateQueries({ queryKey: integration.catalog().queryKey });
    },
    onError: () => toast.error(t("PC*Miler connection test failed")),
  });

  return (
    <div className="space-y-4">
      <PCMilerFormHeader />
      <Form onSubmit={handleSubmit((data) => saveMutation.mutateAsync(data))} className="space-y-4">
        <FormGroup cols={1}>
          <FormControl cols="full">
            <Controller
              name="enabled"
              control={control}
              render={({ field }) => (
                <div className="border-border bg-background flex items-center justify-between rounded-md border p-3">
                  <div>
                    <Label htmlFor="pcmiler-enabled">{t("Enable PC*Miler")}</Label>
                    <p className="text-muted-foreground text-xs">
                      {t("Toggle mileage rating for this business unit.")}
                    </p>
                  </div>
                  <Switch
                    id="pcmiler-enabled"
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </div>
              )}
            />
          </FormControl>
          <FormControl cols="full">
            <SensitiveField
              name="configuration.apiKey"
              control={control}
              label={hasApiKey ? t("API Key (leave blank to keep existing key)") : t("API Key")}
              autoComplete="off"
              placeholder={hasApiKey ? "********" : "Enter your Trimble Maps API key"}
            />
          </FormControl>
          <TextField control={control} name="baseUrl" label={t("Base URL")} />
        </FormGroup>
        <DialogFooter className="sm:justify-between">
          <Button
            type="button"
            variant="ghost"
            onClick={() => testConnectionMutation.mutateAsync()}
            isLoading={testConnectionMutation.isPending}
            loadingText={t("Testing...")}
            disabled={configQuery.isLoading || saveMutation.isPending}
          >
            {t("Test connection")}
          </Button>
          <div className="flex flex-col-reverse gap-2 sm:flex-row">
            <Button type="button" variant="outline" onClick={onClose}>
              {t("Cancel")}
            </Button>
            <Button
              type="submit"
              isLoading={saveMutation.isPending}
              loadingText={t("Saving...")}
              disabled={configQuery.isLoading}
            >
              {t("Save changes")}
            </Button>
          </div>
        </DialogFooter>
      </Form>
    </div>
  );
}

function TextField({
  control,
  name,
  label,
  type,
  placeholder,
}: {
  control: ReturnType<typeof useForm<UpdateIntegrationConfigRequest>>["control"];
  name: string;
  label: string;
  type?: string;
  placeholder?: string;
}) {
  return (
    <FormControl cols="full">
      <InputField
        name={`configuration.${name}`}
        control={control}
        label={label}
        type={type}
        placeholder={placeholder}
      />
    </FormControl>
  );
}

function PCMilerFormHeader() {
  const t = useT();

  return (
    <IntegrationConnectHeader
      name="PC*Miler"
      logo={{ light: pcMilerLogoLight, dark: pcMilerLogoDark }}
      title={t("Connect with PC*Miler")}
      description={
        <>
          {t("Configure mileage rating with")}{" "}
          <ExternalLink href="https://developer.trimblemaps.com/">
            {t("Trimble maps APIs.")}
          </ExternalLink>
        </>
      }
    />
  );
}

function defaultValues(): UpdateIntegrationConfigRequest {
  return {
    enabled: false,
    configuration: {
      apiKey: "",
      baseUrl: "https://pcmiler.alk.com/apis/rest/v1.0/Service.svc",
    },
  };
}
