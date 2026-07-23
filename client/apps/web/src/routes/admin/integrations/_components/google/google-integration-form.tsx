const googleMapsPinLogo = "/integrations/logos/googleMaps.svg";
import trenovaLogo from "@/assets/logo.webp";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { LazyImage } from "@/components/image";
import { ExternalLink } from "@/components/link";
import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { UpdateIntegrationConfigRequest } from "@/types/integration";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { toast } from "sonner";

export function GoogleMapsForm({ open, onClose }: { open: boolean; onClose: () => void }) {
  const queryClient = useQueryClient();

  const configQuery = useQuery({
    ...queries.integration.config("GoogleMaps"),
    enabled: open,
  });

  const { control, reset, handleSubmit, setError } = useForm<UpdateIntegrationConfigRequest>({
    defaultValues: {
      enabled: false,
      configuration: {
        apiKey: "",
      },
    },
  });

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
    setFormError: setError,
    resourceName: "Google Maps configuration",
    onSuccess: async () => {
      toast.success("Google Maps integration updated");
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
            <div className="flex items-center justify-between rounded-md border border-border bg-background p-3">
              <div>
                <Label htmlFor="google-enabled">Enable Google Maps</Label>
                <p className="text-xs text-muted-foreground">
                  Toggle integration state for this business unit.
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
        <DialogFooter className="flex flex-row items-center sm:justify-between">
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            size="sm"
            type="submit"
            isLoading={saveMutation.isPending}
            loadingText="Saving..."
            disabled={configQuery.isLoading}
          >
            Save Changes
          </Button>
        </DialogFooter>
      </Form>
    </div>
  );
}

export function GoogleMapsFormHeader() {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-center gap-4">
        <LazyImage src={trenovaLogo} className="size-8" />
        <div className="flex items-center justify-center gap-1">
          <div className="size-1 rounded-full bg-muted-foreground" />
          <div className="size-1 rounded-full bg-muted-foreground" />
          <div className="size-1 rounded-full bg-muted-foreground" />
        </div>
        <LazyImage src={googleMapsPinLogo} alt="Google Maps Logo" className="size-8" />
      </div>
      <div className="flex flex-col gap-2 text-center">
        <h3 className="text-lg font-semibold">Connect with Google Maps</h3>
        <div className="flex flex-row items-center justify-center gap-1">
          <p className="text-xs text-muted-foreground">To get a Google Maps API key, visit the</p>
          <ExternalLink
            href="https://console.cloud.google.com/google/maps-apis/overview"
            className="text-xs"
          >
            Google Cloud Console.
          </ExternalLink>
        </div>
      </div>
    </div>
  );
}
