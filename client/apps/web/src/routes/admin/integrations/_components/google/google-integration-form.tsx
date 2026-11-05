const googleMapsPinLogo = "/integrations/logos/googleMaps.svg";
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
import { integrationService } from "@/services/integration";
import type { UpdateIntegrationConfigRequest } from "@/types/integration";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { IntegrationConnectHeader } from "../shared/integration-connect-header";

export function GoogleMapsForm({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT();

  const queryClient = useQueryClient();

  const configQuery = useQuery({
    ...integration.config("GoogleMaps"),
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
      integrationService.updateConfig("GoogleMaps", payload),
    form,
    resourceName: "Google Maps configuration",
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: integration.config("GoogleMaps").queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: integration.catalog().queryKey,
        }),
      ]);
    },
  });

  return (
    <div className="space-y-4">
      <GoogleMapsFormHeader />
      <Form onSubmit={handleSubmit((data) => saveMutation.mutateAsync(data))} className="space-y-4">
        <FormGroup cols={1}>
          <FormControl cols="full">
            <div className="border-border bg-background flex items-center justify-between rounded-md border p-3">
              <div>
                <Label htmlFor="google-enabled">{t("Enable Google maps")}</Label>
                <p className="text-muted-foreground text-xs">
                  {t("Toggle integration state for this business unit.")}
                </p>
              </div>
              <Controller
                name="enabled"
                control={control}
                render={({ field }) => (
                  <Switch
                    id="google-enabled"
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
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
              placeholder={hasApiKey ? "********" : "Enter your Google Maps API Key"}
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

export function GoogleMapsFormHeader() {
  const t = useT();

  return (
    <IntegrationConnectHeader
      name="Google Maps"
      logo={{ light: googleMapsPinLogo }}
      title={t("Connect with Google maps")}
      description={
        <>
          {t("To get a Google Maps API key, visit the")}{" "}
          <ExternalLink href="https://console.cloud.google.com/google/maps-apis/overview">
            {t("Google cloud console.")}
          </ExternalLink>
        </>
      }
    />
  );
}
