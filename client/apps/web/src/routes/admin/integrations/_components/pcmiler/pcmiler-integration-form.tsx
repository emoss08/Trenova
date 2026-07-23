const pcMilerLogoLight = "/integrations/logos/pc-miler-logo-light.png";
const pcMilerLogoDark = "/integrations/logos/pc-miler-logo-dark.svg";

import trenovaLogo from "@/assets/logo.webp";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { InputField } from "@/components/fields/input-field";
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

const INTEGRATION_TYPE = "PCMiler";

export function PCMilerIntegrationForm({ open, onClose }: { open: boolean; onClose: () => void }) {
  const queryClient = useQueryClient();
  const configQuery = useQuery({ ...queries.integration.config(INTEGRATION_TYPE), enabled: open });
  const { control, reset, handleSubmit, setError } = useForm<UpdateIntegrationConfigRequest>({
    defaultValues: defaultValues(),
  });

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
      apiService.integrationService.updateConfig(INTEGRATION_TYPE, payload),
    setFormError: setError,
    resourceName: "PC*Miler configuration",
    onSuccess: async () => {
      toast.success("PC*Miler integration updated");
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.integration.config(INTEGRATION_TYPE).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: queries.integration.catalog().queryKey }),
      ]);
    },
  });

  const testConnectionMutation = useMutation({
    mutationFn: () => apiService.integrationService.testConnection(INTEGRATION_TYPE),
    onSuccess: async () => {
      toast.success("PC*Miler connection successful");
      await queryClient.invalidateQueries({ queryKey: queries.integration.catalog().queryKey });
    },
    onError: () => toast.error("PC*Miler connection test failed"),
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
                <div className="flex items-center justify-between rounded-md border border-border bg-background p-3">
                  <div>
                    <Label htmlFor="pcmiler-enabled">Enable PC*Miler</Label>
                    <p className="text-xs text-muted-foreground">
                      Toggle mileage rating for this business unit.
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
              label={`API Key ${hasApiKey ? "(leave blank to keep existing key)" : ""}`}
              autoComplete="off"
              placeholder={hasApiKey ? "********" : "Enter your Trimble Maps API key"}
            />
          </FormControl>
          <TextField control={control} name="baseUrl" label="Base URL" />
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
  const { theme } = useTheme();
  const logo = theme === "dark" ? pcMilerLogoDark : pcMilerLogoLight;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-center gap-4">
        <LazyImage src={trenovaLogo} className="size-8" />
        <div className="flex items-center justify-center gap-1">
          <div className="size-1 rounded-full bg-muted-foreground" />
          <div className="size-1 rounded-full bg-muted-foreground" />
          <div className="size-1 rounded-full bg-muted-foreground" />
        </div>
        <LazyImage src={logo} alt="PC*Miler Logo" className="h-8 max-w-24 object-contain" />
      </div>
      <div className="flex flex-col gap-2 text-center">
        <h3 className="text-lg font-semibold">Connect with PC*Miler</h3>
        <div className="flex flex-row items-center justify-center gap-1">
          <p className="text-xs text-muted-foreground">Configure mileage rating with</p>
          <ExternalLink href="https://developer.trimblemaps.com/" className="text-xs">
            Trimble Maps APIs.
          </ExternalLink>
        </div>
      </div>
    </div>
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
