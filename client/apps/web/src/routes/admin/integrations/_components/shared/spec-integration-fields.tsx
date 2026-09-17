import { useT } from "@trenova/shared/i18n/use-t";
import trenovaLogo from "@/assets/logo.webp";
import { InputField } from "@/components/fields/input-field";
import { SelectField } from "@/components/fields/select-field";
import { SensitiveField } from "@/components/fields/sensitive-field";
import { SensitiveTextareaField } from "@/components/fields/sensitive-textarea-field";
import { TextareaField } from "@/components/fields/textarea-field";
import { LazyImage } from "@/components/image";
import { ExternalLink } from "@/components/link";
import type { ConfigFieldSpec, UpdateIntegrationConfigRequest } from "@/types/integration";
import { useTheme } from "@trenova/shared/components/theme-provider";
import { Button } from "@trenova/shared/components/ui/button";
import { DialogFooter } from "@trenova/shared/components/ui/dialog";
import { FormControl } from "@trenova/shared/components/ui/form";
import { Label } from "@trenova/shared/components/ui/label";
import { Switch } from "@trenova/shared/components/ui/switch";
import { cn } from "@trenova/shared/lib/utils";
import type { ReactNode } from "react";
import { type Control, Controller } from "react-hook-form";

export type SpecIntegrationControl = Control<UpdateIntegrationConfigRequest>;

export type SpecIntegrationVendor = {
  integrationType: string;
  name: string;
  logoLight?: string;
  logoDark?: string;
  headline: string;
  blurb: string;
  docsLabel: string;
  docsUrl: string;
  prerequisite?: string;
};

const NO_MULTILINE_KEYS: ReadonlySet<string> = new Set();

export function SpecIntegrationFields({
  fields,
  control,
  storedByKey,
  multilineKeys = NO_MULTILINE_KEYS,
  renderOptionLabel,
}: {
  fields: ConfigFieldSpec[];
  control: SpecIntegrationControl;
  storedByKey: ReadonlyMap<string, boolean>;
  multilineKeys?: ReadonlySet<string>;
  renderOptionLabel?: (field: ConfigFieldSpec, option: string) => string;
}) {
  return (
    <>
      {fields.map((field) => (
        <SpecIntegrationField
          key={field.key}
          field={field}
          control={control}
          hasStoredValue={storedByKey.get(field.key) ?? false}
          multiline={multilineKeys.has(field.key)}
          renderOptionLabel={renderOptionLabel}
        />
      ))}
    </>
  );
}

export function SpecIntegrationField({
  field,
  control,
  hasStoredValue,
  multiline = false,
  renderOptionLabel,
}: {
  field: ConfigFieldSpec;
  control: SpecIntegrationControl;
  hasStoredValue: boolean;
  multiline?: boolean;
  renderOptionLabel?: (field: ConfigFieldSpec, option: string) => string;
}) {
  const name = `configuration.${field.key}` as const;
  const label =
    field.sensitive && hasStoredValue ? `${field.label} (leave blank to keep)` : field.label;
  const placeholder = field.sensitive && hasStoredValue ? "********" : field.placeholder;

  if (field.sensitive) {
    return (
      <FormControl cols="full">
        {multiline ? (
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
          options={(field.options ?? []).map((option) => ({
            label: renderOptionLabel ? renderOptionLabel(field, option) : option,
            value: option,
          }))}
        />
      </FormControl>
    );
  }

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

  if (multiline) {
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

export function IntegrationEnabledSwitch({
  control,
  id,
  label,
  description,
}: {
  control: SpecIntegrationControl;
  id: string;
  label: string;
  description: ReactNode;
}) {
  return (
    <FormControl cols="full">
      <Controller
        name="enabled"
        control={control}
        render={({ field }) => (
          <div className="border-border bg-background flex items-center justify-between rounded-md border p-3">
            <div>
              <Label htmlFor={id}>{label}</Label>
              <p className="text-muted-foreground text-xs">{description}</p>
            </div>
            <Switch id={id} checked={field.value} onCheckedChange={field.onChange} />
          </div>
        )}
      />
    </FormControl>
  );
}

export function SpecIntegrationFooter({
  onCancel,
  onTestConnection,
  isTesting,
  isSaving,
  isLoading,
  canTest = true,
  className,
}: {
  className?: string;
  onCancel: () => void;
  onTestConnection: () => void;
  isTesting: boolean;
  isSaving: boolean;
  isLoading: boolean;
  canTest?: boolean;
}) {
  const t = useT();

  return (
    <DialogFooter className={cn("flex flex-row items-center sm:justify-between", className)}>
      <Button type="button" variant="outline" onClick={onCancel}>
        {t("Cancel")}
      </Button>
      <div className="flex items-center gap-2">
        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={onTestConnection}
          isLoading={isTesting}
          loadingText={t("Testing...")}
          disabled={isLoading || isSaving || !canTest}
        >
          {t("Test Connection")}
        </Button>
        <Button
          size="sm"
          type="submit"
          isLoading={isSaving}
          loadingText={t("Saving...")}
          disabled={isLoading}
        >
          {t("Save Changes")}
        </Button>
      </div>
    </DialogFooter>
  );
}

export function SpecIntegrationHeader({ vendor }: { vendor: SpecIntegrationVendor }) {
  const { theme } = useTheme();
  const logo = theme === "dark" ? (vendor.logoDark ?? vendor.logoLight) : vendor.logoLight;

  return (
    <div className="flex flex-col gap-2">
      <div className="flex items-center justify-center gap-4">
        <LazyImage src={trenovaLogo} className="size-8" />
        <div className="flex items-center justify-center gap-1">
          <div className="bg-muted-foreground size-1 rounded-full" />
          <div className="bg-muted-foreground size-1 rounded-full" />
          <div className="bg-muted-foreground size-1 rounded-full" />
        </div>
        {logo ? (
          <LazyImage
            src={logo}
            alt={`${vendor.name} Logo`}
            className="h-8 max-w-24 object-contain"
          />
        ) : (
          <span className="bg-muted text-foreground/80 flex size-8 items-center justify-center rounded-md text-xs font-semibold">
            {vendor.name.slice(0, 2).toUpperCase()}
          </span>
        )}
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

export function SpecIntegrationPrerequisite({ children }: { children: ReactNode }) {
  return (
    <p className="border-border bg-muted/40 text-muted-foreground rounded-md border p-3 text-xs">
      {children}
    </p>
  );
}
