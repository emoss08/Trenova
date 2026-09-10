import trenovaLogo from "@/assets/logo.webp";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SensitiveTextareaField } from "@/components/fields/sensitive-textarea-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { LazyImage } from "@/components/image";
import { ExternalLink } from "@/components/link";
import { useApiMutation } from "@/hooks/use-api-mutation";
import { queries } from "@/lib/queries";
import { apiService } from "@/services/api";
import type { ConfigFieldSpec, UpdateIntegrationConfigRequest } from "@/types/integration";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button } from "@trenova/shared/components/ui/button";
import { DialogFooter } from "@trenova/shared/components/ui/dialog";
import { Form, FormControl, FormGroup } from "@trenova/shared/components/ui/form";
import { Label } from "@trenova/shared/components/ui/label";
import { Switch } from "@trenova/shared/components/ui/switch";
import { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { toast } from "sonner";

/**
 * The fuel card connections are large forms — an SFTP endpoint, its credentials,
 * and the layout of the file it serves — and the three networks differ only in
 * their copy. Rendering them from the field spec the server already returns keeps
 * the form in step with the Go `ConfigSpecs` on its own, so adding a field to a
 * connector never means editing a form here.
 */

export type FuelFeedVendor = {
  integrationType: string;
  name: string;
  logoLight: string;
  logoDark?: string;
  headline: string;
  blurb: string;
  docsLabel: string;
  docsUrl: string;
  /** Shown above the form when connecting needs something outside Trenova. */
  prerequisite?: string;
};

// The layout field only makes sense for a fixed-width file, so it follows the
// format select rather than sitting there confusing everyone reading a CSV.
const FIXED_WIDTH_ONLY = new Set(["fixedWidthLayout"]);
// Keys whose value is many lines or one very long one, and unreadable in an input.
const MULTILINE_KEYS = new Set(["fixedWidthLayout", "privateKey", "knownHostKey"]);
const FILE_FORMAT_KEY = "fileFormat";
const AUTH_MODE_KEY = "authMode";

export function FuelFeedForm({
  vendor,
  open,
  onClose,
}: {
  vendor: FuelFeedVendor;
  open: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const configQuery = useQuery({
    ...queries.integration.config(vendor.integrationType),
    enabled: open,
  });
  const response = configQuery.data;
  const spec = response?.spec ?? [];

  const form = useForm<UpdateIntegrationConfigRequest>({
    defaultValues: { enabled: false, configuration: {} },
  });
  const { control, reset, handleSubmit, watch } = form;

  useEffect(() => {
    if (!open || !response) {
      return;
    }

    const valueByKey = new Map(response.fields.map((field) => [field.key, field.value ?? ""]));
    const configuration: Record<string, string> = {};
    for (const field of response.spec) {
      // A sensitive value is never sent back to the browser, so its input starts
      // blank and an unchanged blank leaves the stored secret alone.
      configuration[field.key] = field.sensitive
        ? ""
        : (valueByKey.get(field.key) ?? field.default ?? "");
    }

    reset({ enabled: response.enabled, configuration });
  }, [open, response, reset]);

  const saveMutation = useApiMutation({
    mutationFn: (payload: UpdateIntegrationConfigRequest) =>
      apiService.integrationService.updateConfig(vendor.integrationType, payload),
    form,
    resourceName: `${vendor.name} configuration`,
    onSuccess: async () => {
      toast.success(`${vendor.name} integration updated`);
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.integration.config(vendor.integrationType).queryKey,
        }),
        queryClient.invalidateQueries({ queryKey: queries.integration.catalog().queryKey }),
      ]);
    },
  });

  const testConnectionMutation = useMutation({
    mutationFn: () => apiService.integrationService.testConnection(vendor.integrationType),
    onSuccess: async () => {
      toast.success(`${vendor.name} connection successful`);
      await queryClient.invalidateQueries({ queryKey: queries.integration.catalog().queryKey });
    },
    onError: () => toast.error(`${vendor.name} connection test failed`),
  });

  const fileFormat = watch(`configuration.${FILE_FORMAT_KEY}` as const);
  const authMode = watch(`configuration.${AUTH_MODE_KEY}` as const);
  const storedByKey = new Map(response?.fields.map((field) => [field.key, field.hasValue]) ?? []);

  const visible = spec.filter((field) =>
    isFieldVisible(field, { fileFormat, authMode, hasFormatField: hasKey(spec, FILE_FORMAT_KEY) }),
  );

  return (
    <div className="space-y-4">
      <FuelFeedHeader vendor={vendor} />
      {vendor.prerequisite ? (
        <p className="border-border bg-muted/40 text-muted-foreground rounded-md border p-3 text-xs">
          {vendor.prerequisite}
        </p>
      ) : null}
      <Form onSubmit={handleSubmit((data) => saveMutation.mutateAsync(data))} className="space-y-4">
        <FormGroup cols={1}>
          <FormControl cols="full">
            <Controller
              name="enabled"
              control={control}
              render={({ field }) => (
                <div className="border-border bg-background flex items-center justify-between rounded-md border p-3">
                  <div>
                    <Label htmlFor={`${vendor.integrationType}-enabled`}>
                      Enable {vendor.name}
                    </Label>
                    <p className="text-muted-foreground text-xs">
                      Read transactions on a schedule and post the ones that resolve.
                    </p>
                  </div>
                  <Switch
                    id={`${vendor.integrationType}-enabled`}
                    checked={field.value}
                    onCheckedChange={field.onChange}
                  />
                </div>
              )}
            />
          </FormControl>
          {visible.map((field) => (
            <SpecField
              key={field.key}
              field={field}
              control={control}
              hasStoredValue={storedByKey.get(field.key) ?? false}
            />
          ))}
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

function hasKey(spec: ConfigFieldSpec[], key: string) {
  return spec.some((field) => field.key === key);
}

export function isFieldVisible(
  field: ConfigFieldSpec,
  context: { fileFormat?: string; authMode?: string; hasFormatField: boolean },
): boolean {
  if (FIXED_WIDTH_ONLY.has(field.key)) {
    return context.hasFormatField && context.fileFormat === "FixedWidth";
  }
  if (field.key === "password") {
    return context.authMode !== "privateKey";
  }
  if (field.key === "privateKey") {
    return context.authMode === "privateKey";
  }

  return true;
}

function SpecField({
  field,
  control,
  hasStoredValue,
}: {
  field: ConfigFieldSpec;
  control: ReturnType<typeof useForm<UpdateIntegrationConfigRequest>>["control"];
  hasStoredValue: boolean;
}) {
  const name = `configuration.${field.key}` as const;
  const label =
    field.sensitive && hasStoredValue ? `${field.label} (leave blank to keep)` : field.label;
  const placeholder = field.sensitive && hasStoredValue ? "********" : field.placeholder;

  if (field.sensitive) {
    return (
      <FormControl cols="full">
        {MULTILINE_KEYS.has(field.key) ? (
          <SensitiveTextareaField
            name={name}
            control={control}
            label={label}
            description={field.helpText}
            placeholder={placeholder}
            rows={6}
          />
        ) : (
          <SensitiveField
            name={name}
            control={control}
            label={label}
            description={field.helpText}
            autoComplete="off"
            placeholder={placeholder}
          />
        )}
      </FormControl>
    );
  }

  if (field.type === "select") {
    return (
      <FormControl cols="full">
        <SelectField
          name={name}
          control={control}
          label={label}
          description={field.helpText}
          placeholder={field.placeholder}
          options={(field.options ?? []).map((option) => ({ label: option, value: option }))}
        />
      </FormControl>
    );
  }

  // Configuration values are stored as strings, so a boolean field has to map
  // the switch onto "true" and "false" rather than a real boolean.
  if (field.type === "boolean") {
    return (
      <FormControl cols="full">
        <Controller
          name={name}
          control={control}
          render={({ field: controlled }) => (
            <div className="border-border bg-background flex items-center justify-between gap-4 rounded-md border p-3">
              <div>
                <Label htmlFor={`config-${field.key}`}>{label}</Label>
                {field.helpText ? (
                  <p className="text-muted-foreground text-xs">{field.helpText}</p>
                ) : null}
              </div>
              <Switch
                id={`config-${field.key}`}
                checked={controlled.value !== "false"}
                onCheckedChange={(checked) => controlled.onChange(checked ? "true" : "false")}
              />
            </div>
          )}
        />
      </FormControl>
    );
  }

  if (MULTILINE_KEYS.has(field.key)) {
    return (
      <FormControl cols="full">
        <TextareaField
          name={name}
          control={control}
          label={label}
          description={field.helpText}
          placeholder={field.placeholder}
          rows={6}
        />
      </FormControl>
    );
  }

  return (
    <FormControl cols="full">
      <InputField
        name={name}
        control={control}
        label={label}
        description={field.helpText}
        type={field.type === "number" ? "number" : "text"}
        placeholder={field.placeholder}
      />
    </FormControl>
  );
}

function FuelFeedHeader({ vendor }: { vendor: FuelFeedVendor }) {
  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-center gap-4">
        <LazyImage src={trenovaLogo} className="size-8" />
        <div className="flex items-center justify-center gap-1">
          <div className="bg-muted-foreground size-1 rounded-full" />
          <div className="bg-muted-foreground size-1 rounded-full" />
          <div className="bg-muted-foreground size-1 rounded-full" />
        </div>
        <LazyImage
          src={vendor.logoLight}
          alt={`${vendor.name} Logo`}
          className="h-8 max-w-24 object-contain"
        />
      </div>
      <div className="flex flex-col gap-2 text-center">
        <h3 className="text-lg font-semibold">{vendor.headline}</h3>
        <div className="flex flex-row items-center justify-center gap-1">
          <p className="text-muted-foreground text-xs">{vendor.blurb}</p>
          <ExternalLink href={vendor.docsUrl} className="text-xs">
            {vendor.docsLabel}
          </ExternalLink>
        </div>
      </div>
    </div>
  );
}
