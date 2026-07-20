const eiaLogoLight = "/integrations/logos/eia-light.svg";
const eiaLogoDark = "/integrations/logos/eia-dark.svg";

import trenovaLogo from "@/assets/logo.webp";
import { InputField } from "@/components/fields/input-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { LazyImage } from "@/components/image";
import { ExternalLink } from "@/components/link";
import { useTheme } from "@/components/theme-provider";
import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { UpdateIntegrationConfigRequest } from "@/types/integration";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { toast } from "sonner";

const INTEGRATION_TYPE = "EIAFuelPrices";

export function EIAFuelPricesForm({ open, onClose }: { open: boolean; onClose: () => void }) {
  const queryClient = useQueryClient();

  const configQuery = useQuery({
    ...queries.integration.config(INTEGRATION_TYPE),
    enabled: open,
  });

  const { control, reset, handleSubmit, setError } = useForm<UpdateIntegrationConfigRequest>({
    defaultValues: {
      enabled: false,
      configuration: {
        apiKey: "",
        baseUrl: "https://api.eia.gov/v2",
      },
    },
  });

  const response = configQuery.data;
  const hasApiKey = response?.fields?.some((f) => f.key === "apiKey" && f.hasValue) ?? false;

  useEffect(() => {
    if (!open || !response) {
      return;
    }

    const valueByKey = new Map(response.fields.map((field) => [field.key, field.value ?? ""]));
    reset({
      enabled: response.enabled,
      configuration: {
        apiKey: "",
        baseUrl: valueByKey.get("baseUrl") || "https://api.eia.gov/v2",
      },
    });
  }, [open, response, reset]);

  const saveMutation = useApiMutation({
    mutationFn: (payload: UpdateIntegrationConfigRequest) =>
      apiService.integrationService.updateConfig(INTEGRATION_TYPE, payload),
    setFormError: setError,
    resourceName: "EIA fuel price configuration",
    onSuccess: async () => {
      toast.success("EIA fuel price integration updated");
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.integration.config(INTEGRATION_TYPE).queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.integration.catalog().queryKey,
        }),
      ]);
    },
  });

  const testConnectionMutation = useMutation({
    mutationFn: () => apiService.integrationService.testConnection(INTEGRATION_TYPE),
    onSuccess: async () => {
      toast.success("EIA connection successful");
      await queryClient.invalidateQueries({
        queryKey: queries.integration.catalog().queryKey,
      });
    },
    onError: () => {
      toast.error("EIA connection test failed");
    },
  });

  return (
    <div className="space-y-4">
      <EIAFuelPricesFormHeader />
      <Form onSubmit={handleSubmit((data) => saveMutation.mutateAsync(data))} className="space-y-4">
        <FormGroup cols={1}>
          <FormControl cols="full">
            <div className="flex items-center justify-between rounded-md border border-border bg-background p-3">
              <div>
                <Label htmlFor="eia-enabled">Enable EIA Fuel Prices</Label>
                <p className="text-xs text-muted-foreground">
                  Ingests weekly DOE diesel prices every Tuesday and auto-provisions all 11 DOE
                  regional indices.
                </p>
              </div>
              <Controller
                name="enabled"
                control={control}
                render={({ field }) => (
                  <Switch
                    id="eia-enabled"
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
              placeholder={hasApiKey ? "********" : "Enter your free EIA API key"}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              name="configuration.baseUrl"
              control={control}
              label="Base URL"
              autoComplete="off"
              placeholder="https://api.eia.gov/v2"
            />
          </FormControl>
        </FormGroup>
        <DialogFooter className="flex flex-row items-center sm:justify-between">
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={() => testConnectionMutation.mutateAsync()}
              isLoading={testConnectionMutation.isPending}
              loadingText="Testing..."
              disabled={configQuery.isLoading || saveMutation.isPending}
            >
              Test Connection
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
          </div>
        </DialogFooter>
      </Form>
    </div>
  );
}

function EIAFuelPricesFormHeader() {
  const { theme } = useTheme();
  const logo = theme === "dark" ? eiaLogoDark : eiaLogoLight;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-center gap-4">
        <LazyImage src={trenovaLogo} className="size-8" />
        <div className="flex items-center justify-center gap-1">
          <div className="size-1 rounded-full bg-muted-foreground" />
          <div className="size-1 rounded-full bg-muted-foreground" />
          <div className="size-1 rounded-full bg-muted-foreground" />
        </div>
        <LazyImage src={logo} alt="EIA Logo" className="h-8 max-w-24 object-contain" />
      </div>
      <div className="flex flex-col gap-2 text-center">
        <h3 className="text-lg font-semibold">Connect with EIA Fuel Prices</h3>
        <div className="flex flex-row items-center justify-center gap-1">
          <p className="text-xs text-muted-foreground">
            Free API key powers weekly DOE diesel price ingestion.
          </p>
          <ExternalLink href="https://www.eia.gov/opendata/" className="text-xs">
            Get a key
          </ExternalLink>
        </div>
      </div>
    </div>
  );
}
