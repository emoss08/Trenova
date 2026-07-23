const oandaLogoLight = "/integrations/logos/oanada-light.svg";
const oandaLogoDark = "/integrations/logos/oanada-dark.svg";

import trenovaLogo from "@/assets/logo.webp";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SelectField } from "@/components/fields/select-field";
import { InputField } from "@/components/fields/input-field";
import { LazyImage } from "@/components/image";
import { ExternalLink } from "@/components/link";
import { useTheme } from "@trenova/shared/components/theme-provider";
import { Button } from "@trenova/shared/components/ui/button";
import { DialogFooter } from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Label } from "@trenova/shared/components/ui/label";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { UpdateIntegrationConfigRequest } from "@/types/integration";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { toast } from "sonner";

const INTEGRATION_TYPE = "OANDAExchangeRates";

const rateTypeOptions = [
  {
    label: "Midpoint",
    value: "mid",
    description: "Default settlement policy",
  },
  {
    label: "Bid",
    value: "bid",
    description: "Provider bid rate",
  },
  {
    label: "Ask",
    value: "ask",
    description: "Provider ask rate",
  },
];

export function OANDAExchangeRatesForm({ open, onClose }: { open: boolean; onClose: () => void }) {
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
        baseUrl: "https://exchange-rates-api.oanda.com",
        defaultRateType: "mid",
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
        baseUrl: valueByKey.get("baseUrl") || "https://exchange-rates-api.oanda.com",
        defaultRateType: valueByKey.get("defaultRateType") || "mid",
      },
    });
  }, [open, response, reset]);

  const saveMutation = useApiMutation({
    mutationFn: (payload: UpdateIntegrationConfigRequest) =>
      apiService.integrationService.updateConfig(INTEGRATION_TYPE, payload),
    setFormError: setError,
    resourceName: "OANDA exchange-rate configuration",
    onSuccess: async () => {
      toast.success("OANDA exchange-rate integration updated");
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
      toast.success("OANDA connection successful");
      await queryClient.invalidateQueries({
        queryKey: queries.integration.catalog().queryKey,
      });
    },
    onError: () => {
      toast.error("OANDA connection test failed");
    },
  });

  return (
    <div className="space-y-4">
      <OANDAExchangeRatesFormHeader />
      <Form onSubmit={handleSubmit((data) => saveMutation.mutateAsync(data))} className="space-y-4">
        <FormGroup cols={1}>
          <FormControl cols="full">
            <div className="flex items-center justify-between rounded-md border border-border bg-background p-3">
              <div>
                <Label htmlFor="oanda-enabled">Enable OANDA FX</Label>
                <p className="text-xs text-muted-foreground">
                  Toggle settlement-grade FX data for this business unit.
                </p>
              </div>
              <Controller
                name="enabled"
                control={control}
                render={({ field }) => (
                  <Switch
                    id="oanda-enabled"
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
              placeholder={hasApiKey ? "********" : "Enter your OANDA API key"}
            />
          </FormControl>
          <FormControl cols="full">
            <InputField
              name="configuration.baseUrl"
              control={control}
              label="Base URL"
              autoComplete="off"
              placeholder="https://exchange-rates-api.oanda.com"
            />
          </FormControl>
          <FormControl cols="full">
            <SelectField
              name="configuration.defaultRateType"
              control={control}
              label="Default Rate Type"
              options={rateTypeOptions}
              placeholder="Select rate type"
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

function OANDAExchangeRatesFormHeader() {
  const { theme } = useTheme();
  const logo = theme === "dark" ? oandaLogoDark : oandaLogoLight;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-center gap-4">
        <LazyImage src={trenovaLogo} className="size-8" />
        <div className="flex items-center justify-center gap-1">
          <div className="size-1 rounded-full bg-muted-foreground" />
          <div className="size-1 rounded-full bg-muted-foreground" />
          <div className="size-1 rounded-full bg-muted-foreground" />
        </div>
        <LazyImage src={logo} alt="OANDA Logo" className="h-8 max-w-24 object-contain" />
      </div>
      <div className="flex flex-col gap-2 text-center">
        <h3 className="text-lg font-semibold">Connect with OANDA Exchange Rates</h3>
        <div className="flex flex-row items-center justify-center gap-1">
          <p className="text-xs text-muted-foreground">Midpoint is used by default for quotes.</p>
          <ExternalLink
            href="https://www.oanda.com/foreign-exchange-data-services/en/exchange-rates-api/"
            className="text-xs"
          >
            OANDA FXDS
          </ExternalLink>
        </div>
      </div>
    </div>
  );
}
