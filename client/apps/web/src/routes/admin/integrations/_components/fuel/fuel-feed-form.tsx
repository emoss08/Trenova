import { useT } from "@trenova/shared/i18n/use-t";
import type { ConfigFieldSpec } from "@/types/integration";
import { Form, FormGroup } from "@trenova/shared/components/ui/form";
import {
  IntegrationEnabledSwitch,
  SpecIntegrationFields,
  SpecIntegrationFooter,
  SpecIntegrationHeader,
  SpecIntegrationPrerequisite,
  type SpecIntegrationVendor,
} from "../shared/spec-integration-fields";
import { useSpecIntegrationConfig } from "../shared/use-spec-integration-config";

/**
 * The fuel card connections are large forms — an SFTP endpoint, its credentials,
 * and the layout of the file it serves — and the three networks differ only in
 * their copy. Rendering them from the field spec the server already returns keeps
 * the form in step with the Go `ConfigSpecs` on its own, so adding a field to a
 * connector never means editing a form here.
 */

export type FuelFeedVendor = SpecIntegrationVendor;

// The layout field only makes sense for a fixed-width file, so it follows the
// format select rather than sitting there confusing everyone reading a CSV.
const FIXED_WIDTH_ONLY = new Set(["fixedWidthLayout"]);
// Keys whose value is many lines or one very long one, and unreadable in an input.
const MULTILINE_KEYS: ReadonlySet<string> = new Set([
  "fixedWidthLayout",
  "privateKey",
  "knownHostKey",
]);
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
  const t = useT();

  const { configQuery, spec, form, storedByKey, saveMutation, testConnectionMutation } =
    useSpecIntegrationConfig({
      integrationType: vendor.integrationType,
      name: vendor.name,
      open,
    });
  const { control, handleSubmit, watch } = form;

  const fileFormat = watch(`configuration.${FILE_FORMAT_KEY}` as const);
  const authMode = watch(`configuration.${AUTH_MODE_KEY}` as const);

  const visible = spec.filter((field) =>
    isFieldVisible(field, { fileFormat, authMode, hasFormatField: hasKey(spec, FILE_FORMAT_KEY) }),
  );

  return (
    <div className="space-y-4">
      <SpecIntegrationHeader vendor={vendor} />
      {vendor.prerequisite ? (
        <SpecIntegrationPrerequisite>{vendor.prerequisite}</SpecIntegrationPrerequisite>
      ) : null}
      <Form onSubmit={handleSubmit((data) => saveMutation.mutateAsync(data))} className="space-y-4">
        <FormGroup cols={1}>
          <IntegrationEnabledSwitch
            control={control}
            id={`${vendor.integrationType}-enabled`}
            label={t("Enable {0}", vendor.name)}
            description={t("Read transactions on a schedule and post the ones that resolve.")}
          />
          <SpecIntegrationFields
            fields={visible}
            control={control}
            storedByKey={storedByKey}
            multilineKeys={MULTILINE_KEYS}
          />
        </FormGroup>
        <SpecIntegrationFooter
          onCancel={onClose}
          onTestConnection={() => testConnectionMutation.mutateAsync()}
          isTesting={testConnectionMutation.isPending}
          isSaving={saveMutation.isPending}
          isLoading={configQuery.isLoading}
        />
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
