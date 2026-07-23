import { InputField } from "@/components/fields/input-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Form, FormControl, FormGroup } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { ApiRequestError } from "@/lib/api";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { UpdateIntegrationConfigRequest } from "@/types/integration";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon } from "lucide-react";
import { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { toast } from "sonner";

function getFieldValue(
  fields: { key: string; value?: string; hasValue: boolean }[] | undefined,
  key: string,
): { value?: string; hasValue: boolean } {
  const field = fields?.find((f) => f.key === key);
  return { value: field?.value, hasValue: field?.hasValue ?? false };
}

export function SamsaraConfigurationContent({ open }: { open: boolean }) {
  const queryClient = useQueryClient();

  const configQuery = useQuery({
    ...queries.integration.config("Samsara"),
    enabled: open,
  });
  const { control, watch, reset, setValue, handleSubmit, setError } =
    useForm<UpdateIntegrationConfigRequest>({
      defaultValues: {
        enabled: false,
        configuration: {
          token: "",
          baseUrl: "",
        },
      },
    });

  const response = configQuery.data;
  const hasToken = getFieldValue(response?.fields, "token").hasValue;
  const enabled = watch("enabled");
  const token = watch("configuration.token");

  useEffect(() => {
    if (!open || !response) {
      return;
    }

    reset({
      enabled: response.enabled,
      configuration: {
        token: "",
        baseUrl: getFieldValue(response.fields, "baseUrl").value ?? "",
      },
    });
  }, [open, response, reset]);

  const saveMutation = useApiMutation({
    mutationFn: (payload: UpdateIntegrationConfigRequest) =>
      apiService.integrationService.updateConfig("Samsara", payload),
    setFormError: setError,
    resourceName: "Samsara configuration",
    onSuccess: async () => {
      setValue("configuration.token", "");
      toast.success("Samsara integration updated");
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.integration.config("Samsara").queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.integration.catalog().queryKey,
        }),
      ]);
    },
  });
  const testMutation = useMutation({
    mutationFn: () => apiService.integrationService.testConnection("Samsara"),
    onSuccess: () => {
      setValue("enabled", true, { shouldDirty: true });
      toast.success("Samsara connection successful");
    },
    onError: (error) => {
      if (error instanceof ApiRequestError) {
        toast.error("Samsara connection test failed", {
          description: error.data.detail || error.data.title,
        });
        return;
      }

      toast.error("Samsara connection test failed");
    },
  });

  const onSubmit = async (data: UpdateIntegrationConfigRequest) => {
    await saveMutation.mutateAsync(data);

    const testResult = await testMutation.mutateAsync();
    if (testResult.success) {
      setValue("enabled", true, { shouldDirty: false });
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.integration.config("Samsara").queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.integration.catalog().queryKey,
        }),
      ]);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-col border-b border-border p-4 leading-tight">
        <p className="text-2xl font-semibold">Samsara Configuration</p>
        <span className="text-sm text-muted-foreground">
          Configure your Samsara integration settings for this organization.
        </span>
      </div>
      <section className="px-4">
        <Form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <FormGroup cols={1}>
            <FormControl cols="full">
              <div className="flex items-center justify-between rounded-md border border-border bg-background p-3">
                <div>
                  <Label htmlFor="samsara-enabled">Enable Samsara</Label>
                  <p className="text-xs text-muted-foreground">
                    Explicitly toggle integration state for this business unit.
                  </p>
                </div>
                <Controller
                  name="enabled"
                  control={control}
                  render={({ field }) => (
                    <Switch
                      id="samsara-enabled"
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  )}
                />
              </div>
            </FormControl>
            <FormControl cols="full">
              <InputField
                name="configuration.baseUrl"
                control={control}
                label="Base URL (optional)"
                placeholder="https://api.samsara.com"
              />
            </FormControl>
            <FormControl cols="full">
              <SensitiveField
                name="configuration.token"
                control={control}
                label={`API Token ${hasToken ? "(leave blank to keep existing token)" : ""}`}
                autoComplete="token"
                placeholder={hasToken ? "********" : "Enter Samsara API token"}
              />
            </FormControl>
            {enabled && !hasToken && token.trim() === "" && (
              <FormControl cols="full">
                <Alert variant="warning">
                  <AlertTriangleIcon />
                  <AlertTitle>Token required</AlertTitle>
                  <AlertDescription>Provide a token before enabling Samsara.</AlertDescription>
                </Alert>
              </FormControl>
            )}
            <FormControl cols="full">
              <div className="flex flex-wrap gap-2">
                <Button
                  size="sm"
                  type="submit"
                  isLoading={saveMutation.isPending || testMutation.isPending}
                  loadingText={saveMutation.isPending ? "Saving..." : "Testing..."}
                  disabled={configQuery.isLoading}
                >
                  Save Configuration
                </Button>
              </div>
            </FormControl>
          </FormGroup>
        </Form>
      </section>
    </div>
  );
}
