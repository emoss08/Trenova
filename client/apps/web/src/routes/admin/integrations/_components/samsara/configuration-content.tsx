import { InputField } from "@/components/fields/input-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { Alert, AlertDescription, AlertTitle } from "@trenova/shared/components/ui/alert";
import { Button } from "@trenova/shared/components/ui/button";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Label } from "@trenova/shared/components/ui/label";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { ApiRequestError } from "@trenova/shared/lib/api";
import { API_BASE_URL } from "@trenova/shared/lib/constants";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { UpdateIntegrationConfigRequest } from "@/types/integration";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { AlertTriangleIcon, CheckIcon, CopyIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { Controller, useForm, useWatch } from "react-hook-form";
import { toast } from "sonner";
import { SamsaraFormMappingSection } from "./form-mapping-section";
import { SamsaraSyncHealthSection } from "./sync-health-section";

function getFieldValue(
  fields: { key: string; value?: string; hasValue: boolean }[] | undefined,
  key: string,
): { value?: string; hasValue: boolean } {
  const field = fields?.find((f) => f.key === key);
  return { value: field?.value, hasValue: field?.hasValue ?? false };
}

function buildWebhookUrl(webhookToken: string | undefined): string | null {
  if (!webhookToken) {
    return null;
  }

  const apiBase = API_BASE_URL.startsWith("http")
    ? API_BASE_URL
    : `${window.location.origin}${API_BASE_URL}`;

  return `${apiBase}/webhooks/samsara/${webhookToken}/`;
}

export function SamsaraConfigurationContent({ open }: { open: boolean }) {
  const queryClient = useQueryClient();
  const [copied, setCopied] = useState(false);

  const configQuery = useQuery({
    ...queries.integration.config("Samsara"),
    enabled: open,
  });
  const { control, reset, setValue, handleSubmit, setError } =
    useForm<UpdateIntegrationConfigRequest>({
      defaultValues: {
        enabled: false,
        configuration: {
          token: "",
          baseUrl: "",
          webhookSecret: "",
        },
      },
    });

  const response = configQuery.data;
  const hasToken = getFieldValue(response?.fields, "token").hasValue;
  const hasWebhookSecret = getFieldValue(response?.fields, "webhookSecret").hasValue;
  const webhookUrl = buildWebhookUrl(getFieldValue(response?.fields, "webhookToken").value);
  const enabled = useWatch({ control, name: "enabled" });
  const token = useWatch({ control, name: "configuration.token" });

  useEffect(() => {
    if (!open || !response) {
      return;
    }

    reset({
      enabled: response.enabled,
      configuration: {
        token: "",
        baseUrl: getFieldValue(response.fields, "baseUrl").value ?? "",
        webhookSecret: "",
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
      setValue("configuration.webhookSecret", "");
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

  const copyWebhookUrl = async () => {
    if (!webhookUrl) {
      return;
    }
    await navigator.clipboard.writeText(webhookUrl);
    setCopied(true);
    toast.success("Webhook URL copied to clipboard");
    setTimeout(() => setCopied(false), 1500);
  };

  const onSubmit = async (data: UpdateIntegrationConfigRequest) => {
    const configuration = { ...data.configuration };
    if (!configuration.webhookSecret?.trim()) {
      delete configuration.webhookSecret;
    }

    await saveMutation.mutateAsync({ ...data, configuration });

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
              <div className="flex flex-col gap-0.5 border-t border-border pt-4">
                <p className="text-sm font-semibold">Webhooks</p>
                <p className="text-xs text-muted-foreground">
                  Receive real-time vehicle and driver events from Samsara instead of waiting on
                  polling.
                </p>
              </div>
            </FormControl>
            <FormControl cols="full">
              <SensitiveField
                name="configuration.webhookSecret"
                control={control}
                label={`Webhook Secret ${
                  hasWebhookSecret ? "(leave blank to keep existing secret)" : ""
                }`}
                autoComplete="off"
                placeholder={hasWebhookSecret ? "********" : "Enter Samsara webhook signing secret"}
                description="Base64 signing secret from Samsara's webhook configuration, used to verify the X-Samsara-Signature header on incoming events."
              />
            </FormControl>
            <FormControl cols="full">
              {webhookUrl ? (
                <div className="flex flex-col gap-1.5">
                  <Label>Webhook Endpoint</Label>
                  <div className="flex items-center gap-2 rounded-md border border-border bg-muted/40 p-2">
                    <p className="min-w-0 flex-1 truncate font-mono text-xs">{webhookUrl}</p>
                    <Button
                      type="button"
                      variant="ghost"
                      size="sm"
                      className="h-7 px-2"
                      onClick={() => void copyWebhookUrl()}
                    >
                      {copied ? (
                        <CheckIcon className="size-3.5" />
                      ) : (
                        <CopyIcon className="size-3.5" />
                      )}
                      <span className="sr-only">Copy webhook URL</span>
                    </Button>
                  </div>
                  <p className="text-xs text-muted-foreground">
                    Point Samsara&apos;s webhook at this URL. It is unique to your organization.
                  </p>
                </div>
              ) : (
                <p className="text-xs text-muted-foreground">
                  Save the configuration to generate your webhook endpoint.
                </p>
              )}
            </FormControl>
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
      <section className="px-4">
        <SamsaraSyncHealthSection open={open} />
      </section>
      <section className="px-4 pb-4">
        <SamsaraFormMappingSection open={open} />
      </section>
    </div>
  );
}
