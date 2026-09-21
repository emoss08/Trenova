const googleMapsPinLogo = "/integrations/logos/googleMaps.svg";
import { useT } from "@trenova/shared/i18n/use-t";
import trenovaLogo from "@/assets/logo.webp";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { LazyImage } from "@/components/image";
import { ExternalLink } from "@/components/link";
import { Button } from "@trenova/shared/components/ui/button";
import {
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Label } from "@trenova/shared/components/ui/label";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { UpdateIntegrationConfigRequest } from "@/types/integration";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { toast } from "sonner";

export function GoogleMapsForm({ open, onClose }: { open: boolean; onClose: () => void }) {
  const t = useT();

  const queryClient = useQueryClient();

  const configQuery = useQuery({
    ...queries.integration.config("GoogleMaps"),
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
      apiService.integrationService.updateConfig("GoogleMaps", payload),
    form,
    resourceName: "Google Maps configuration",
    onSuccess: async () => {
      toast.success(t("Google Maps integration updated"));
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.integration.config("GoogleMaps").queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.integration.catalog().queryKey,
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
              label={`API Key ${hasApiKey ? "(leave blank to keep existing key)" : ""}`}
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
    <div className="flex flex-col gap-2">
      <div className="flex items-center gap-4">
        <LazyImage src={trenovaLogo} className="size-8" />
        <div className="flex items-center justify-center gap-1">
          <div className="bg-muted-foreground size-1 rounded-full" />
          <div className="bg-muted-foreground size-1 rounded-full" />
          <div className="bg-muted-foreground size-1 rounded-full" />
        </div>
        <LazyImage src={googleMapsPinLogo} alt={t("Google maps logo")} className="size-8" />
      </div>
      <DialogHeader>
        <DialogTitle>{t("Connect with Google maps")}</DialogTitle>
        <DialogDescription>
          {t("To get a Google Maps API key, visit the")}{" "}
          <ExternalLink href="https://console.cloud.google.com/google/maps-apis/overview">
            {t("Google cloud console.")}
          </ExternalLink>
        </DialogDescription>
      </DialogHeader>
    </div>
  );
}
