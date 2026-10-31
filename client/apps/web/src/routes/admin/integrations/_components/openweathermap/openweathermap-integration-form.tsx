const openWeatherMapLogo = "/integrations/logos/open_weather_logo.webp";
import { useT } from "@trenova/shared/i18n/use-t";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { ExternalLink } from "@/components/link";
import { Button } from "@trenova/shared/components/ui/button";
import { DialogFooter } from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Label } from "@trenova/shared/components/ui/label";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { integration } from "@/lib/queries/integration";
import { apiService } from "@/services/api";
import type { UpdateIntegrationConfigRequest } from "@/types/integration";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { IntegrationConnectHeader } from "../shared/integration-connect-header";

export function OpenWeatherMapForm({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT();

  const queryClient = useQueryClient();

  const configQuery = useQuery({
    ...integration.config("OpenWeatherMap"),
    enabled: open,
  });

  const form = useForm<UpdateIntegrationConfigRequest>({
    defaultValues: {
      enabled: false,
      configuration: {
        apiKey: "",
      },
    },
  });
  const { control, reset, handleSubmit } = form;

  const response = configQuery.data;
  const hasApiKey = response?.fields?.some((f) => f.key === "apiKey" && f.hasValue) ?? false;

  useEffect(() => {
    if (!open || !response) {
      return;
    }

    reset({
      enabled: response.enabled,
      configuration: {
        apiKey: "",
      },
    });
  }, [open, response, reset]);

  const saveMutation = useApiMutation({
    mutationFn: (payload: UpdateIntegrationConfigRequest) =>
      apiService.integrationService.updateConfig("OpenWeatherMap", payload),
    form,
    resourceName: "OpenWeatherMap configuration",
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: integration.config("OpenWeatherMap").queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: integration.catalog().queryKey,
        }),
      ]);
    },
  });

  return (
    <div className="space-y-4">
      <OpenWeatherMapFormHeader />
      <Form onSubmit={handleSubmit((data) => saveMutation.mutateAsync(data))} className="space-y-4">
        <FormGroup cols={1}>
          <FormControl cols="full">
            <div className="border-border bg-background flex items-center justify-between rounded-md border p-3">
              <div>
                <Label htmlFor="owm-enabled">{t("Enable OpenWeatherMap")}</Label>
                <p className="text-muted-foreground text-xs">
                  {t("Toggle integration state for this business unit.")}
                </p>
              </div>
              <Controller
                name="enabled"
                control={control}
                render={({ field }) => (
                  <Switch id="owm-enabled" checked={field.value} onCheckedChange={field.onChange} />
                )}
              />
            </div>
          </FormControl>
          <FormControl cols="full">
            <SensitiveField
              name="configuration.apiKey"
              control={control}
              label={hasApiKey ? t("API Key (leave blank to keep existing key)") : t("API Key")}
              autoComplete="off"
              placeholder={hasApiKey ? "********" : "Enter your OpenWeatherMap API Key"}
            />
          </FormControl>
        </FormGroup>
        <DialogFooter>
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
        </DialogFooter>
      </Form>
    </div>
  );
}

function OpenWeatherMapFormHeader() {
  const t = useT();

  return (
    <IntegrationConnectHeader
      name="OpenWeatherMap"
      logo={{ light: openWeatherMapLogo }}
      title={t("Connect with OpenWeatherMap")}
      description={
        <>
          {t("To get a free API key, visit")}{" "}
          <ExternalLink href="https://home.openweathermap.org/api_keys">
            {t("OpenWeatherMap API keys.")}
          </ExternalLink>
        </>
      }
    />
  );
}
